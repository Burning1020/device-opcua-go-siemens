package driver

import (
	"context"
	"encoding/json"
	"fmt"
	sdkModel "github.com/edgexfoundry/device-sdk-go/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"github.com/spf13/cast"
	"sync"
	"time"
)

var once 	sync.Once
var driver 	*Driver
var (
	ctx 	context.Context
	cancel  context.CancelFunc
)

type Driver struct {
	Logger           logger.LoggingClient
	AsyncCh          chan<- *sdkModel.AsyncValues
	CommandResponses sync.Map
}

func NewProtocolDriver() sdkModel.ProtocolDriver {
	once.Do(func() {
		driver = new(Driver)
	})
	return driver
}

// Initialize performs protocol-specific initialization for the device
// service.
func (d *Driver) Initialize(lc logger.LoggingClient, asyncCh chan<- *sdkModel.AsyncValues) error {
	ctx, cancel = context.WithCancel(context.Background())
	d.Logger = lc
	d.AsyncCh = asyncCh

	return nil
}

func (d *Driver) DisconnectDevice(deviceName string, protocols map[string]models.ProtocolProperties) error {
	d.Logger.Warn("Driver was disconnected")
	return nil
}

// HandleReadCommands triggers a protocol Read operation for the specified device.
func (d *Driver) HandleReadCommands(deviceName string, protocols map[string]models.ProtocolProperties,
	reqs []sdkModel.CommandRequest) ([]*sdkModel.CommandValue, error) {
	// load Protocol config
	config, nodeMapping, err := CreateConfigurationAndMapping(protocols)
	if err != nil {
		driver.Logger.Error(fmt.Sprintf("error create configuration: %s", err))
		return nil, err
	}
	// create an opcua client and open connection based on config
	client, err := createClient(config)
	if err != nil {
		driver.Logger.Error(fmt.Sprintf("Failed to create OPCUA client: %s", err))
		return nil, err
	}
	defer client.Close()

	responses := make([]*sdkModel.CommandValue, len(reqs))
	for i, req := range reqs {
		nodeId, ok := nodeMapping[req.DeviceResourceName]
		if !ok {
			driver.Logger.Error(fmt.Sprintf("No NodeId found by DeviceResource:%s", req.DeviceResourceName))
			continue
		}
		res, err := d.handleReadCommandRequest(client, req, nodeId)
		if err != nil {
			driver.Logger.Error(fmt.Sprintf("Handle read commands failed: %v", err))
			continue
		}
		responses[i] = res
	}
	return responses, nil
}

func (d *Driver) handleReadCommandRequest(deviceClient *opcua.Client, req sdkModel.CommandRequest, nodeId string) (*sdkModel.CommandValue, error) {
	// get NewNodeID
	id, err := ua.ParseNodeID(nodeId)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("Invalid node id=%s", nodeId))
	}

	// make and execute ReadRequest
	request := &ua.ReadRequest{
		MaxAge: 2000,
		NodesToRead: []*ua.ReadValueID{
			&ua.ReadValueID{NodeID: id},
		},
		TimestampsToReturn: ua.TimestampsToReturnBoth,
	}
	resp, err := deviceClient.Read(request)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("Read failed: %s", err))
	}
	if resp.Results[0].Status != ua.StatusOK {
		return nil, fmt.Errorf(fmt.Sprintf("Status not OK: %v", resp.Results[0].Status))
	}

	// make new result
	reading := resp.Results[0].Value.Value()
	result, err := newResult(req, reading)
	if err != nil {
		return nil, err
	} else {
		driver.Logger.Info(fmt.Sprintf("Get command finished: %v", result))
	}
	return result, nil
}

// HandleWriteCommands passes a slice of CommandRequest struct each representing
// a ResourceOperation for a specific device resource (aka DeviceObject).
// Since the commands are actuation commands, params provide parameters for the individual
// command.
func (d *Driver) HandleWriteCommands(deviceName string, protocols map[string]models.ProtocolProperties,
	reqs []sdkModel.CommandRequest, params []*sdkModel.CommandValue) error {
	// load Protocol config
	config, nodeMapping, err := CreateConfigurationAndMapping(protocols)
	if err != nil {
		driver.Logger.Error(fmt.Sprintf("error create configuration: %s", err))
		return err
	}

	if reqs[0].DeviceResourceName == SubscribeCommandName {
		// first parameter is $SubscribeCommandName means the command is to subscribe nodes
		nodes := make(map[string]bool)
		for i, req := range reqs[1 : ] {
			nodes[req.DeviceResourceName] = convert2TF(req.Type, params[i + 1])
		}
		go startListening(ctx, deviceName, config, nodeMapping, nodes)
		return nil
	}
	// usual command
	// create an opcua client and open connection based on config
	client, err := createClient(config)
	if err != nil {
		driver.Logger.Error(fmt.Sprintf("Failed to create OPCUA client: %s", err))
		return err
	}
	defer client.Close()

	for i, req := range reqs {
		nodeId, ok := nodeMapping[req.DeviceResourceName]
		if !ok {
			return fmt.Errorf(fmt.Sprintf("No NodeId found by DeviceResource:%s", req.DeviceResourceName))
		}
		err := d.handleWriteCommandRequest(client, req, params[i], nodeId)
		if err != nil {
			return fmt.Errorf(fmt.Sprintf("Handle write commands failed: %v", err))
		}
	}
	return nil
}

func (d *Driver) handleWriteCommandRequest(deviceClient *opcua.Client, req sdkModel.CommandRequest,
	param *sdkModel.CommandValue, nodeId string) error {
	// get NewNodeID
	id, err := ua.ParseNodeID(nodeId)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("Invalid node id=%s", nodeId))
	}

	value, err := newCommandValue(req.Type, param)
	if err != nil {
		return err
	}
	v, err := ua.NewVariant(value)

	if err != nil {
		return fmt.Errorf(fmt.Sprintf("Invalid value: %v", err))
	}

	request := &ua.WriteRequest{
		NodesToWrite: []*ua.WriteValue{
			&ua.WriteValue{
				NodeID:      id,
				AttributeID: ua.AttributeIDValue,
				Value: &ua.DataValue{
					EncodingMask: uint8(13),  // encoding mask
					Value:        v,
				},
			},
		},
	}

	resp, err := deviceClient.Write(request)
	if err != nil {
		driver.Logger.Error(fmt.Sprintf("Write value %v failed: %s", v, err))
		return err
	}
	driver.Logger.Info(fmt.Sprintf("Write value %s %s", req.DeviceResourceName, resp.Results[0]))
	return nil
}


// Stop the protocol-specific DS code to shutdown gracefully, or
// if the force parameter is 'true', immediately. The driver is responsible
// for closing any in-use channels, including the channel used to send async
// readings (if supported).
func (d *Driver) Stop(force bool) error {
	d.Logger.Debug("Driver is doing closing jobs...")
	cancel()
	time.Sleep(1 * time.Second)
	return nil
}

// AddDevice is a callback function that is invoked
// when a new Device associated with this Device Service is added
func (d *Driver) AddDevice(deviceName string, protocols map[string]models.ProtocolProperties, adminState models.AdminState) error {
	d.Logger.Debug(fmt.Sprintf("Device %s is updated", deviceName))
	return nil
}

// UpdateDevice is a callback function that is invoked
// when a Device associated with this Device Service is updated
func (d *Driver) UpdateDevice(deviceName string, protocols map[string]models.ProtocolProperties, adminState models.AdminState) error {
	d.Logger.Debug(fmt.Sprintf("Device %s is updated", deviceName))
	return nil
}

// RemoveDevice is a callback function that is invoked
// when a Device associated with this Device Service is removed
func (d *Driver) RemoveDevice(deviceName string, protocols map[string]models.ProtocolProperties) error {
	d.Logger.Debug(fmt.Sprintf("Device %s is updated", deviceName))
	return nil
}

func createClient(config *Configuration) (*opcua.Client, error) {
	endpoint := fmt.Sprintf("%s://%s:%s%s", config.Protocol, config.Address, config.Port, config.Path)
	endpoints, err := opcua.GetEndpoints(endpoint)
	if err != nil {
		return nil, err
	}
	ep := opcua.SelectEndpoint(endpoints, config.Policy, ua.MessageSecurityModeFromString(config.Mode))
	ep.EndpointURL = endpoint // replace
	if ep == nil {
		return nil, fmt.Errorf("failed to find suitable endpoint")
	}
	opts := []opcua.Option{
		opcua.SecurityPolicy(config.Policy),
		opcua.SecurityModeString(config.Mode),
		opcua.CertificateFile(config.CertFile),
		opcua.PrivateKeyFile(config.KeyFile),
		opcua.SessionTimeout(30 * time.Minute),
		opcua.AuthAnonymous(),
		opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeAnonymous),
	}
	client := opcua.NewClient(ep.EndpointURL, opts...)
	if err := client.Connect(ctx); err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("Failed to create OPCUA client, %s", err))
	}
	return client, nil
}


func createNodeMapping(mappingStr string) (map[string]string, error) {
	var mapping map[string]string
	b := []byte(mappingStr)
	if err := json.Unmarshal(b, &mapping); err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("Umarshal failed: %s", err))
	}
	return mapping, nil
}

func newResult(req sdkModel.CommandRequest, reading interface{}) (*sdkModel.CommandValue, error) {
	var result = &sdkModel.CommandValue{}
	var err error
	var resTime = time.Now().UnixNano() / int64(time.Millisecond)
	castError := "fail to parse %v reading, %v"

	if !checkValueInRange(req.Type, reading) {
		err = fmt.Errorf("parse reading fail. Reading %v is out of the value type(%v)'s range", reading, req.Type)
		driver.Logger.Error(err.Error())
		return result, err
	}

	switch req.Type {
	case sdkModel.Bool:
		val, err := cast.ToBoolE(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, req.DeviceResourceName, err)
		}
		result, err = sdkModel.NewBoolValue(req.DeviceResourceName, resTime, val)
	case sdkModel.String:
		val, err := cast.ToStringE(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, req.DeviceResourceName, err)
		}
		result = sdkModel.NewStringValue(req.DeviceResourceName, resTime, val)
	case sdkModel.Uint8:
		val, err := cast.ToUint8E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, req.DeviceResourceName, err)
		}
		result, err = sdkModel.NewUint8Value(req.DeviceResourceName, resTime, val)
	case sdkModel.Uint16:
		val, err := cast.ToUint16E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, req.DeviceResourceName, err)
		}
		result, err = sdkModel.NewUint16Value(req.DeviceResourceName, resTime, val)
	case sdkModel.Uint32:
		val, err := cast.ToUint32E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, req.DeviceResourceName, err)
		}
		result, err = sdkModel.NewUint32Value(req.DeviceResourceName, resTime, val)
	case sdkModel.Uint64:
		val, err := cast.ToUint64E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, req.DeviceResourceName, err)
		}
		result, err = sdkModel.NewUint64Value(req.DeviceResourceName, resTime, val)
	case sdkModel.Int8:
		val, err := cast.ToInt8E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, req.DeviceResourceName, err)
		}
		result, err = sdkModel.NewInt8Value(req.DeviceResourceName, resTime, val)
	case sdkModel.Int16:
		val, err := cast.ToInt16E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, req.DeviceResourceName, err)
		}
		result, err = sdkModel.NewInt16Value(req.DeviceResourceName, resTime, val)
	case sdkModel.Int32:
		val, err := cast.ToInt32E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, req.DeviceResourceName, err)
		}
		result, err = sdkModel.NewInt32Value(req.DeviceResourceName, resTime, val)
	case sdkModel.Int64:
		val, err := cast.ToInt64E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, req.DeviceResourceName, err)
		}
		result, err = sdkModel.NewInt64Value(req.DeviceResourceName, resTime, val)
	case sdkModel.Float32:
		val, err := cast.ToFloat32E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, req.DeviceResourceName, err)
		}
		result, err = sdkModel.NewFloat32Value(req.DeviceResourceName, resTime, val)
	case sdkModel.Float64:
		val, err := cast.ToFloat64E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, req.DeviceResourceName, err)
		}
		result, err = sdkModel.NewFloat64Value(req.DeviceResourceName, resTime, val)
	default:
		err = fmt.Errorf("return result fail, none supported value type: %v", req.Type)
	}

	return result, err
}


func convert2TF(valueType sdkModel.ValueType, param *sdkModel.CommandValue) bool {
	var TF bool
	value, err := newCommandValue(valueType, param)
	if err != nil {
		return false
	}
	switch valueType {
	case sdkModel.Bool:
		TF = value.(bool)
	case sdkModel.String:
		TF = value.(string) == "on"
	case sdkModel.Uint8:
		TF = value.(uint8) == uint8(1)
	case sdkModel.Uint16:
		TF = value.(uint16) == uint16(1)
	case sdkModel.Uint32:
		TF = value.(uint32) == uint32(1)
	case sdkModel.Uint64:
		TF = value.(uint64) == uint64(1)
	case sdkModel.Int8:
		TF = value.(int8) == int8(1)
	case sdkModel.Int16:
		TF = value.(int16) == int16(1)
	case sdkModel.Int32:
		TF = value.(int32) == int32(1)
	case sdkModel.Int64:
		TF = value.(int64) == int64(1)
	case sdkModel.Float32:
		TF = value.(float32) == float32(1)
	case sdkModel.Float64:
		TF = value.(float64) == float64(1)
	}
	return TF
}

func newCommandValue(valueType sdkModel.ValueType, param *sdkModel.CommandValue) (interface{}, error) {
	var commandValue interface{}
	var err error
	switch valueType {
	case sdkModel.Bool:
		commandValue, err = param.BoolValue()
	case sdkModel.String:
		commandValue, err = param.StringValue()
	case sdkModel.Uint8:
		commandValue, err = param.Uint8Value()
	case sdkModel.Uint16:
		commandValue, err = param.Uint16Value()
	case sdkModel.Uint32:
		commandValue, err = param.Uint32Value()
	case sdkModel.Uint64:
		commandValue, err = param.Uint64Value()
	case sdkModel.Int8:
		commandValue, err = param.Int8Value()
	case sdkModel.Int16:
		commandValue, err = param.Int16Value()
	case sdkModel.Int32:
		commandValue, err = param.Int32Value()
	case sdkModel.Int64:
		commandValue, err = param.Int64Value()
	case sdkModel.Float32:
		commandValue, err = param.Float32Value()
	case sdkModel.Float64:
		commandValue, err = param.Float64Value()
	default:
		err = fmt.Errorf("fail to convert param, none supported value type: %v", valueType)
	}

	return commandValue, err
}