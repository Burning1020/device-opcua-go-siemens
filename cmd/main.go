package main

import (
	"github.com/edgexfoundry/device-opcua-go"
	"github.com/edgexfoundry/device-opcua-go/internal/driver"
	"github.com/edgexfoundry/device-sdk-go/pkg/startup"
)

const (
	serviceName string = "edgex-device-opcua"
)

func main() {
	sd := driver.NewProtocolDriver()
	startup.Bootstrap(serviceName, device_opcua.Version, sd)
}
