//
package driver

import (
	"context"
	"encoding/json"
	"fmt"
	sdk "github.com/edgexfoundry/device-sdk-go"
	sdkModel "github.com/edgexfoundry/device-sdk-go/pkg/models"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/monitor"
	"io/ioutil"
	"time"
)

const (
	DataPath			= "subscriptionData.json"   // the path of subscription data
	MassageChanCap  	= 16						// the capacity of massage chanel
	ReadingArrLen		= 100						// the capacity of reading length
	WaitingDuration 	=  1000 * time.Millisecond			// time duration of sent a event
)

// CMS is a group of opcua_client, node monitor, subscription related, opcua_nodes and cancel func.
type CMS struct {
	client  	*opcua.Client
	monitor 	*monitor.NodeMonitor
	sub     	*monitor.Subscription
	nodes   	map[string]bool   		// key-value struct of valueDescriptor name and subscribe state
	cancel      context.CancelFunc		// callback cancel function when stop subscription
}

// start listening for data change massage
func startListening(deviceName string, config *Configuration, nodeMapping map[string]string, nodes map[string]bool) {
	subCtx, cancel := context.WithCancel(ctx)
	cms, exist := cmsMap[deviceName]
	if exist {
		var toAdd, toRemove []string  // toAdd/toRemove represents new nodes to subscribe and old nodes to unsubscribe
		for node := range nodes{
			if nodes[node] && !cms.nodes[node] {
				toAdd = append(toAdd, nodeMapping[node])
			} else if !nodes[node] && cms.nodes[node] {
				toRemove = append(toRemove, nodeMapping[node])
			}
		}
		_ = cms.sub.AddNodes(toAdd...)
		_ = cms.sub.RemoveNodes(toRemove...)
		cms.nodes = nodes	// update cms when changed

		// we wish user always want to stop subscription, so let stop=true.
		// if any of node's state is true, which means user want to subscribe one node at least, so let stop=false.
		// if stop=true still, stop the subscription and delete CMS.
		stop := true
		for _, state := range cms.nodes {
			if state {
				stop = false
				break
			}
		}
		if stop {
			cms.cancel()
			delete(cmsMap, deviceName)
		}
		saveSubState()  // save to file
		return
	}

	// nodeIds array contains the node ids that need to subscribe
	nodeIds := make([]string, 0)
	for node := range nodes {
		if nodes[node] {
			nodeIds = append(nodeIds, nodeMapping[node])
		}
	}
	if len(nodeIds) < 1 {  // all node state is off
		return
	}
	wg.Add(1)  // wg is a WaitingGroup waiting for clean up work finished
	defer wg.Done()
	// create an opcua client and open connection based on config
	c, err := createClient(config)
	if err != nil {
		driver.Logger.Error(fmt.Sprintf("failed to create OPCUA c: %s", err))
		return
	}
	defer c.Close()
	// create node Monitor
	nodeMonitor, _ := monitor.NewNodeMonitor(c)

	/**
	 * @deprecated. handle function when data change massage was dropped
	 * nodeMonitor.SetErrorHandler(func(_ *opcua.Client, sub *monitor.Subscription, err error) {
	 * 	//driver.Logger.Error(fmt.Sprintf("error when subscribe device=%s : err=%s", deviceName, err.Error()))
	 * })
	 */
	// make a channel for data change
	notifyCh := make(chan *monitor.DataChangeMessage, MassageChanCap)
	defer close(notifyCh)

	sub, _ := nodeMonitor.ChanSubscribe(subCtx, notifyCh, nodeIds...)
	defer sub.Unsubscribe()
	cmsMap[deviceName] = &CMS{
		client: 	c,
		monitor:	nodeMonitor,
		sub:    	sub,
		nodes:    	nodes,
		cancel:		cancel,
	}
	saveSubState()  // save to file
	driver.Logger.Info(fmt.Sprintf("start subscribe device=%s", deviceName))

	// reverse nodeMapping, bind nodeId with node
	for node, Id := range nodeMapping {
		nodeMapping[Id] = node
	}
	cvs := make([]*sdkModel.CommandValue, 0, ReadingArrLen)
	ticker := time.NewTicker(WaitingDuration)
	defer ticker.Stop()

	for {
		select {
		case <- subCtx.Done():
			// cancel fun was called then ctx was done
			return
		case msg := <-notifyCh:
			deviceResource := nodeMapping[msg.NodeID.String()]
			cv := toCommandValue(msg.Value.Value(), deviceName, deviceResource)
			cvs = append(cvs, cv)
			if len(cvs) >= ReadingArrLen {
				sentToAsynCh(cvs, deviceName)
				cvs = make([]*sdkModel.CommandValue, 0, ReadingArrLen)
			}
		case <- ticker.C:
			if len(cvs) > 0 {
				sentToAsynCh(cvs, deviceName)
				cvs = cvs[:0]
				cvs = make([]*sdkModel.CommandValue, 0, ReadingArrLen)
			}
		}
	}
}

func toCommandValue(data interface{}, deviceName string, deviceResource string) *sdkModel.CommandValue {
	//driver.Logger.Info(fmt.Sprintf("[Incoming listener] Incoming reading received: name=%v deviceResource=%v value=%v", deviceName, deviceResource, data))
	deviceObject, ok := sdk.RunningService().DeviceResource(deviceName, deviceResource, "get")
	if !ok {
		driver.Logger.Warn(fmt.Sprintf("[Incoming listener] Incoming reading ignored. No DeviceObject found: name=%v deviceResource=%v value=%v", deviceName, deviceResource, data))
		return nil
	}

	req := sdkModel.CommandRequest{
		DeviceResourceName: deviceResource,
		Type:               sdkModel.ParseValueType(deviceObject.Properties.Value.Type),
	}

	result, err := newResult(req, data)
	if err != nil {
		driver.Logger.Warn(fmt.Sprintf("[Incoming listener] Incoming reading ignored. name=%v deviceResource=%v value=%v", deviceName, deviceResource, data))
		return nil
	}
	return result
}

// sent event to asynchronous channel
func sentToAsynCh(cvs []*sdkModel.CommandValue, deviceName string)  {
	asyncValues := &sdkModel.AsyncValues{
		DeviceName:    deviceName,
		CommandValues: cvs,
	}
	driver.AsyncCh <- asyncValues
}

func saveSubState() {
	subState := make(map[string]map[string]bool)
	for deviceName, cms := range cmsMap {
		subState[deviceName] = cms.nodes
	}
	jsonStr, err := json.MarshalIndent(subState, "", "    ")
	if err != nil {
		driver.Logger.Error(fmt.Sprintf("failed to marsh node state: %s", err))
		return
	}
	if err = ioutil.WriteFile(DataPath, jsonStr, 0771); err != nil {
		driver.Logger.Error(fmt.Sprintf("failed to write %s: %s", DataPath, err))
	}
}

func loadSubState() {
	subState := make(map[string]map[string]bool)
	b, err := ioutil.ReadFile(DataPath)
	if err != nil {
		driver.Logger.Error(fmt.Sprintf("failed to read %s: %s", DataPath, err))
		return
	}
	if err = json.Unmarshal(b, &subState); err != nil {
		driver.Logger.Error(fmt.Sprintf("failed to unmarshal data: %s", err))
	}
	for deviceName, nodes := range subState {
		device, _ := sdk.RunningService().GetDeviceByName(deviceName)
		config, nodeMapping, _ := CreateConfigurationAndMapping(device.Protocols)
		go startListening(deviceName, config, nodeMapping, nodes)
	}
}