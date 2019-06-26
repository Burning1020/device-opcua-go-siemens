//
package driver

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/edgexfoundry/device-sdk-go"
	sdkModel "github.com/edgexfoundry/device-sdk-go/pkg/models"
	"github.com/edgexfoundry/edgex-go/pkg/models"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

var service *sdk.Service

func startIncomingListening() error {

	var (
		devicename = driver.Config.IncomingDataServer.DeviceName
		policy   = driver.Config.IncomingDataServer.Policy
		mode     = driver.Config.IncomingDataServer.Mode
		certFile = driver.Config.IncomingDataServer.CertFile
		keyFile  = driver.Config.IncomingDataServer.KeyFile
		nodeID   = driver.Config.IncomingDataServer.NodeID
	)

	service = sdk.RunningService()
	addr, err := findDevice(service.Devices(), devicename)
	if err != nil {
		return err
	}
	var endpoint = getUrlFromAddressable(addr)

	ctx := context.Background()
	endpoints, err := opcua.GetEndpoints(endpoint)
	if err != nil {
		return err
	}
	ep := opcua.SelectEndpoint(endpoints, policy, ua.MessageSecurityModeFromString(mode))
	// replace Burning-Laptop with ip adress
	ep.EndpointURL = endpoint
	if ep == nil {
		return fmt.Errorf("Failed to find suitable endpoint")
	}

	opts := []opcua.Option{
		opcua.SecurityPolicy(policy),
		opcua.SecurityModeString(mode),
		opcua.CertificateFile(certFile),
		opcua.PrivateKeyFile(keyFile),
		opcua.AuthAnonymous(),
		opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeAnonymous),
	}

	c := opcua.NewClient(ep.EndpointURL, opts...)
	if err := c.Connect(ctx); err != nil {
		return err
	}
	defer c.Close()

	sub, err := c.Subscribe(&opcua.SubscriptionParameters{
		Interval: 500 * time.Millisecond,
	})
	if err != nil {
		return err
	}
	defer sub.Cancel()

	id, err := ua.ParseNodeID(nodeID)
	if err != nil {
		return err
	}

	// arbitrary client handle for the monitoring item
	handle := uint32(1) // useless
	miCreateRequest := opcua.NewMonitoredItemCreateRequestWithDefaults(id, ua.AttributeIDValue, handle)
	res, err := sub.Monitor(ua.TimestampsToReturnBoth, miCreateRequest)
	if err != nil || res.Results[0].StatusCode != ua.StatusOK {
		return err
	}

	go sub.Run(ctx) // start Publish loop

	// read from subscription's notification channel until ctx is cancelled
	for {
		select {
		case <-ctx.Done():
			return nil
		case res := <-sub.Notifs:
			if res.Error != nil {
				driver.lc.Debug(fmt.Sprintf("%s", res.Error))
				continue
			}

			switch x := res.Value.(type) {
			case *ua.DataChangeNotification:
				for _, item := range x.MonitoredItems {
					data := item.Value.Value.Value
					onIncomingDataReceived(data)
				}
			}
		}
	}
}

func onIncomingDataReceived(data interface{}) {
	deviceName := driver.Config.IncomingDataServer.DeviceName
	cmd := driver.Config.IncomingDataServer.NodeID
	reading := data



	deviceObject, ok := service.DeviceObject(deviceName, cmd, "get")
	if !ok {
		driver.lc.Warn(fmt.Sprintf("[Incoming listener] Incoming reading ignored. No DeviceObject found: name=%v deviceResource=%v value=%v", deviceName, cmd, data))
		return
	}

	ro, ok := service.ResourceOperation(deviceName, cmd, "get")
	if !ok {
		driver.lc.Warn(fmt.Sprintf("[Incoming listener] Incoming reading ignored. No ResourceOperation found: name=%v deviceResource=%v value=%v", deviceName, cmd, data))
		return
	}

	result, err := newResult(deviceObject, ro, reading)

	if err != nil {
		driver.lc.Warn(fmt.Sprintf("[Incoming listener] Incoming reading ignored. name=%v deviceResource=%v value=%v", deviceName, cmd, data))
		return
	}

	asyncValues := &sdkModel.AsyncValues{
		DeviceName:    deviceName,
		CommandValues: []*sdkModel.CommandValue{result},
	}

	driver.lc.Info(fmt.Sprintf("[Incoming listener] Incoming reading received: name=%v deviceResource=%v value=%v", deviceName, cmd, data))

	driver.asyncCh <- asyncValues

}

// search for device that match devicename
func findDevice(devices []models.Device, devicename string) (models.Addressable, error) {
	var addr models.Addressable
	for _, device := range devices {
		if device.Name == devicename {
			addr = device.Addressable
			return addr, nil
		}
	}
	return addr, fmt.Errorf("[Incoming listener] No such device, name=%s", devicename)
}
