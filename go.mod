module github.com/edgexfoundry/device-opcua-go

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/edgexfoundry/device-sdk-go v0.0.0-20190111001241-58ceab4ca78d
	github.com/edgexfoundry/edgex-go v0.0.0-20190429145530-9c4e7bdd85c1
	github.com/globalsign/mgo v0.0.0-20181015135952-eeefdecb41b8 // indirect
	github.com/go-kit/kit v0.8.0 // indirect
	github.com/go-logfmt/logfmt v0.4.0 // indirect
	github.com/gopcua/opcua v0.0.0-20190331100126-58f31bc4f4a2
	github.com/gorilla/mux v1.7.1 // indirect
	github.com/hashicorp/consul/api v1.0.1 // indirect
	github.com/robfig/cron v0.0.0-20180505203441-b41be1df6967 // indirect
	gopkg.in/mgo.v2 v2.0.0-20180705113604-9856a29383ce // indirect
	gopkg.in/yaml.v2 v2.2.2 // indirect
)

replace golang.org/x/crypto v0.0.0-20181029021203-45a5f77698d3 => github.com/golang/crypto v0.0.0-20181029021203-45a5f77698d3

replace (
	golang.org/x/net v0.0.0-20181023162649-9b4f9f5ad519 => github.com/golang/net v0.0.0-20181023162649-9b4f9f5ad519
	golang.org/x/net v0.0.0-20181201002055-351d144fa1fc => github.com/golang/net v0.0.0-20181201002055-351d144fa1fc
)

replace golang.org/x/sync v0.0.0-20181221193216-37e7f081c4d4 => github.com/golang/sync v0.0.0-20181221193216-37e7f081c4d4

replace (
	golang.org/x/sys v0.0.0-20180823144017-11551d06cbcc => github.com/golang/sys v0.0.0-20180823144017-11551d06cbcc
	golang.org/x/sys v0.0.0-20181026203630-95b1ffbd15a5 => github.com/golang/sys v0.0.0-20181026203630-95b1ffbd15a5
)
