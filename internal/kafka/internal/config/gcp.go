package config

import (
	"fmt"
	"os"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cloudproviders"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/pflag"
)

const (
	defaultGCPCredentialsFilePath = "secrets/gcp.api-credentials"
)

type GCPConfig struct {
	GCPCredentials         GCPCredentials
	gcpCredentialsFilePath string
}

type GCPCredentials struct {
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url" validate:"required"`
	AuthURI                 string `json:"auth_uri" validate:"required"`
	ClientEmail             string `json:"client_email" validate:"required"`
	ClientID                string `json:"client_id" validate:"required"`
	ClientX509CertURL       string `json:"client_x509_cert_url" validate:"required"`
	PrivateKey              string `json:"private_key" validate:"required"`
	PrivateKeyID            string `json:"private_key_id" validate:"required"`
	ProjectID               string `json:"project_id" validate:"required"`
	TokenURI                string `json:"token_uri" validate:"required"`
	Type                    string `json:"type" validate:"required"`
}

func NewGCPConfig() *GCPConfig {
	return &GCPConfig{
		gcpCredentialsFilePath: defaultGCPCredentialsFilePath,
	}
}

func (c *GCPConfig) ReadFiles() error {
	var gcpCredentials GCPCredentials

	err := shared.ReadJSONFile(c.gcpCredentialsFilePath, &gcpCredentials)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error reading file %q: %v", c.gcpCredentialsFilePath, err)
	}

	c.GCPCredentials = gcpCredentials
	return nil
}

func (c *GCPConfig) Validate(env *environments.Env) error {
	var providersConfig *ProviderConfig
	env.MustResolve(&providersConfig)

	err := c.GCPCredentials.validate(providersConfig.ProvidersConfig.SupportedProviders)
	return err
}

func (c *GCPConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.gcpCredentialsFilePath, "gcp-api-credentials-file", defaultGCPCredentialsFilePath, "Path to a file containing GCP API Credentials in JSON format")
}

func (c *GCPCredentials) validate(providerList ProviderList) error {
	_, found := providerList.GetByName(cloudproviders.GCP.String())
	if !found {
		return nil
	}

	validate := validator.New()
	err := validate.Struct(c)
	if err != nil {
		return fmt.Errorf("error validating GCP API credentials: %v", err)
	}

	return nil
}
