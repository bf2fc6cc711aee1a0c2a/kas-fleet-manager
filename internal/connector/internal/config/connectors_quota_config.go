package config

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
	"os"
)

const AnnotationProfileKey = "connector_mgmt.bf2.org/profile"

type ConnectorsQuotaConfig struct {
	ConnectorsQuotaMap        ConnectorsQuotaProfileMap
	ConnectorsQuotaConfigFile string
	EvalNamespaceQuotaProfile string
}

func NewConnectorsQuotaConfig() *ConnectorsQuotaConfig {
	return &ConnectorsQuotaConfig{
		ConnectorsQuotaMap:        make(ConnectorsQuotaProfileMap),
		ConnectorsQuotaConfigFile: "config/connectors-quota-configuration.yaml",
		EvalNamespaceQuotaProfile: "evaluation-profile",
	}
}

func (c *ConnectorsQuotaConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.ConnectorsQuotaConfigFile, "connectors-quota-config-file", c.ConnectorsQuotaConfigFile, "Connectors quota configuration file")
	fs.StringVar(&c.EvalNamespaceQuotaProfile, "connectors-eval-namespace-quota-profile", c.EvalNamespaceQuotaProfile, "Connectors quota profile name for evaluation namespaces")
}

func (c *ConnectorsQuotaConfig) ReadFiles() (err error) {
	err = readQuotaConfigFile(c.ConnectorsQuotaConfigFile, c.ConnectorsQuotaMap)

	if err == nil {
		if _, ok := c.ConnectorsQuotaMap[c.EvalNamespaceQuotaProfile]; !ok {
			err = fmt.Errorf("Configuration file '%s' is missing evaluation namespace quota profile '%s'",
				c.ConnectorsQuotaConfigFile, c.EvalNamespaceQuotaProfile)
		}
	} else if os.IsNotExist(err) {
		err = fmt.Errorf("Configuration file for connectors-quota-config-file not found: '%s'", c.ConnectorsQuotaConfigFile)
	} else {
		err = fmt.Errorf("Error reading configuration file '%s': %s", c.ConnectorsQuotaConfigFile, err)
	}

	return err
}

// Read the contents of file into the quota list config
func readQuotaConfigFile(file string, val ConnectorsQuotaProfileMap) error {
	fileContents, err := shared.ReadFile(file)
	if err != nil {
		return err
	}

	var quotaList ConnectorsQuotaProfileList
	err = yaml.UnmarshalStrict([]byte(fileContents), &quotaList)
	if err == nil {
		for _, profile := range quotaList {
			val[profile.Name] = profile.Quotas
		}
	}

	return err
}

func (c *ConnectorsQuotaConfig) GetNamespaceQuota(profileName string) (NamespaceQuota, bool) {
	profile, ok := c.ConnectorsQuotaMap[profileName]
	return profile.NamespaceQuota, ok
}

func (c *ConnectorsQuotaConfig) GetEvalNamespaceQuota() (NamespaceQuota, bool) {
	profile, ok := c.ConnectorsQuotaMap[c.EvalNamespaceQuotaProfile]
	return profile.NamespaceQuota, ok
}
