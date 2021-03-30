package config

import (
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
	Organisations   OrganisationList `yaml:"allowed_users_per_organisation"`
	ServiceAccounts AllowedAccounts  `yaml:"allowed_service_accounts"`
}

type AccessControlListConfig struct {
	AllowList           AllowListConfiguration
	DenyList            DeniedUsers
	EnableAllowList     bool
	EnableDenyList      bool
	AllowListConfigFile string
	DenyListConfigFile  string
}

func NewAccessControlListConfig() *AccessControlListConfig {
	return &AccessControlListConfig{
		AllowListConfigFile: "config/allow-list-configuration.yaml",
		DenyListConfigFile:  "config/deny-list-configuration.yaml",
		EnableAllowList:     false,
		EnableDenyList:      false,
	}
}

func (c *AccessControlListConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.AllowListConfigFile, "allow-list-config-file", c.AllowListConfigFile, "AllowList configuration file")
	fs.BoolVar(&c.EnableAllowList, "enable-allow-list", c.EnableAllowList, "Enable access control via the allowed list of users")
	fs.StringVar(&c.DenyListConfigFile, "deny-list-config-file", c.DenyListConfigFile, "DenyList configuration file")
	fs.BoolVar(&c.EnableDenyList, "enable-deny-list", c.EnableDenyList, "Enable access control via the denied list of users")
	fs.IntVar(&MaxAllowedInstances, "max-allowed-instances", MaxAllowedInstances, "Default maximum number of allowed instances that can be created by a user")
}

func (c *AccessControlListConfig) ReadFiles() error {
	var err error

	err = readAllowListConfigFile(c.AllowListConfigFile, &c.AllowList)

	if c.EnableDenyList && err == nil {
		err = readDenyListConfigFile(c.DenyListConfigFile, &c.DenyList)
	}

	return err
}

// Read the contents of file into the allow list config
func readAllowListConfigFile(file string, val *AllowListConfiguration) error {
	fileContents, err := readFile(file)
	if err != nil {
		return err
	}

	return yaml.UnmarshalStrict([]byte(fileContents), val)
}

// Read the contents of file into the deny list config
func readDenyListConfigFile(file string, val *DeniedUsers) error {
	fileContents, err := readFile(file)
	if err != nil {
		return err
	}

	return yaml.UnmarshalStrict([]byte(fileContents), val)
}
