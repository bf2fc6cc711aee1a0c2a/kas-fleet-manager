package config

import (
	"flag"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/provider"
	"github.com/spf13/pflag"
)

type ApplicationConfig struct {
	AWS                        *AWSConfig                  `json:"aws"`
	SupportedProviders         *ProviderConfig             `json:"providers"`
	AccessControlList          *AccessControlListConfig    `json:"allow_list"`
	ObservabilityConfiguration *ObservabilityConfiguration `json:"observability"`
	Keycloak                   *KeycloakConfig             `json:"keycloak"`
	Kafka                      *KafkaConfig                `json:"kafka_tls"`
	DataplaneClusterConfig     *DataplaneClusterConfig     `json:"osd_cluster"`
}

var _ provider.ConfigModule = &ApplicationConfig{}

func NewApplicationConfig() *ApplicationConfig {
	return &ApplicationConfig{
		AWS:                        NewAWSConfig(),
		SupportedProviders:         NewSupportedProvidersConfig(),
		AccessControlList:          NewAccessControlListConfig(),
		ObservabilityConfiguration: NewObservabilityConfigurationConfig(),
		Keycloak:                   NewKeycloakConfig(),
		Kafka:                      NewKafkaConfig(),
		DataplaneClusterConfig:     NewDataplaneClusterConfig(),
	}
}

func (c *ApplicationConfig) AddFlags(flagset *pflag.FlagSet) {
	flagset.AddGoFlagSet(flag.CommandLine)
	c.AWS.AddFlags(flagset)
	c.SupportedProviders.AddFlags(flagset)
	c.AccessControlList.AddFlags(flagset)
	c.ObservabilityConfiguration.AddFlags(flagset)
	c.Keycloak.AddFlags(flagset)
	c.Kafka.AddFlags(flagset)
	c.DataplaneClusterConfig.AddFlags(flagset)
}

func (c *ApplicationConfig) ReadFiles() (err error) {
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
	err = c.DataplaneClusterConfig.ReadFiles()
	if err != nil {
		return err
	}
	return
}
