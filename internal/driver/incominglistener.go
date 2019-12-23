//
package driver

import (
	"context"
	"fmt"
	sdk "github.com/edgexfoundry/device-sdk-go"
	sdkModel "github.com/edgexfoundry/device-sdk-go/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/monitor"
	"time"
)

var service 	*sdk.Service
var cmsMap 		map[string]*CMS

type CMS struct {
	client 		*opcua.Client
	monitor 	*monitor.NodeMonitor
	sub  		*monitor.Subscription
	nodes		map[string]bool
}
func init()  {
	cmsMap = make(map[string]*CMS)
}

func startListening(ctx context.Context, deviceName string, protocols map[string]models.ProtocolProperties,
	reqs []sdkModel.CommandRequest, params []*sdkModel.CommandValue) {

	// create Protocol config and the mapping rule between DeviceResource and NodeId
	config, nodeMapping, err := CreateConfigurationAndMapping(protocols)
	if err != nil {
		driver.Logger.Error(fmt.Sprintf("error create configuration: %s", err))
		return
	}
	// reverse nodeMapping
	for k, v := range nodeMapping {
		nodeMapping[v] = k
	}

	cms, exist := cmsMap[deviceName]
	if exist {
		// nodeIds array contains the node ids that need to subscribe
		var toAdd, toRemove []string
		newNodes := make(map[string]bool)
		for i, req := range reqs[1 : ] {
			vd := req.DeviceResourceName
			newNodes[vd] = convert2TF(req.Type, params[i + 1])
			nodeId, ok := nodeMapping[vd]
			if !ok {
				driver.Logger.Error(fmt.Sprintf("No NodeId found by DeviceResource:%s", vd))
				return
			}
			if newNodes[vd] && !cms.nodes[vd] {
				toAdd = append(toAdd, nodeId)
			} else if !newNodes[vd] && cms.nodes[vd] {
				toRemove = append(toRemove, nodeId)
			}
		}
		cms.nodes = newNodes
		_ = cms.sub.AddNodes(toAdd...)
		_ = cms.sub.RemoveNodes(toRemove...)
		return
	}

	// create an opcua client and open connection based on config
	c, err := createClient(config)
	if err != nil {
		driver.Logger.Error(fmt.Sprintf("failed to create OPCUA c: %s", err))
		return
	}
	defer c.Close()
	nodeMonitor, err := monitor.NewNodeMonitor(c)
	if err != nil {
		driver.Logger.Error(fmt.Sprintf("failed to create Node Monitor: %s", err))
		return
	}
	nodeMonitor.SetErrorHandler(func(_ *opcua.Client, sub *monitor.Subscription, err error) {
		driver.Logger.Error(fmt.Sprintf("error in device=%s : sub=%d err=%s", deviceName, sub.SubscriptionID(), err.Error()))
	})

	// nodeIds array contains the node ids that need to subscribe
	var nodeIds []string
	nodes :=  make(map[string]bool)
	for i, req := range reqs[1 : ] {
		vd := req.DeviceResourceName
		nodes[vd] = convert2TF(req.Type, params[i + 1])
		if nodes[vd] {
			nodeId, ok := nodeMapping[vd]
			if !ok {
				driver.Logger.Error(fmt.Sprintf("No NodeId found by DeviceResource:%s", vd))
				return
			}
			nodeIds = append(nodeIds, nodeId)
		}
	}
	ch := make(chan *monitor.DataChangeMessage, 16)
	sub, err := nodeMonitor.ChanSubscribe(ctx, ch, nodeIds...)
	defer sub.Unsubscribe()
	if err != nil {
		driver.Logger.Error(fmt.Sprintf("failed to create subscription: %s", err))
		return
	}
	cmsMap[deviceName] = &CMS{
		client: 	c,
		monitor:	nodeMonitor,
		sub:    	sub,
		nodes:    	nodes,
	}

	driver.Logger.Info(fmt.Sprintf("start subscribe device=%s", deviceName))
	for {
		select {
		case <- ctx.Done():
			return
		case msg := <- ch:
			if msg.Error != nil {
				driver.Logger.Error(fmt.Sprintf("device=%s error=%s", deviceName, msg.Error))
				return
			}
			deviceResource := nodeMapping[msg.NodeID.String()]
			onIncomingDataReceived(msg.Value.Value(), deviceName, deviceResource)
			time.Sleep(0)
		}
	}

}

func onIncomingDataReceived(data interface{}, deviceName string, deviceResource string) {
	deviceObject, ok := service.DeviceResource(deviceName, deviceResource, "get")
	if !ok {
		driver.Logger.Warn(fmt.Sprintf("[Incoming listener] Incoming reading ignored. No DeviceObject found: name=%v deviceResource=%v value=%v", deviceName, deviceResource, data))
		return
	}

	req := sdkModel.CommandRequest{
		DeviceResourceName: deviceResource,
		Type:               sdkModel.ParseValueType(deviceObject.Properties.Value.Type),
	}

	result, err := newResult(req, data)
	if err != nil {
		driver.Logger.Warn(fmt.Sprintf("[Incoming listener] Incoming reading ignored. name=%v deviceResource=%v value=%v", deviceName, deviceResource, data))
		return
	}

	asyncValues := &sdkModel.AsyncValues{
		DeviceName:    deviceName,
		CommandValues: []*sdkModel.CommandValue{result},
	}
	driver.Logger.Info(fmt.Sprintf("[Incoming listener] Incoming reading received: name=%v deviceResource=%v value=%v", deviceName, deviceResource, data))
	driver.AsyncCh <- asyncValues
}
