# OPC-UA Device Service

## Overview
This repository is a Go-based EdgeX Foundry Device Service which uses OPC-UA protocol to interact with the devices or IoT objects.

Read [README_CN.md](./README_CN.md) for Chinese version.

## Features
1. Subscribe device node
2. Execute read command
2. Execute write command

## Prerequisite
* MongoDB / Redis
* Edgex-go: core data, core command, core metadata
* OPCUA Server

## Predefined configuration

### Device Profile
A Device Profile can be thought of as a template of a type or classification of Device. 

Write device profile for your own devices, define deviceResources, deviceCommands and coreCommands. Please refer to `cmd/res/OpcuaServer.yaml`

Note: device profile must contains a "SubMark" string Value Descriptor and a "Subscribe" **SET** command if want to subscribe device node. 
And write **mappings** property. "SubMark" is used to distinguish Subscribe and other command.

### Pre-define Devices
Define devices for device-sdk to auto upload device profile and create device instance. Please modify `configuration.toml` file which under `./cmd/res` folder.

```toml
# Pre-define Devices
[DeviceList.Protocols]
      [DeviceList.Protocols.opcua]
          Protocol = "opc.tcp"
          Address = "192.168.3.165"
          Port = "53530"
          Path = "/OPCUA/SimulationServer"
          MappingStr = "{ \"Counter\": \"ns=5;s=Counter1\", \"Random\": \"ns=5;s=Random1\" }"
          Policy = "None"
          Mode = "None"
          CertFile = ""
          KeyFile = ""
```

**Protocol**, **Policy**, **Mode**, **CertFile** and **KeyFile** properties are not necessary, they all have default value as mentioned above.

Note: **MappingStr** property is JSON format and needs escape characters.

## Installation and Execution
```bash
make build
make run
make docker
```

## Subscribe device node
Trigger a Subscribe command through these methods:

- Edgex UI client. Sigh in -> Add Gateway and select it -> Select one DeviceService and click its Devices button -> 
Click target device's Commands button -> Select "Subscribe" set Method -> Ignore "SubMark" and fill the blank near ValueDescriptor, 
use "on" or "off" to represent subscribe this node or not.

- Any HTTP Client like [PostMan](https://www.getpostman.com/). Use core command API to exec "subscribe" command. 

## Reference
* EdgeX Foundry Services: https://github.com/edgexfoundry/edgex-go
* Go OPCUA library: https://github.com/gopcua/opcua
* OPCUA Server: https://www.prosysopc.com/products/opc-ua-simulation-server

## Buy me a cup of coffee
If you like this repository, star it and encourage me.