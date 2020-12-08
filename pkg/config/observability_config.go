package config

import (
	"github.com/spf13/pflag"
)

type ObservabilityConfiguration struct {
	// Dex configuration
	DexUrl          string `json:"dex_url" yaml:"dex_url"`
	DexUsername     string `json:"username" yaml:"username"`
	DexPassword     string `json:"password" yaml:"password"`
	DexSecret       string `json:"secret" yaml:"secret"`
	DexSecretFile   string `json:"secret_file" yaml:"secret_file"`
	DexPasswordFile string `json:"password_file" yaml:"password_file"`

	// Observatorium configuration

	ObservatoriumGateway string `json:"gateway" yaml:"gateway"`
	ObservatoriumTenant  string `json:"tenant" yaml:"gateway"`
}

func NewObservabilityConfigurationConfig() *ObservabilityConfiguration {
	return &ObservabilityConfiguration{
		DexSecretFile:        "secrets/dex.secret",
		DexPasswordFile:      "secrets/dex.password",
		ObservatoriumTenant:  "test",
		DexUsername:          "admin@example.com",
		ObservatoriumGateway: "https://observatorium-observatorium.apps.pbraun-observatorium.observability.rhmw.io",
		DexUrl:               "http://dex-dex.apps.pbraun-observatorium.observability.rhmw.io",
	}
}

func (c *ObservabilityConfiguration) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.DexUrl, "dex-url", c.DexUrl, "Dex url")
	fs.StringVar(&c.DexUsername, "dex-username", c.DexUsername, "Dex username")
	fs.StringVar(&c.DexSecretFile, "dex-secret-file", c.DexSecretFile, "Dex secret file")
	fs.StringVar(&c.DexPasswordFile, "dex-password-file", c.DexPasswordFile, "Dex password file")

	fs.StringVar(&c.ObservatoriumGateway, "observatorium-gateway", c.ObservatoriumGateway, "Observatorium gateway")
	fs.StringVar(&c.ObservatoriumTenant, "observatorium-tenant", c.ObservatoriumTenant, "Observatorium tenant")
}

func (c *ObservabilityConfiguration) ReadFiles() error {
	dexPassword, err := readFile(c.DexPasswordFile)
	if err != nil {
		return err
	}
	dexSecret, err := readFile(c.DexSecretFile)
	if err != nil {
		return err
	}

	c.DexPassword = dexPassword
	c.DexSecret = dexSecret
	return nil
}
