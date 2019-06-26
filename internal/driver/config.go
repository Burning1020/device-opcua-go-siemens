// set OPCUA Server Nodes configuration
package driver

import (
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/BurntSushi/toml"
)

type configuration struct {
	IncomingDataServer 	SubscribeInfo
}

type SubscribeInfo struct {
	Endpoint		string
	Policy 			string
	Mode  			string
	CertFile	 	string
	KeyFile 		string
	NodeID 			string
	DeviceName		string
}

// LoadConfigFromFile use to load toml configuration
func LoadConfigFromFile() (*configuration, error) {

	config := new(configuration)

	confDir := flag.Lookup("confdir").Value.(flag.Getter).Get().(string)
	if len(confDir) == 0 {
		confDir = flag.Lookup("c").Value.(flag.Getter).Get().(string)
	}

	if len(confDir) == 0 {
		confDir = "./res"
	}

	filePath := fmt.Sprintf("%v/configuration-driver.toml", confDir)

	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		return config, fmt.Errorf("could not load configuration file (%s): %v", filePath, err.Error())
	}

	err = toml.Unmarshal(file, config)
	if err != nil {
		return config, fmt.Errorf("unable to parse configuration file (%s): %v", filePath, err.Error())
	}
	return config, err
}
