package acl

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

type DeniedUsers []string
type AcceptedOrganisations []string

func (deniedAccounts DeniedUsers) IsUserDenied(username string) bool {
	return arrays.Contains(deniedAccounts, username)
}

func (acceptedOrganisations AcceptedOrganisations) IsOrganisationAccepted(orgId string) bool {
	return arrays.Contains(acceptedOrganisations, orgId)
}

type AccessControlListConfig struct {
	DenyList             DeniedUsers
	AccessList           AcceptedOrganisations
	DenyListConfigFile   string
	AccessListConfigFile string
	EnableDenyList       bool
	EnableAccessList     bool
}

func NewAccessControlListConfig() *AccessControlListConfig {
	return &AccessControlListConfig{
		DenyListConfigFile:   "config/deny-list-configuration.yaml",
		AccessListConfigFile: "config/access-list-configuration.yaml",
		EnableDenyList:       false,
		EnableAccessList:     false,
	}
}

func (c *AccessControlListConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.DenyListConfigFile, "deny-list-config-file", c.DenyListConfigFile, "DenyList configuration file")
	fs.BoolVar(&c.EnableDenyList, "enable-deny-list", c.EnableDenyList, "Enable access control via the denied list of users")
	fs.StringVar(&c.AccessListConfigFile, "access-list-config-file", c.AccessListConfigFile, "AccessList configuration file")
	fs.BoolVar(&c.EnableAccessList, "enable-access-list", c.EnableAccessList, "Enable access control via the accepted list of organisations")
}

func (c *AccessControlListConfig) ReadFiles() (err error) {
	if c.EnableDenyList {
		err := readDenyListConfigFile(c.DenyListConfigFile, &c.DenyList)
		if err != nil {
			return err
		}
	}
	if c.EnableAccessList {
		err = readAccessListConfigFile(c.AccessListConfigFile, &c.AccessList)
		if err != nil {
			return err
		}
	}

	return nil
}

// Read the contents of file into the deny list config
func readDenyListConfigFile(file string, val *DeniedUsers) error {
	fileContents, err := shared.ReadFile(file)
	if err != nil {
		return err
	}

	return yaml.UnmarshalStrict([]byte(fileContents), val)
}

// Read the contents of file into the access list config
func readAccessListConfigFile(file string, val *AcceptedOrganisations) error {
	fileContents, err := shared.ReadFile(file)
	if err != nil {
		return err
	}

	return yaml.UnmarshalStrict([]byte(fileContents), val)
}
