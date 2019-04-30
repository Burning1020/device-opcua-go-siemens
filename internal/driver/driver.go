// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2018 Canonical Ltd
// Copyright (C) 2018 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

//
package driver

import (
	"fmt"
	"strings"
	"sync"
	"time"

	ds_models "github.com/edgexfoundry/device-sdk-go/pkg/models"
	"github.com/edgexfoundry/edgex-go/pkg/clients/logging"
	"github.com/edgexfoundry/edgex-go/pkg/models"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

var once sync.Once
var driver *Driver

type Driver struct {
	lc           logger.LoggingClient
	asyncCh      chan<- *ds_models.AsyncValues
	switchButton bool
	Config       *configuration
	CommandResponses map[string]string
}

func NewProtocolDriver() ds_models.ProtocolDriver {
	once.Do(func() {
		driver = new(Driver)
		driver.CommandResponses = make(map[string]string)
	})
	return driver
}

// DisconnectDevice handles protocol-specific cleanup when a device
// is removed.
func (d *Driver) DisconnectDevice(address *models.Addressable) error {
	return nil
}

// Initialize performs protocol-specific initialization for the device
// service.
func (d *Driver) Initialize(lc logger.LoggingClient, asyncCh chan<- *ds_models.AsyncValues) error {
	d.lc = lc
	d.asyncCh = asyncCh
	config, err := LoadConfigFromFile()
	if err != nil {
		panic(fmt.Errorf("Driver.Initialize: Read OPCUA driver configuration failed: %v", err))
	}
	d.Config = config

	go func() {
		err := startIncomingListening()
		if err != nil {
			panic(fmt.Errorf("Driver.Initialize: Start incoming data Listener failed: %v", err))
		}
	}()
	return nil
}

// HandleReadCommands triggers a protocol Read operation for the specified device.
func (d *Driver) HandleReadCommands(addr *models.Addressable, reqs []ds_models.CommandRequest) ([]*ds_models.CommandValue, error) {

	driver.lc.Debug(fmt.Sprintf("Driver.HandleReadCommands: device=%s, operation=%v, attributes=%v", addr.Name, reqs[0].RO.Operation, reqs[0].DeviceObject.Attributes))

	var responses = make([]*ds_models.CommandValue, len(reqs))
	var err error

	// create device client and open connection
	var endpoint = getUrlFromAddressable(addr)
	c := &opcua.Client{EndpointURL: endpoint}
	err = c.Connect()
	if err != nil {
		driver.lc.Warn(fmt.Sprintf("Driver.HandleReadCommands: Failed to create OPCUA client, caused by %s", err))
		return responses, err
	}
	defer c.Close()

	for i, req := range reqs {
		// handle every reqs
		res, err := d.handleReadCommandRequest(c, req, addr)
		if err != nil {
			driver.lc.Info(fmt.Sprintf("Driver.HandleReadCommands: Handle read commands failed: %v", err))
			return responses, err
		}
		responses[i] = res
	}

	return responses, err
}

func (d *Driver) handleReadCommandRequest(deviceClient *opcua.Client, req ds_models.CommandRequest, addr *models.Addressable) (*ds_models.CommandValue, error) {
	var result = &ds_models.CommandValue{}
	var err error

	var nodeID, ok = findNodeidFromObj(addr, req)
	if !ok {
		err = fmt.Errorf("Driver.handleReadCommands: No such nodeId for this type. device=%s, attributes=%v", addr.Name, req.DeviceObject.Attributes)
		return result, err
	}

	// get NewNodeID
	id, err := ua.NewNodeID(nodeID)
	if err != nil {
		driver.lc.Error(fmt.Sprintf("Driver.handleReadCommands: Invalid node id=%s", nodeID))
		return result, err
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
		driver.lc.Error(fmt.Sprintf("Driver.handleReadCommands: Read failed: %s", err))
	}
	if resp.Results[0].Status != ua.StatusOK {
		driver.lc.Error(fmt.Sprintf("Driver.handleReadCommands: Status not OK: %v", resp.Results[0].Status))

	}

	// make new result
	reading := resp.Results[0].Value.Value
	result, err = newResult(req.DeviceObject, req.RO, reading)
	if err != nil {
		driver.lc.Error(fmt.Sprintf("Driver.HandleReadCommands: Reading ignored. DeviceResource=%v", req.RO.Object))
	}

	return result, err
}

// HandleWriteCommands passes a slice of CommandRequest struct each representing
// a ResourceOperation for a specific device resource (aka DeviceObject).
// Since the commands are actuation commands, params provide parameters for the individual
// command.
func (d *Driver) HandleWriteCommands(addr *models.Addressable, reqs []ds_models.CommandRequest,
	params []*ds_models.CommandValue) error {

	if len(reqs) != 1 {
		err := fmt.Errorf("Driver.HandleWriteCommands; too many command requests; only one supported")
		return err
	}
	if len(params) != 1 {
		err := fmt.Errorf("Driver.HandleWriteCommands; the number of parameter is not correct; only one supported")
		return err
	}

	driver.lc.Debug(fmt.Sprintf("Driver.HandleWriteCommands: device: %s, operation: %v, parameters: %v", addr.Name, reqs[0].RO.Operation, params))
	var err error
	if d.switchButton, err = params[0].BoolValue(); err != nil {
		err := fmt.Errorf("Driver.HandleWriteCommands; the data type of parameter should be Boolean, parameter: %s", params[0].String())
		return err
	}

	return nil
}

// Stop the protocol-specific DS code to shutdown gracefully, or
// if the force parameter is 'true', immediately. The driver is responsible
// for closing any in-use channels, including the channel used to send async
// readings (if supported).
func (d *Driver) Stop(force bool) error {
	driver.lc.Debug(fmt.Sprintf("Driver.Stop called: force=%v", force))
	return nil
}

func newResult(deviceObject models.DeviceObject, ro models.ResourceOperation, reading interface{}) (*ds_models.CommandValue, error) {
	var result = &ds_models.CommandValue{}
	var err error
	var resTime = time.Now().UnixNano() / int64(time.Millisecond)

	switch deviceObject.Properties.Value.Type {
	case "Bool":
		result, err = ds_models.NewBoolValue(&ro, resTime, reading.(bool))
	case "String":
		result = ds_models.NewStringValue(&ro, resTime, reading.(string))
	case "Uint8":
		result, err = ds_models.NewUint8Value(&ro, resTime, reading.(uint8))
	case "Uint16":
		result, err = ds_models.NewUint16Value(&ro, resTime, reading.(uint16))
	case "Uint32":
		result, err = ds_models.NewUint32Value(&ro, resTime, reading.(uint32))
	case "Uint64":
		result, err = ds_models.NewUint64Value(&ro, resTime, reading.(uint64))
	case "Int8":
		result, err = ds_models.NewInt8Value(&ro, resTime, reading.(int8))
	case "Int16":
		result, err = ds_models.NewInt16Value(&ro, resTime, reading.(int16))
	case "Int32":
		result, err = ds_models.NewInt32Value(&ro, resTime, reading.(int32))
	case "Int64":
		result, err = ds_models.NewInt64Value(&ro, resTime, reading.(int64))
	case "Float32":
		result, err = ds_models.NewFloat32Value(&ro, resTime, reading.(float32))
	case "Float64":
		result, err = ds_models.NewFloat64Value(&ro, resTime, reading.(float64))
	default:
		err = fmt.Errorf("return result fail, none supported value type: %v", deviceObject.Properties.Value.Type)
	}

	return result, err
}

func getUrlFromAddressable(addr *models.Addressable) string {
	var url string
	if strings.EqualFold(addr.Protocol, "TCP") {
		url = fmt.Sprintf("opc.tcp://")
	} else {
		url = fmt.Sprintf("http://")
	}

	url += fmt.Sprintf("%s:%d%s", addr.Address, addr.Port, addr.Path)
	return url
}

func findNodeidFromObj(addr *models.Addressable, req ds_models.CommandRequest) (string, bool) {
	var nodeID string
	deviceResource := req.RO.Object
	deviceName := addr.Name
	// refers to other search function using for loop
	for _, s := range driver.Config.Servers {
		if deviceName == s.Name {
			for _, n := range s.Nodes {
				if strings.Contains(n.NodeID, deviceResource) {
					nodeID = n.NodeID
					return nodeID, true
				}
			}
		}
	}
	return nodeID, false
}
