# OPC-UA Device Service

## Overview
This repository is a Go-based EdgeX Foundry Device Service which uses OPC-UA protocol to interact with the devices or IoT objects.
For more details, please refer to [README_CN.md](https://github.com/Burning1020/device-opcua-go/blob/master/README_CN.md)

## Prerequisite
* MongoDB
* Edgex-go
* OPCUA Server

## Predefined configuration

### Servers and Nodes
Modify `configuration-driver.toml` file which under `./cmd/res` folder
```toml
[[Servers]]
  Name = "SimulationServer"
  [[Servers.Nodes]]
    NodeID = "ns=5;s=Counter1"
  [[Servers.Nodes]]
    NodeID = "ns=5;s=Random1"
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
    Address = "Burning-Laptop"
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