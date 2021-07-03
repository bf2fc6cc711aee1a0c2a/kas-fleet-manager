package acl

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

var MaxAllowedInstances int = 1

// Returns the default max allowed instances for both internal (users and orgs in allow list config) and external users
func GetDefaultMaxAllowedInstances() int {
	return MaxAllowedInstances
}

type AllowedListItem interface {
	// IsInstanceCountWithinLimit returns true if the given count is within limits
	IsInstanceCountWithinLimit(count int) bool
	// GetMaxAllowedInstances returns maximum number of allowed instances.
	GetMaxAllowedInstances() int
}

type Organisation struct {
	Id                  string          `yaml:"id"`
	AllowAll            bool            `yaml:"allow_all"`
	MaxAllowedInstances int             `yaml:"max_allowed_instances"`
	AllowedAccounts     AllowedAccounts `yaml:"allowed_users"`
}

func (org Organisation) IsUserAllowed(username string) bool {
	if !org.HasAllowedAccounts() {
		return org.AllowAll
	}
	_, found := org.AllowedAccounts.GetByUsername(username)
	return found
}

func (org Organisation) HasAllowedAccounts() bool {
	return len(org.AllowedAccounts) > 0
}

func (org Organisation) IsInstanceCountWithinLimit(count int) bool {
	return count < org.GetMaxAllowedInstances()
}

func (org Organisation) GetMaxAllowedInstances() int {
	if org.MaxAllowedInstances <= 0 {
		return MaxAllowedInstances
	}

	return org.MaxAllowedInstances
}

type OrganisationList []Organisation

func (orgList OrganisationList) GetById(Id string) (Organisation, bool) {
	for _, organisation := range orgList {
		if Id == organisation.Id {
			return organisation, true
		}
	}

	return Organisation{}, false
}

type AllowedAccount struct {
	Username            string `yaml:"username"`
	MaxAllowedInstances int    `yaml:"max_allowed_instances"`
}

func (account AllowedAccount) IsInstanceCountWithinLimit(count int) bool {
	return count < account.GetMaxAllowedInstances()
}

func (account AllowedAccount) GetMaxAllowedInstances() int {
	if account.MaxAllowedInstances <= 0 {
		return MaxAllowedInstances
	}

	return account.MaxAllowedInstances
}

type AllowedAccounts []AllowedAccount

func (allowedAccounts AllowedAccounts) GetByUsername(username string) (AllowedAccount, bool) {
	for _, user := range allowedAccounts {
		if username == user.Username {
			return user, true
		}
	}

	return AllowedAccount{}, false
}

type DeniedUsers []string

func (deniedAccounts DeniedUsers) IsUserDenied(username string) bool {
	for _, user := range deniedAccounts {
		if username == user {
			return true
		}
	}

	return false
}

type AllowListConfiguration struct {
	AllowAnyRegisteredUsers bool             `yaml:"allow_any_registered_users"`
	Organisations           OrganisationList `yaml:"allowed_users_per_organisation"`
	ServiceAccounts         AllowedAccounts  `yaml:"allowed_service_accounts"`
}

type AccessControlListConfig struct {
	AllowList                  AllowListConfiguration
	DenyList                   DeniedUsers
	EnableDenyList             bool
	AllowListConfigFile        string
	DenyListConfigFile         string
	EnableInstanceLimitControl bool
}

func NewAccessControlListConfig() *AccessControlListConfig {
	return &AccessControlListConfig{
		AllowListConfigFile:        "config/allow-list-configuration.yaml",
		DenyListConfigFile:         "config/deny-list-configuration.yaml",
		EnableDenyList:             false,
		EnableInstanceLimitControl: false,
	}
}

func (c *AccessControlListConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.AllowListConfigFile, "allow-list-config-file", c.AllowListConfigFile, "AllowList configuration file")
	fs.StringVar(&c.DenyListConfigFile, "deny-list-config-file", c.DenyListConfigFile, "DenyList configuration file")
	fs.BoolVar(&c.EnableDenyList, "enable-deny-list", c.EnableDenyList, "Enable access control via the denied list of users")
	fs.IntVar(&MaxAllowedInstances, "max-allowed-instances", MaxAllowedInstances, "Default maximum number of allowed instances that can be created by a user")
	fs.BoolVar(&c.EnableInstanceLimitControl, "enable-instance-limit-control", c.EnableInstanceLimitControl, "Enable to enforce limits on how much instances a user can create")
}

func (c *AccessControlListConfig) ReadFiles() error {
	err := readAllowListConfigFile(c.AllowListConfigFile, &c.AllowList)

	if c.EnableDenyList && err == nil {
		err = readDenyListConfigFile(c.DenyListConfigFile, &c.DenyList)
	}

	return err
}

// Read the contents of file into the allow list config
func readAllowListConfigFile(file string, val *AllowListConfiguration) error {
	fileContents, err := shared.ReadFile(file)
	if err != nil {
		return err
	}

	return yaml.UnmarshalStrict([]byte(fileContents), val)
}

// Read the contents of file into the deny list config
func readDenyListConfigFile(file string, val *DeniedUsers) error {
	fileContents, err := shared.ReadFile(file)
	if err != nil {
		return err
	}

	return yaml.UnmarshalStrict([]byte(fileContents), val)
}

func (c *AccessControlListConfig) GetAllowedAccountByUsernameAndOrgId(username string, orgId string) (AllowedAccount, bool) {
	var user AllowedAccount
	var found bool
	org, _ := c.AllowList.Organisations.GetById(orgId)
	user, found = org.AllowedAccounts.GetByUsername(username)
	if found {
		return user, found
	}
	return c.AllowList.ServiceAccounts.GetByUsername(username)
}
