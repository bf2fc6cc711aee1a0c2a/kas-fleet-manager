package acl

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

type EnterpriseClusterRegistrationAcceptedOrganizations []string

func (acceptedOrganizations EnterpriseClusterRegistrationAcceptedOrganizations) IsOrganizationAccepted(orgId string) bool {
	return arrays.Contains(acceptedOrganizations, orgId)
}

type EnterpriseClusterRegistrationAccessControlListConfig struct {
	EnterpriseClusterRegistrationAccessControlList           EnterpriseClusterRegistrationAcceptedOrganizations
	EnterpriseClusterRegistrationAccessControlListConfigFile string
}

func NewEnterpriseClusterRegistrationAccessControlListConfig() *EnterpriseClusterRegistrationAccessControlListConfig {
	return &EnterpriseClusterRegistrationAccessControlListConfig{
		EnterpriseClusterRegistrationAccessControlListConfigFile: "config/enterprise-cluster-registration-list-config.yaml",
	}
}

func (c *EnterpriseClusterRegistrationAccessControlListConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.EnterpriseClusterRegistrationAccessControlListConfigFile, "enterprise-cluster-registration-access-control-list-config-file", c.EnterpriseClusterRegistrationAccessControlListConfigFile, "AccessList configuration file")
}

func (c *EnterpriseClusterRegistrationAccessControlListConfig) ReadFiles() (err error) {
	err = readEnterpriseClusterRegistrationAccessControlListConfigFile(c.EnterpriseClusterRegistrationAccessControlListConfigFile, &c.EnterpriseClusterRegistrationAccessControlList)
	if err != nil {
		return err
	}

	return nil
}

// Read the contents of file into the access list config
func readEnterpriseClusterRegistrationAccessControlListConfigFile(file string, val *EnterpriseClusterRegistrationAcceptedOrganizations) error {
	fileContents, err := shared.ReadFile(file)
	if err != nil {
		return err
	}

	return yaml.UnmarshalStrict([]byte(fileContents), val)
}
