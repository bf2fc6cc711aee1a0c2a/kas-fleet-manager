package config

import (
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

var MaxAllowedInstances int = 1

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

type AllowListConfiguration struct {
	Organisations   OrganisationList `yaml:"allowed_users_per_organisation"`
	ServiceAccounts AllowedAccounts  `yaml:"allowed_service_accounts"`
}

type AllowListConfig struct {
	AllowList           AllowListConfiguration
	EnableAllowList     bool
	AllowListConfigFile string
}

func NewAllowListConfig() *AllowListConfig {
	return &AllowListConfig{
		AllowListConfigFile: "config/allow-list-configuration.yaml",
		EnableAllowList:     false,
	}
}

func (c *AllowListConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.AllowListConfigFile, "allow-list-config-file", c.AllowListConfigFile, "AllowList configuration file")
	fs.BoolVar(&c.EnableAllowList, "enable-allow-list", c.EnableAllowList, "Enable allow list of users")
	fs.IntVar(&MaxAllowedInstances, "max-allowed-instances", MaxAllowedInstances, "Maximumm number of allowed instances that can be created by the user")
}

func (c *AllowListConfig) ReadFiles() error {
	return readFileOrganisationsConfig(c.AllowListConfigFile, &c.AllowList)
}

// Read the contents of file into the allow list config
func readFileOrganisationsConfig(file string, val *AllowListConfiguration) error {
	fileContents, err := readFile(file)
	if err != nil {
		return err
	}

	return yaml.UnmarshalStrict([]byte(fileContents), val)
}
