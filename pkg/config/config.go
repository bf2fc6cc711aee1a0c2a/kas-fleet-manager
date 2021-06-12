package config

import (
	"flag"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/common"
	"github.com/spf13/pflag"
)

type ConfigModule interface {
	AddFlags(fs *pflag.FlagSet)
	ReadFiles() error
}

type ServiceInjector interface {
	Injections() (common.InjectionMap, error)
}

type ApplicationConfig struct {
	Server                     *ServerConfig               `json:"server"`
	Metrics                    *MetricsConfig              `json:"metrics"`
	HealthCheck                *HealthCheckConfig          `json:"health_check"`
	Database                   *DatabaseConfig             `json:"database"`
	OCM                        *OCMConfig                  `json:"ocm"`
	Sentry                     *SentryConfig               `json:"sentry"`
	AWS                        *AWSConfig                  `json:"aws"`
	SupportedProviders         *ProviderConfig             `json:"providers"`
	AccessControlList          *AccessControlListConfig    `json:"allow_list"`
	ObservabilityConfiguration *ObservabilityConfiguration `json:"observability"`
	Keycloak                   *KeycloakConfig             `json:"keycloak"`
	Kafka                      *KafkaConfig                `json:"kafka_tls"`
	OSDClusterConfig           *OSDClusterConfig           `json:"osd_cluster"`
	KasFleetShardConfig        *KasFleetshardConfig        `json:"kas-fleetshard"`
}

var _ ConfigModule = &ApplicationConfig{}

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
		AccessControlList:          NewAccessControlListConfig(),
		ObservabilityConfiguration: NewObservabilityConfigurationConfig(),
		Keycloak:                   NewKeycloakConfig(),
		Kafka:                      NewKafkaConfig(),
		OSDClusterConfig:           NewOSDClusterConfig(),
		KasFleetShardConfig:        NewKasFleetshardConfig(),
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
	c.AccessControlList.AddFlags(flagset)
	c.ObservabilityConfiguration.AddFlags(flagset)
	c.Keycloak.AddFlags(flagset)
	c.Kafka.AddFlags(flagset)
	c.OSDClusterConfig.AddFlags(flagset)
	c.KasFleetShardConfig.AddFlags(flagset)
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
	err = c.AccessControlList.ReadFiles()
	if err != nil {
		return err
	}
	err = c.Kafka.ReadFiles()
	if err != nil {
		return err
	}
	err = c.OSDClusterConfig.ReadFiles()
	if err != nil {
		return err
	}
	err = c.KasFleetShardConfig.ReadFiles()
	if err != nil {
		return err
	}
	return nil
}
