
package driver

import (
	"fmt"
	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"reflect"
	"strconv"
)

const (
	defaultProtocol = "opc.tcp"
	defaultPolicy 	= "None"
	defaultMode   	= "None"
)

// Configuration can be configured in configuration.toml
type Configuration struct {
	Protocol        string
	Address     	string
	Port			string
	Path 			string
	Policy 			string
	Mode  			string
	CertFile	 	string
	KeyFile 		string
	MappingStr      string
}

func (config *Configuration) setDefaultVal()  {
	if config.Protocol == "" {
		config.Protocol = defaultProtocol
	}
	if config.Policy == "" {
		config.Policy = defaultPolicy
	}
	if config.Mode == "" {
		config.Mode = defaultMode
	}
}
// CreateConfigurationAndMapping use to load connectionInfo for read and write command
func CreateConfigurationAndMapping(protocols map[string]models.ProtocolProperties) (*Configuration, map[string]string, error) {
	config := new(Configuration)
	protocol, ok := protocols[Protocol]
	if !ok {
		return nil, nil, fmt.Errorf("unable to load config, 'opcua' not exist")
	}
	err := load(protocol, config)
	if err != nil {
		return nil, nil, err
	}
	config.setDefaultVal()

	mapping, err := createNodeMapping(config.MappingStr)
	if err != nil {
		return config, nil, err
	}
	return config, mapping, nil
}

// load by reflect to check map key and then fetch the value
func load(config map[string]string, des interface{}) error {
	val := reflect.ValueOf(des).Elem()
	for i := 0; i < val.NumField(); i++ {
		typeField := val.Type().Field(i)
		valueField := val.Field(i)

		val := config[typeField.Name]
		switch valueField.Kind() {
		case reflect.Int:
			intVal, err := strconv.Atoi(val)
			if err != nil {
				return err
			}
			valueField.SetInt(int64(intVal))
		case reflect.String:
			valueField.SetString(val)
		default:
			return fmt.Errorf("none supported value type %v ,%v", valueField.Kind(), typeField.Name)
		}
	}
	return nil
}
