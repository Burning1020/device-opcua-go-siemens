# OPC-UA Device Service

## Overview
This repository is a Go-based EdgeX Foundry Device Service which uses OPC-UA protocol to interact with the devices or IoT objects.

## 

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

### Device list
Define devices info for device-sdk to auto upload device profile and create device instance. Please modify `configuration.toml` file which under `./cmd/res` folder
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

## Installation and Execution
```bash
make build
make run
```
## Reference

* EdgeXFoundry documentation: https://docs.edgexfoundry.org/
* EdgeX Foundry Services: https://github.com/edgexfoundry/edgex-go
* Go OPCUA library: https://github.com/gopcua/opcua
* OPCUA Server: https://www.prosysopc.com/products/opc-ua-simulation-server