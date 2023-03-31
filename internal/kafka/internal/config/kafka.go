package config

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/spf13/pflag"
)

type KafkaConfig struct {
	EnableKafkaCNAMERegistration bool
	KafkaDomainName              string
	BrowserUrl                   string

	Quota                  *KafkaQuotaConfig
	SupportedInstanceTypes *KafkaSupportedInstanceTypesConfig
	EnableKafkaOwnerConfig bool
	KafkaOwnerList         []string
	KafkaOwnerListFile     string
}

func NewKafkaConfig() *KafkaConfig {
	return &KafkaConfig{
		EnableKafkaCNAMERegistration: false,
		EnableKafkaOwnerConfig:       false,
		KafkaDomainName:              "kafka.bf2.dev",
		Quota:                        NewKafkaQuotaConfig(),
		SupportedInstanceTypes:       NewKafkaSupportedInstanceTypesConfig(),
		KafkaOwnerListFile:           "config/kafka-owner-list.yaml",
		BrowserUrl:                   "http://localhost:8080/",
	}
}

func (c *KafkaConfig) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&c.EnableKafkaCNAMERegistration, "enable-kafka-cname-registration", c.EnableKafkaCNAMERegistration, "Enable custom CNAME registration for Kafka instances")
	fs.StringVar(&c.KafkaDomainName, "kafka-domain-name", c.KafkaDomainName, "The domain name to use for Kafka instances")
	fs.StringVar(&c.Quota.Type, "quota-type", c.Quota.Type, "The type of the quota service to be used. The available options are: 'ams' for AMS backed implementation and 'quota-management-list' for quota list backed implementation (default).")
	fs.BoolVar(&c.Quota.AllowDeveloperInstance, "allow-developer-instance", c.Quota.AllowDeveloperInstance, "Allow the creation of kafka developer instances")
	fs.StringVar(&c.SupportedInstanceTypes.ConfigurationFile, "supported-kafka-instance-types-config-file", c.SupportedInstanceTypes.ConfigurationFile, "File containing the supported instance types configuration")
	fs.StringVar(&c.BrowserUrl, "browser-url", c.BrowserUrl, "Browser url to kafka admin UI")
	fs.BoolVar(&c.EnableKafkaOwnerConfig, "enable-kafka-owner-config", c.EnableKafkaOwnerConfig, "Enable configuration for setting kafka owners")
	fs.StringVar(&c.KafkaOwnerListFile, "kafka-owner-list-file", c.KafkaOwnerListFile, "File containing list of kafka owners")
	fs.IntVar(&c.Quota.MaxAllowedDeveloperInstances, "max-allowed-developer-instances", c.Quota.MaxAllowedDeveloperInstances, "As a user, one can create up to N defined max developer instances if they do not have quota to create standard instances")
}

func (c *KafkaConfig) ReadFiles() error {
	err := shared.ReadYamlFile(c.SupportedInstanceTypes.ConfigurationFile, &c.SupportedInstanceTypes.Configuration)
	if err != nil {
		return err
	}

	if c.EnableKafkaOwnerConfig {
		err = shared.ReadYamlFile(c.KafkaOwnerListFile, &c.KafkaOwnerList)
		if err != nil {
			return err
		}
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
		return nil, errors.New(errors.ErrorGeneral, fmt.Sprintf("unable to get size for instance type: '%s'", instanceType))
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

func (c *KafkaConfig) GetBillingModels(instanceType string) ([]KafkaBillingModel, error) {
	kafkaInstanceType, err := c.SupportedInstanceTypes.Configuration.GetKafkaInstanceTypeByID(instanceType)
	if err != nil {
		return nil, err
	}

	return kafkaInstanceType.SupportedBillingModels, nil
}

func (c *KafkaConfig) GetBillingModelByID(instanceType, billingModelID string) (KafkaBillingModel, error) {
	kafkaInstanceType, err := c.SupportedInstanceTypes.Configuration.GetKafkaInstanceTypeByID(instanceType)
	if err != nil {
		return KafkaBillingModel{}, err
	}

	billingModel, err := kafkaInstanceType.GetKafkaSupportedBillingModelByID(billingModelID)

	if err != nil {
		return KafkaBillingModel{}, err
	}

	return *billingModel, nil
}
