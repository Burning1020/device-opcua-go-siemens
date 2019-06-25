# OPC-UA Device Service

## Overview
This repository is a Go-based EdgeX Foundry Device Service which uses OPC-UA protocol to interact with the devices or IoT objects.
For more details, please refer to [README_CN.md](https://github.com/Burning1020/device-opcua-go/blob/master/README_CN.md)

## Feature

1. Subscribe data from OPCUA endpoint
2. Execute read command according to schedule event

## Prerequisite
* MongoDB
* Edgex-go
* OPCUA Server

## Predefined configuration

### Servers and Nodes
Modify `configuration-driver.toml` file which under `./cmd/res` folder if needed
```toml
# Schedule event configutation
[[Servers]]
  Name = "SimulationServer"
  [[Servers.Nodes]]
    NodeID = "ns=5;s=Counter1"
    DeviceResource = "Counter1"
  [[Servers.Nodes]]
    NodeID = "ns=5;s=Random1"
    DeviceResource = "Random1"

# Subscribe configuration
[IncomingDataServer]
  Endpoint = "opc.tcp://192.168.3.170:53530/OPCUA/SimulationServer"  # OPC UA Endpoint URL
  Policy = "None"               # Security policy: None, Basic128Rsa15, Basic256, Basic256Sha256. Default: auto
  Mode = "None"                 # Security mode: None, Sign, SignAndEncrypt. Default: auto
  CertFile = ""                 # Path to cert.pem. Required for security mode/policy != None
  KeyFile = ""                  # Path to private key.pem. Required for security mode/policy != None
  NodeID = "ns=5;s=Counter1"    # Node id to subscribe to
  Name = "SimulationServer"     #
  DeviceResource = "Counter1"   # Device Resource relevant to endpoint value
```

### Pre-define Devices
Define devices for device-sdk to auto upload device profile and create device instance. Please modify `configuration.toml` file which under `./cmd/res` folder
```toml
# Pre-define Devices
[[DeviceList]]
  Name = "SimulationServer"
  Profile = "OPCUA-Server"
  Description = "OPCUA device is created for test purpose"
  Labels = [ "test" ]
  [DeviceList.Addressable]
    Address = ""
    Port = 53530
    Protocol = "TCP"
    Path = "/OPCUA/SimulationServer"
```

### Pre-define Schedules and ScheduleEvents
Define schedules and schedule events for core-command to auto exec command periodically. Please modify `configuration.toml` file
```toml
# Pre-define Schedule Configuration
[[Schedules]]
Name = "5sec-schedule"
Frequency = "PT5S"

[[ScheduleEvents]]
Name = "readCounter"
Schedule = "5sec-schedule"
  [ScheduleEvents.Addressable]
  HTTPMethod = "GET"
  Path = "/api/v1/device/name/SimulationServer/GetCountNum"
```

## Installation and Execution
```bash
make build
make run
```

## Reference
* EdgeX Foundry Services: https://github.com/edgexfoundry/edgex-go
* Go OPCUA library: https://github.com/gopcua/opcua
* OPCUA Server: https://www.prosysopc.com/products/opc-ua-simulation-server

## Buy me a cup of coffee
If you like this project, please star it to make encouragements.