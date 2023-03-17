package observatorium

import (
	"fmt"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/pflag"
)

const (
	AuthTypeDex = "dex"
	AuthTypeSso = "redhat"
)

const (
	defaultObservabilityCloudwatchCredentialsSecretName      = "clo-cloudwatchlogs-creds"
	defaultObservabilityCloudwatchCredentialsSecretNamespace = "openshift-logging"
)

type ObservabilityConfiguration struct {
	// Dex configuration
	DexUrl          string `json:"dex_url" yaml:"dex_url"`
	DexUsername     string `json:"username" yaml:"username"`
	DexPassword     string `json:"password" yaml:"password"`
	DexSecret       string `json:"secret" yaml:"secret"`
	DexSecretFile   string `json:"secret_file" yaml:"secret_file"`
	DexPasswordFile string `json:"password_file" yaml:"password_file"`

	// Red Hat SSO configuration
	RedHatSsoAuthServerUrl     string `json:"redhat_sso_auth_server_url" yaml:"redhat_sso_auth_server_url"`
	RedHatSsoRealm             string `json:"redhat_sso_realm" yaml:"redhat_sso_realm"`
	RedHatSsoTenant            string `json:"redhat_sso_tenant" yaml:"redhat_sso_tenant"`
	RedHatSsoTokenRefresherUrl string `json:"redhat_sso_token_refresher_url" yaml:"redhat_sso_token_refresher_url"`

	// Observatorium configuration
	ObservatoriumGateway string        `json:"gateway" yaml:"gateway"`
	ObservatoriumTenant  string        `json:"observatorium_tenant" yaml:"observatorium_tenant"`
	AuthType             string        `json:"auth_type" yaml:"auth_type"`
	AuthToken            string        `json:"auth_token"`
	AuthTokenFile        string        `json:"auth_token_file"`
	Cookie               string        `json:"cookie"`
	Timeout              time.Duration `json:"timeout"`
	Insecure             bool          `json:"insecure"`
	Debug                bool          `json:"debug"`
	EnableMock           bool          `json:"enable_mock"`

	// Configuration repo for the Observability operator
	ObservabilityConfigTag     string `json:"observability_config_tag"`
	ObservabilityConfigRepo    string `json:"observability_config_repo"`
	ObservabilityConfigChannel string `json:"observability_config_channel"`

	// Configuration of AWS CloudWatch Logging for Observability
	ObservabilityCloudWatchLoggingConfig ObservabilityCloudWatchLoggingConfig
	DataPlaneObservabilityConfig         DataPlaneObservabilityConfig
}

var _ environments.ConfigModule = &ObservabilityConfiguration{}
var _ environments.ServiceValidator = &ObservabilityConfiguration{}

type ObservabilityCloudWatchLoggingConfig struct {
	Credentials                   ObservabilityCloudwatchLoggingConfigCredentials             `yaml:"aws_iam_credentials" validate:"dive"`
	EnterpriseCredentials         []ObservabilityEnterpriseCloudwatchLoggingConfigCredentials `yaml:"aws_iam_credentials_enterprise" validate:"dive"`
	K8sCredentialsSecretName      string                                                      `yaml:"k8s_credentials_secret_name" validate:"omitempty,oneof=clo-cloudwatchlogs-creds"`
	K8sCredentialsSecretNamespace string                                                      `yaml:"k8s_credentials_secret_namespace" validate:"omitempty,oneof=openshift-logging"`
	CloudwatchLoggingEnabled      bool                                                        `validate:"-"`
	configFilePath                string                                                      `validate:"-"`
}

type DataPlaneObservabilityOIDCCredentials struct {
	ClientID     string `yaml:"client_id" validate:"required"`
	ClientSecret string `yaml:"client_secret" validate:"required"`
}

type DataPlaneObservabilityOIDCConfiguration struct {
	AuthorizationServer string                                 `yaml:"authorization_server" validate:"required"`
	Realm               string                                 `yaml:"realm" validate:"required"`
	Credentials         *DataPlaneObservabilityOIDCCredentials `yaml:"credentials" validate:"required,dive"`
}

type DataPlaneObservabilityRemoteWriteConfiguration struct {
	RemoteWriteUrl    string                                   `yaml:"remote_write_url" validate:"required"`
	OIDCConfiguration *DataPlaneObservabilityOIDCConfiguration `yaml:"oidc_configuration" validate:"omitempty,dive"`
}

type DataPlaneObservabilityConfig struct {
	configFilePath string
	Enabled        bool

	RemoteWriteConfiguration *DataPlaneObservabilityRemoteWriteConfiguration `yaml:"remote_write_configuration" validate:"required,dive"`
	GithubResourcesAuthToken string
}

func (c *DataPlaneObservabilityConfig) GetOIDCClientID() string {
	if c.Enabled && c.RemoteWriteConfiguration.OIDCConfiguration != nil {
		return c.RemoteWriteConfiguration.OIDCConfiguration.Credentials.ClientID
	}

	return ""
}

func (c *DataPlaneObservabilityConfig) GetOIDCClientSecret() string {
	if c.Enabled && c.RemoteWriteConfiguration.OIDCConfiguration != nil {
		return c.RemoteWriteConfiguration.OIDCConfiguration.Credentials.ClientSecret
	}

	return ""
}

func (c *DataPlaneObservabilityConfig) GetOIDCAuthorizationServer() string {
	if c.Enabled && c.RemoteWriteConfiguration.OIDCConfiguration != nil {
		return c.RemoteWriteConfiguration.OIDCConfiguration.AuthorizationServer
	}

	return ""
}

func (c *DataPlaneObservabilityConfig) GetOIDCRealm() string {
	if c.Enabled && c.RemoteWriteConfiguration.OIDCConfiguration != nil {
		return c.RemoteWriteConfiguration.OIDCConfiguration.Realm
	}

	return ""
}

func (c *DataPlaneObservabilityConfig) GetRemoteWriteUrl() string {
	if c.Enabled && c.RemoteWriteConfiguration != nil {
		return c.RemoteWriteConfiguration.RemoteWriteUrl
	}

	return ""
}

func (c *DataPlaneObservabilityConfig) HasOIDCConfiguration() bool {
	return c.Enabled && c.RemoteWriteConfiguration.OIDCConfiguration != nil
}

func (c *ObservabilityCloudWatchLoggingConfig) validate() error {
	if !c.CloudwatchLoggingEnabled {
		return nil
	}

	validate := validator.New()
	err := validate.Struct(c)
	if err != nil {
		return fmt.Errorf("error validating Observability CloudWatch Logging config: %v", err)
	}

	return nil
}

func (c *DataPlaneObservabilityConfig) validate() error {
	if !c.Enabled {
		return nil
	}

	validate := validator.New()
	err := validate.Struct(c)
	if err != nil {
		return fmt.Errorf("error validating data plane observability config config: %v", err)
	}

	return nil
}

func (c *DataPlaneObservabilityConfig) readConfigFile() error {
	if !c.Enabled {
		return nil
	}

	if c.configFilePath == "" {
		return fmt.Errorf("error reading data plane observability configuration: file path cannot be empty")
	}

	err := shared.ReadYamlFile(c.configFilePath, c)
	if err != nil {
		return err
	}

	return nil
}

func (c *ObservabilityCloudWatchLoggingConfig) readConfigFile() error {
	if !c.CloudwatchLoggingEnabled {
		return nil
	}

	if c.configFilePath == "" {
		return fmt.Errorf("error reading observability cloudwatch logging configuration: observability cloudwatch logging credentials file path cannot be empty")
	}

	err := shared.ReadYamlFile(c.configFilePath, &c)
	if err != nil {
		return err
	}
	c.setDefaults()
	return nil
}

func (c *ObservabilityCloudWatchLoggingConfig) setDefaults() {
	if c.K8sCredentialsSecretName == "" {
		c.K8sCredentialsSecretName = defaultObservabilityCloudwatchCredentialsSecretName
	}
	if c.K8sCredentialsSecretNamespace == "" {
		c.K8sCredentialsSecretNamespace = defaultObservabilityCloudwatchCredentialsSecretNamespace
	}
}

func (c *ObservabilityCloudWatchLoggingConfig) GetEnterpriseCredentials(orgID string) *ObservabilityCloudwatchLoggingConfigCredentials {
	credentials := &ObservabilityCloudwatchLoggingConfigCredentials{}
	idx, enterpriseCredential := arrays.FindFirst(c.EnterpriseCredentials, func(enterpriseCredential ObservabilityEnterpriseCloudwatchLoggingConfigCredentials) bool {
		return orgID == enterpriseCredential.OrgID
	})
	if idx == arrays.ElementNotFound {
		return nil
	}

	credentials.AccessKey = enterpriseCredential.Credentials.AccessKey
	credentials.SecretAccessKey = enterpriseCredential.Credentials.SecretAccessKey

	return credentials
}

type ObservabilityCloudwatchLoggingConfigCredentials struct {
	AccessKey       string `yaml:"aws_access_key" validate:"required"`
	SecretAccessKey string `yaml:"aws_secret_access_key" validate:"required"`
}

type ObservabilityEnterpriseCloudwatchLoggingConfigCredentials struct {
	Credentials ObservabilityCloudwatchLoggingConfigCredentials `yaml:"credentials" validate:"dive"`
	OrgID       string                                          `yaml:"org_id" validate:"required"`
}

func NewObservabilityConfigurationConfig() *ObservabilityConfiguration {
	return &ObservabilityConfiguration{
		DexUrl:                     "http://dex-dex.apps.pbraun-observatorium.observability.rhmw.io",
		DexSecretFile:              "secrets/dex.secret",
		DexPasswordFile:            "secrets/dex.password",
		DexUsername:                "admin@example.com",
		ObservatoriumGateway:       "https://observatorium-observatorium.apps.pbraun-observatorium.observability.rhmw.io",
		ObservatoriumTenant:        "test",
		AuthType:                   "dex",
		AuthToken:                  "",
		AuthTokenFile:              "secrets/observatorium.token",
		Timeout:                    240 * time.Second,
		Debug:                      true, // TODO: false
		EnableMock:                 false,
		Insecure:                   true, // TODO: false
		ObservabilityConfigRepo:    "https://api.github.com/repos/bf2fc6cc711aee1a0c2a/observability-resources-mk/contents",
		ObservabilityConfigChannel: "resources", // Pointing to resources as the individual directories for prod and staging are no longer needed
		ObservabilityConfigTag:     "main",
		RedHatSsoTenant:            "",
		RedHatSsoAuthServerUrl:     "",
		RedHatSsoRealm:             "",
		RedHatSsoTokenRefresherUrl: "",
	}
}

func (c *ObservabilityConfiguration) AddFlags(fs *pflag.FlagSet) {

	fs.StringVar(&c.DexUrl, "dex-url", c.DexUrl, "Dex url")
	fs.StringVar(&c.DexUsername, "dex-username", c.DexUsername, "Dex username")
	fs.StringVar(&c.DexSecretFile, "dex-secret-file", c.DexSecretFile, "Dex secret file")
	fs.StringVar(&c.DexPasswordFile, "dex-password-file", c.DexPasswordFile, "Dex password file")

	fs.StringVar(&c.RedHatSsoTenant, "observability-red-hat-sso-tenant", c.RedHatSsoTenant, "Red Hat SSO tenant")
	fs.StringVar(&c.RedHatSsoAuthServerUrl, "observability-red-hat-sso-auth-server-url", c.RedHatSsoAuthServerUrl, "Red Hat SSO auth server URL")
	fs.StringVar(&c.RedHatSsoTokenRefresherUrl, "observability-red-hat-sso-token-refresher-url", c.RedHatSsoTokenRefresherUrl, "Red Hat SSO token refresher URL")
	fs.StringVar(&c.RedHatSsoRealm, "observability-red-hat-sso-realm", c.RedHatSsoRealm, "Red Hat SSO realm")

	fs.StringVar(&c.ObservatoriumGateway, "observatorium-gateway", c.ObservatoriumGateway, "Observatorium gateway")
	fs.StringVar(&c.ObservatoriumTenant, "observatorium-tenant", c.ObservatoriumTenant, "Observatorium tenant")
	fs.StringVar(&c.AuthType, "observatorium-auth-type", c.AuthType, "Observatorium Authentication Type")
	fs.StringVar(&c.AuthTokenFile, "observatorium-token-file", c.AuthTokenFile, "Token File for Observatorium client")
	fs.DurationVar(&c.Timeout, "observatorium-timeout", c.Timeout, "Timeout for Observatorium client")
	fs.BoolVar(&c.Insecure, "observatorium-ignore-ssl", c.Insecure, "ignore SSL Observatorium certificate")
	fs.BoolVar(&c.EnableMock, "enable-observatorium-mock", c.EnableMock, "Enable mock Observatorium client")
	fs.BoolVar(&c.Debug, "observatorium-debug", c.Debug, "Debug flag for Observatorium client")

	fs.StringVar(&c.ObservabilityConfigRepo, "observability-config-repo", c.ObservabilityConfigRepo, "Repo for the observability operator configuration repo")
	fs.StringVar(&c.ObservabilityConfigChannel, "observability-config-channel", c.ObservabilityConfigChannel, "Channel for the observability operator configuration repo")
	fs.StringVar(&c.ObservabilityConfigTag, "observability-config-tag", c.ObservabilityConfigTag, "Tag or branch to use inside the observability configuration repo")

	fs.StringVar(&c.ObservabilityCloudWatchLoggingConfig.configFilePath, "observability-cloudwatchlogging-config-file-path", "secrets/observability-cloudwatchlogs-config.yaml", "Path to a file containing the configuration for Observability related to AWS CloudWatch Logging in YAML format. Only takes effect when --observability-cloudwatchlogging-enable is set")
	fs.BoolVar(&c.ObservabilityCloudWatchLoggingConfig.CloudwatchLoggingEnabled, "observability-cloudwatchlogging-enable", false, "Enable Observability to deliver data plane logs to AWS CloudWatch")

	fs.StringVar(&c.DataPlaneObservabilityConfig.configFilePath, "dataplane-observability-config", "secrets/dataplane-observability-config.yaml", "Path to a file containing the observability configuration for the data plane")
	fs.BoolVar(&c.DataPlaneObservabilityConfig.Enabled, "dataplane-observability-config-enabled", true, "Determines if the data plane observability config file must be provided")
}

func (c *ObservabilityConfiguration) ReadFiles() error {
	dexPassword, err := shared.ReadFile(c.DexPasswordFile)
	if err != nil {
		return err
	}
	dexSecret, err := shared.ReadFile(c.DexSecretFile)
	if err != nil {
		return err
	}
	c.DexPassword = dexPassword
	c.DexSecret = dexSecret

	if c.AuthToken == "" && c.AuthTokenFile != "" {
		err := shared.ReadFileValueString(c.AuthTokenFile, &c.AuthToken)
		if err != nil {
			return err
		}
	}

	err = c.ObservabilityCloudWatchLoggingConfig.readConfigFile()
	if err != nil {
		return err
	}

	err = c.DataPlaneObservabilityConfig.readConfigFile()
	if err != nil {
		return err
	}

	return nil
}

func (c *ObservabilityConfiguration) Validate(env *environments.Env) error {
	err := c.ObservabilityCloudWatchLoggingConfig.validate()
	if err != nil {
		return err
	}

	return c.DataPlaneObservabilityConfig.validate()
}
