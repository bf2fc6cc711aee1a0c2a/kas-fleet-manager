package config

import (
	"flag"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"

	"github.com/spf13/pflag"
)

var projectRootDirectory = shared.GetProjectRootDir()

type ApplicationConfig struct {
	Server                     *ServerConfig               `json:"server"`
	Metrics                    *MetricsConfig              `json:"metrics"`
	HealthCheck                *HealthCheckConfig          `json:"health_check"`
	Database                   *DatabaseConfig             `json:"database"`
	OCM                        *OCMConfig                  `json:"ocm"`
	Sentry                     *SentryConfig               `json:"sentry"`
	AWS                        *AWSConfig                  `json:"aws"`
	SupportedProviders         *ProviderConfig             `json:"providers"`
	AllowList                  *AllowListConfig            `json:"allow_list"`
	ObservabilityConfiguration *ObservabilityConfiguration `json:"observability"`
	Keycloak                   *KeycloakConfig             `json:"keycloak"`
	Kafka                      *KafkaConfig                `json:"kafka_tls"`
	ClusterCreationConfig      *ClusterCreationConfig      `json:"cluster_creation"`
	ConnectorsConfig           *ConnectorsConfig           `json:"connectors"`
	KasFleetShardConfig        *KasFleetshardConfig        `json:"kas-fleetshard"`
	Vault                      *VaultConfig                `json:"vault"`
}

func NewApplicationConfig() *ApplicationConfig {
	return &ApplicationConfig{
		Server:                     NewServerConfig(),
		Metrics:                    NewMetricsConfig(),
		HealthCheck:                NewHealthCheckConfig(),
		Database:                   NewDatabaseConfig(),
		OCM:                        NewOCMConfig(),
		Sentry:                     NewSentryConfig(),
		AWS:                        NewAWSConfig(),
		SupportedProviders:         NewSupportedProvidersConfig(),
		AllowList:                  NewAllowListConfig(),
		ObservabilityConfiguration: NewObservabilityConfigurationConfig(),
		Keycloak:                   NewKeycloakConfig(),
		Kafka:                      NewKafkaConfig(),
		ClusterCreationConfig:      NewClusterCreationConfig(),
		ConnectorsConfig:           NewConnectorsConfig(),
		KasFleetShardConfig:        NewKasFleetshardConfig(),
		Vault:                      NewVaultConfig(),
	}
}

func (c *ApplicationConfig) AddFlags(flagset *pflag.FlagSet) {
	flagset.AddGoFlagSet(flag.CommandLine)
	c.Server.AddFlags(flagset)
	c.Metrics.AddFlags(flagset)
	c.HealthCheck.AddFlags(flagset)
	c.Database.AddFlags(flagset)
	c.OCM.AddFlags(flagset)
	c.Sentry.AddFlags(flagset)
	c.AWS.AddFlags(flagset)
	c.SupportedProviders.AddFlags(flagset)
	c.AllowList.AddFlags(flagset)
	c.ObservabilityConfiguration.AddFlags(flagset)
	c.Keycloak.AddFlags(flagset)
	c.Kafka.AddFlags(flagset)
	c.ClusterCreationConfig.AddFlags(flagset)
	c.ConnectorsConfig.AddFlags(flagset)
	c.KasFleetShardConfig.AddFlags(flagset)
	c.Vault.AddFlags(flagset)
}

func (c *ApplicationConfig) ReadFiles() error {
	err := c.Server.ReadFiles()
	if err != nil {
		return err
	}
	err = c.Metrics.ReadFiles()
	if err != nil {
		return err
	}
	err = c.HealthCheck.ReadFiles()
	if err != nil {
		return err
	}
	err = c.Database.ReadFiles()
	if err != nil {
		return err
	}
	err = c.OCM.ReadFiles()
	if err != nil {
		return err
	}
	err = c.Sentry.ReadFiles()
	if err != nil {
		return err
	}
	err = c.AWS.ReadFiles()
	if err != nil {
		return err
	}
	err = c.SupportedProviders.ReadFiles()
	if err != nil {
		return err
	}
	err = c.ObservabilityConfiguration.ReadFiles()
	if err != nil {
		return err
	}
	err = c.Keycloak.ReadFiles()
	if err != nil {
		return err
	}
	if c.AllowList.EnableAllowList {
		err = c.AllowList.ReadFiles()
		if err != nil {
			return err
		}
	}
	err = c.Kafka.ReadFiles()
	if err != nil {
		return err
	}
	if c.ConnectorsConfig.Enabled {
		err = c.ConnectorsConfig.ReadFiles()
		if err != nil {
			return err
		}
	}
	err = c.ClusterCreationConfig.ReadFiles()
	if err != nil {
		return err
	}
	err = c.Vault.ReadFiles()
	if err != nil {
		return err
	}
	err = c.KasFleetShardConfig.ReadFiles()
	if err != nil {
		return err
	}

	return nil
}

// Read the contents of file into integer value
func readFileValueInt(file string, val *int) error {
	fileContents, err := readFile(file)
	if err != nil {
		return err
	}

	*val, err = strconv.Atoi(fileContents)
	return err
}

// Read the contents of file into string value
func readFileValueString(file string, val *string) error {
	fileContents, err := readFile(file)
	if err != nil {
		return err
	}

	*val = strings.TrimSuffix(fileContents, "\n")
	return err
}

// Read the contents of file into boolean value
func readFileValueBool(file string, val *bool) error {
	fileContents, err := readFile(file)
	if err != nil {
		return err
	}

	*val, err = strconv.ParseBool(fileContents)
	return err
}

func readFile(file string) (string, error) {
	absFilePath := buildFullFilePath(file)

	// If no file is provided then we don't try to read it
	if absFilePath == "" {
		return "", nil
	}

	// Read the file
	buf, err := ioutil.ReadFile(absFilePath)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func buildFullFilePath(filename string) string {
	// If the value is in quotes, unquote it
	unquotedFile, err := strconv.Unquote(filename)
	if err != nil {
		// values without quotes will raise an error, ignore it.
		unquotedFile = filename
	}

	// If no file is provided, leave val unchanged.
	if unquotedFile == "" {
		return ""
	}

	// Ensure the absolute file path is used
	absFilePath := unquotedFile
	if !filepath.IsAbs(unquotedFile) {
		absFilePath = filepath.Join(projectRootDirectory, unquotedFile)
	}
	return absFilePath
}
