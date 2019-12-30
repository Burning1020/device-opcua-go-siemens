//
package driver

import (
	"context"
	"fmt"
	sdk "github.com/edgexfoundry/device-sdk-go"
	sdkModel "github.com/edgexfoundry/device-sdk-go/pkg/models"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/monitor"
	"time"
)

const (
	MassageChanCap  = 16
 	ReadingArrLen	  = 100
 	WaittingDur		=  1 * time.Second
)

var service 	*sdk.Service
var cmsMap 		map[string]*CMS

// CMS is a group of opcua_client, node monitor, subscription related and opcua_nodes
type CMS struct {
	client 		*opcua.Client
	monitor 	*monitor.NodeMonitor
	sub  		*monitor.Subscription
	nodes		map[string]bool
}
func init()  {
	cmsMap = make(map[string]*CMS)
}

func startListening(ctx context.Context, deviceName string, config *Configuration, nodeMapping map[string]string, nodes map[string]bool) {
	// reverse nodeMapping, bind nodeId with node
	for node, Id := range nodeMapping {
		nodeMapping[Id] = node
	}

	cms, exist := cmsMap[deviceName]
	if exist {
		// nodeIds array contains the node ids that need to subscribe
		var toAdd, toRemove []string
		for node := range nodes{
			if nodes[node] && !cms.nodes[node] {
				toAdd = append(toAdd, nodeMapping[node])
			} else if !nodes[node] && cms.nodes[node] {
				toRemove = append(toRemove, nodeMapping[node])
			}
		}
		_ = cms.sub.AddNodes(toAdd...)
		_ = cms.sub.RemoveNodes(toRemove...)
		cms.nodes = nodes
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
	for node := range nodes {
		if nodes[node] {
			nodeIds = append(nodeIds, nodeMapping[node])
		}
	}
	ch := make(chan *monitor.DataChangeMessage, MassageChanCap)
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

	cvs := make([]*sdkModel.CommandValue, 0, ReadingArrLen)
	//for i := 0; i < 3; i++ {		go func() {
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
					cv := toCommandValue(msg.Value.Value(), deviceName, deviceResource)
					cvs = append(cvs, cv)
					//if len(cvs) >= ReadingArrLen{
					//	sentToAsynCh(cvs, deviceName)
					//	cvs = cvs[:0]
					//}
					time.Sleep(0)
				case <- time.After(WaittingDur):
					if len(cvs) > 0 {
						sentToAsynCh(cvs, deviceName)
						cvs = cvs[:0]
					}
				}
			}
		//}()	}
	select {}
}

func toCommandValue(data interface{}, deviceName string, deviceResource string) *sdkModel.CommandValue {
	driver.Logger.Info(fmt.Sprintf("[Incoming listener] Incoming reading received: name=%v deviceResource=%v value=%v", deviceName, deviceResource, data))
	deviceObject, ok := service.DeviceResource(deviceName, deviceResource, "get")
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

func sentToAsynCh(cvs []*sdkModel.CommandValue, deviceName string)  {
	asyncValues := &sdkModel.AsyncValues{
		DeviceName:    deviceName,
		CommandValues: cvs,
	}
	driver.AsyncCh <- asyncValues
}

