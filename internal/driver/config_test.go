package driver

import (
	"fmt"
	"testing"

	"github.com/edgexfoundry/go-mod-core-contracts/models"
)

func Test(t *testing.T) {

	protocols := map[string]models.ProtocolProperties{
		Protocol: {
			Address: "192.168.3.165",
			Port:  "53530",
			Path:  "/OPCUA/SimulationServer",
			Policy:  "None",
			Mode:  "None",
			CertFile:  "",
			KeyFile:  "",
			MappingStr: "{ \"Counter\" = \"ns=5;s=Counter1\", \"Random\" = \"ns=5;s=Random1\" }",
		},
	}

	q, _ := CreateConnectionInfo(protocols)
	fmt.Println(q)
}
