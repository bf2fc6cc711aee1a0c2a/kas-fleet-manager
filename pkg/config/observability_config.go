package config

import (
	"github.com/spf13/pflag"
	"time"
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

	ObservatoriumGateway string        `json:"gateway" yaml:"gateway"`
	ObservatoriumTenant  string        `json:"tenant" yaml:"gateway"`
	AuthToken            string        `json:"auth_token"`
	AuthTokenFile        string        `json:"auth_token_file"`
	Cookie               string        `json:"cookie"`
	Timeout              time.Duration `json:"timeout"`
	Insecure             bool          `json:"insecure"`
	Debug                bool          `json:"debug"`
	EnableMock           bool          `json:"enable_mock"`

	// Configuration repo for the Observability operator
	ObservabilityConfigTag             string `json:"observability_config_tag"`
	ObservabilityConfigRepo            string `json:"observability_config_repo"`
	ObservabilityConfigChannel         string `json:"observability_config_channel"`
	ObservabilityConfigAccessToken     string `json:"observability_config_access_token"`
	ObservabilityConfigAccessTokenFile string `json:"observability_config_access_token_file"`
}

func NewObservabilityConfigurationConfig() *ObservabilityConfiguration {
	return &ObservabilityConfiguration{
		DexSecretFile:                      "secrets/dex.secret",
		DexPasswordFile:                    "secrets/dex.password",
		ObservatoriumTenant:                "test",
		DexUsername:                        "admin@example.com",
		ObservatoriumGateway:               "https://observatorium-observatorium.apps.pbraun-observatorium.observability.rhmw.io",
		DexUrl:                             "http://dex-dex.apps.pbraun-observatorium.observability.rhmw.io",
		AuthToken:                          "",
		AuthTokenFile:                      "secrets/observatorium.token",
		Timeout:                            240 * time.Second,
		Debug:                              true, // TODO: false
		EnableMock:                         false,
		Insecure:                           true, // TODO: false
		ObservabilityConfigRepo:            "https://api.github.com/repos/bf2fc6cc711aee1a0c2a/observability-resources-mk/contents",
		ObservabilityConfigChannel:         "resources", // Pointing to resources as the individual directories for prod and staging are no longer needed
		ObservabilityConfigAccessToken:     "",
		ObservabilityConfigAccessTokenFile: "secrets/observability-config-access.token",
		ObservabilityConfigTag:             "v1.4.0-staging",
	}
}

func (c *ObservabilityConfiguration) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.DexUrl, "dex-url", c.DexUrl, "Dex url")
	fs.StringVar(&c.DexUsername, "dex-username", c.DexUsername, "Dex username")
	fs.StringVar(&c.DexSecretFile, "dex-secret-file", c.DexSecretFile, "Dex secret file")
	fs.StringVar(&c.DexPasswordFile, "dex-password-file", c.DexPasswordFile, "Dex password file")

	fs.StringVar(&c.ObservatoriumGateway, "observatorium-gateway", c.ObservatoriumGateway, "Observatorium gateway")
	fs.StringVar(&c.ObservatoriumTenant, "observatorium-tenant", c.ObservatoriumTenant, "Observatorium tenant")
	fs.StringVar(&c.AuthTokenFile, "observatorium-token-file", c.AuthTokenFile, "Token File for Observatorium client")
	fs.DurationVar(&c.Timeout, "observatorium-timeout", c.Timeout, "Timeout for Observatorium client")
	fs.BoolVar(&c.Insecure, "observatorium-ignore-ssl", c.Insecure, "ignore SSL Observatorium certificate")
	fs.BoolVar(&c.EnableMock, "enable-observatorium-mock", c.EnableMock, "Enable mock Observatorium client")
	fs.BoolVar(&c.Debug, "observatorium-debug", c.Debug, "Debug flag for Observatorium client")

	fs.StringVar(&c.ObservabilityConfigRepo, "observability-config-repo", c.ObservabilityConfigRepo, "Repo for the observability operator configuration repo")
	fs.StringVar(&c.ObservabilityConfigChannel, "observability-config-channel", c.ObservabilityConfigChannel, "Channel for the observability operator configuration repo")
	fs.StringVar(&c.ObservabilityConfigAccessTokenFile, "observability-config-access-token-file", c.ObservabilityConfigAccessTokenFile, "File contains the access token to the observability operator configuration repo")
	fs.StringVar(&c.ObservabilityConfigTag, "observability-config-tag", c.ObservabilityConfigTag, "Tag or branch to use inside the observability configuration repo")

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

	if c.AuthToken == "" && c.AuthTokenFile != "" {
		err := readFileValueString(c.AuthTokenFile, &c.AuthToken)
		if err != nil {
			return err
		}
	}

	if c.ObservabilityConfigAccessToken == "" && c.ObservabilityConfigAccessTokenFile != "" {
		return readFileValueString(c.ObservabilityConfigAccessTokenFile, &c.ObservabilityConfigAccessToken)
	}

	return nil
}
