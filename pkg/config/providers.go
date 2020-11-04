package config

import (
	"fmt"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

type Region struct {
	Name    string `json:"name"`
	Default bool   `json:"default"`
}

type RegionList []Region

func (rl RegionList) GetByName(regionName string) (Region, bool) {
	for _, r := range rl {
		if r.Name == regionName {
			return r, true
		}
	}
	return Region{}, false
}

func (rl RegionList) String() string {
	var names []string
	for _, r := range rl {
		names = append(names, r.Name)
	}
	return fmt.Sprint(names)
}

type Provider struct {
	Name    string     `json:"name"`
	Default bool       `json:"default"`
	Regions RegionList `json:"regions"`
}

type ProviderList []Provider

func (pl ProviderList) GetByName(providerName string) (Provider, bool) {
	for _, p := range pl {
		if p.Name == providerName {
			return p, true
		}
	}
	return Provider{}, false
}

func (pl ProviderList) String() string {
	var names []string
	for _, p := range pl {
		names = append(names, p.Name)
	}
	return fmt.Sprint(names)
}

type ProviderConfiguration struct {
	SupportedProviders ProviderList `yaml:"supported_providers"`
}

type ProviderConfig struct {
	ProvidersConfig     ProviderConfiguration `json:"providers"`
	ProvidersConfigFile string                `json:"providers_config_file"`
}

func NewSupportedProvidersConfig() *ProviderConfig {
	return &ProviderConfig{
		ProvidersConfigFile: "config/provider-configuration.yaml",
	}
}

func (c *ProviderConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.ProvidersConfigFile, "providers-config-file", c.ProvidersConfigFile, "SupportedProviders configuration file")
}

func (c *ProviderConfig) ReadFiles() error {
	return readFileProvidersConfig(c.ProvidersConfigFile, &c.ProvidersConfig)
}

// Read the contents of file into the providers config
func readFileProvidersConfig(file string, val *ProviderConfiguration) error {
	fileContents, err := readFile(file)
	if err != nil {
		return err
	}
	return yaml.UnmarshalStrict([]byte(fileContents), val)
}
