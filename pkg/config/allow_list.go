package config

import (
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

type Organisation struct {
	Id           string   `yaml:"id"`
	AllowAll     bool     `yaml:"allow_all"`
	AllowedUsers []string `yaml:"allowed_users"`
}

func (org Organisation) IsUserAllowed(username string) bool {
	if !org.HasAllowedUsers() {
		return org.AllowAll
	}

	for _, user := range org.AllowedUsers {
		if user == username {
			return true
		}
	}

	return false
}

func (org Organisation) HasAllowedUsers() bool {
	return len(org.AllowedUsers) > 0
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

type AllowedUsers []string

type AllowListConfiguration struct {
	Organisations OrganisationList `yaml:"allowed_users_per_organisation"`
	AllowedUsers  AllowedUsers     `yaml:"allowed_users"`
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
