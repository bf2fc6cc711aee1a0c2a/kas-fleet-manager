package config

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/spf13/pflag"
)

type KafkaConfig struct {
	KafkaTLSCert                   string
	KafkaTLSCertFile               string
	KafkaTLSKey                    string
	KafkaTLSKeyFile                string
	EnableKafkaExternalCertificate bool
	EnableKafkaCNAMERegistration   bool
	KafkaDomainName                string
	BrowserUrl                     string

	KafkaLifespan          *KafkaLifespanConfig
	Quota                  *KafkaQuotaConfig
	SupportedInstanceTypes *KafkaSupportedInstanceTypesConfig
	EnableKafkaOwnerConfig bool
	KafkaOwnerList         []string
	KafkaOwnerListFile     string
}

func NewKafkaConfig() *KafkaConfig {
	return &KafkaConfig{
		KafkaTLSCertFile:               "secrets/kafka-tls.crt",
		KafkaTLSKeyFile:                "secrets/kafka-tls.key",
		EnableKafkaExternalCertificate: false,
		EnableKafkaCNAMERegistration:   false,
		EnableKafkaOwnerConfig:         false,
		KafkaDomainName:                "kafka.bf2.dev",
		KafkaLifespan:                  NewKafkaLifespanConfig(),
		Quota:                          NewKafkaQuotaConfig(),
		SupportedInstanceTypes:         NewKafkaSupportedInstanceTypesConfig(),
		KafkaOwnerListFile:             "config/kafka-owner-list.yaml",
		BrowserUrl:                     "http://localhost:8080/",
	}
}

func (c *KafkaConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.KafkaTLSCertFile, "kafka-tls-cert-file", c.KafkaTLSCertFile, "File containing kafka certificate")
	fs.StringVar(&c.KafkaTLSKeyFile, "kafka-tls-key-file", c.KafkaTLSKeyFile, "File containing kafka certificate private key")
	fs.BoolVar(&c.EnableKafkaExternalCertificate, "enable-kafka-external-certificate", c.EnableKafkaExternalCertificate, "Enable custom certificate for Kafka TLS")
	fs.BoolVar(&c.EnableKafkaCNAMERegistration, "enable-kafka-cname-registration", c.EnableKafkaCNAMERegistration, "Enable custom CNAME registration for Kafka instances")
	fs.BoolVar(&c.KafkaLifespan.EnableDeletionOfExpiredKafka, "enable-deletion-of-expired-kafka", c.KafkaLifespan.EnableDeletionOfExpiredKafka, "Enable the deletion of kafkas when its life span has expired")
	fs.StringVar(&c.KafkaDomainName, "kafka-domain-name", c.KafkaDomainName, "The domain name to use for Kafka instances")
	fs.StringVar(&c.Quota.Type, "quota-type", c.Quota.Type, "The type of the quota service to be used. The available options are: 'ams' for AMS backed implementation and 'quota-management-list' for quota list backed implementation (default).")
	fs.BoolVar(&c.Quota.AllowDeveloperInstance, "allow-developer-instance", c.Quota.AllowDeveloperInstance, "Allow the creation of kafka developer instances")
	fs.StringVar(&c.SupportedInstanceTypes.ConfigurationFile, "supported-kafka-instance-types-config-file", c.SupportedInstanceTypes.ConfigurationFile, "File containing the supported instance types configuration")
	fs.StringVar(&c.BrowserUrl, "browser-url", c.BrowserUrl, "Browser url to kafka admin UI")
	fs.BoolVar(&c.EnableKafkaOwnerConfig, "enable-kafka-owner-config", c.EnableKafkaOwnerConfig, "Enable configuration for setting kafka owners")
	fs.StringVar(&c.KafkaOwnerListFile, "kafka-owner-list-file", c.KafkaOwnerListFile, "File containing list of kafka owners")
}

func (c *KafkaConfig) ReadFiles() error {
	err := shared.ReadFileValueString(c.KafkaTLSCertFile, &c.KafkaTLSCert)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(c.KafkaTLSKeyFile, &c.KafkaTLSKey)
	if err != nil {
		return err
	}
	if c.EnableKafkaOwnerConfig {
		err = shared.ReadYamlFile(c.KafkaOwnerListFile, &c.KafkaOwnerList)
		if err != nil {
			return err
		}
	}

	err = shared.ReadYamlFile(c.SupportedInstanceTypes.ConfigurationFile, &c.SupportedInstanceTypes.Configuration)
	if err != nil {
		return err
	}

	return nil
}

func (c *KafkaConfig) Validate(env *environments.Env) error {
	return c.SupportedInstanceTypes.Configuration.validate()
}

func (c *KafkaConfig) GetFirstAvailableSize(instanceType string) (*KafkaInstanceSize, error) {
	kafkaInstanceType, err := c.SupportedInstanceTypes.Configuration.GetKafkaInstanceTypeByID(instanceType)
	if err != nil {
		return nil, err
	}
	if len(kafkaInstanceType.Sizes) < 1 || kafkaInstanceType.Sizes[0].Id == "" {
		return nil, errors.New(errors.ErrorGeneral, fmt.Sprintf("Unable to get size for instance type: '%s'", instanceType))
	}
	return &kafkaInstanceType.Sizes[0], nil
}

func (c *KafkaConfig) GetKafkaInstanceSize(instanceType, sizeId string) (*KafkaInstanceSize, error) {
	kafkaInstanceType, err := c.SupportedInstanceTypes.Configuration.GetKafkaInstanceTypeByID(instanceType)
	if err != nil {
		return nil, err
	}
	return kafkaInstanceType.GetKafkaInstanceSizeByID(sizeId)
}
