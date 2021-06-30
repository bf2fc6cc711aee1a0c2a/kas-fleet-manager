package config

import (
	"github.com/goava/di"
)

type ApplicationConfig struct {
	di.Inject
	AWS                        *AWSConfig
	SupportedProviders         *ProviderConfig
	AccessControlList          *AccessControlListConfig
	ObservabilityConfiguration *ObservabilityConfiguration
	Keycloak                   *KeycloakConfig
	Kafka                      *KafkaConfig
	DataplaneClusterConfig     *DataplaneClusterConfig
}

func NewApplicationConfig(config ApplicationConfig) *ApplicationConfig {
	return &config
}
