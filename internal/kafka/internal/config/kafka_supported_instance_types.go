package config

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/senseyeio/duration"
	"k8s.io/apimachinery/pkg/api/resource"
)

var validKafkaInstanceTypeIds = []string{
	"eval",
	"standard",
}

type KafkaInstanceType struct {
	Id    string              `yaml:"id"`
	Sizes []KafkaInstanceSize `yaml:"sizes"`
}

// validates kafka instance type config to ensure the following:
// - id must be defined and included in the valid instance type id list
// - sizes cannot be an empty list and each size id must be unique
func (kp *KafkaInstanceType) validate() error {
	if kp.Id == "" || len(kp.Sizes) == 0 {
		return fmt.Errorf("Kafka instance type '%s' is missing required parameters.", kp.Id)
	}

	if !shared.Contains(validKafkaInstanceTypeIds, kp.Id) {
		return fmt.Errorf("kafka instance type id '%s' is not valid. Valid kafka instance types are: '%v'", kp.Id, validKafkaInstanceTypeIds)
	}

	existingSizes := make(map[string]int, len(kp.Sizes))

	for _, kafkaInstanceSize := range kp.Sizes {
		if _, ok := existingSizes[kafkaInstanceSize.Id]; ok {
			return fmt.Errorf("Kafka instance size '%s' for instance type '%s' was defined more than once.", kafkaInstanceSize.Id, kp.Id)
		}
		existingSizes[kafkaInstanceSize.Id]++

		if err := kafkaInstanceSize.validate(kp.Id); err != nil {
			return err
		}
	}

	return nil
}

type KafkaInstanceSize struct {
	Id                          string `yaml:"id"`
	IngressThroughputPerSec     string `yaml:"ingressThroughputPerSec"`
	EgressThroughputPerSec      string `yaml:"egressThroughputPerSec"`
	TotalMaxConnections         int    `yaml:"totalMaxConnections"`
	MaxDataRetentionSize        string `yaml:"maxDataRetentionSize"`
	MaxPartitions               int    `yaml:"maxPartitions"`
	MaxDataRetentionPeriod      string `yaml:"maxDataRetentionPeriod"`
	MaxConnectionAttemptsPerSec int    `yaml:"maxConnectionAttemptsPerSec"`
	QuotaConsumed               int    `yaml:"quotaConsumed"`
	QuotaType                   string `yaml:"quotaType"`
}

// validates Kafka instance size configuration to ensure the following:
//
// - all properties must be defined
// - any non-id string values must be parseable
// - any int values must not be less than or equal to zero
func (k *KafkaInstanceSize) validate(instanceTypeId string) error {
	if k.EgressThroughputPerSec == "" || k.IngressThroughputPerSec == "" ||
		k.MaxDataRetentionPeriod == "" || k.MaxDataRetentionSize == "" || k.Id == "" || k.QuotaType == "" {
		return fmt.Errorf("Kafka instance size '%s' for instance type '%s' is missing required parameters.", k.Id, instanceTypeId)
	}

	egressThroughputQuantity, err := resource.ParseQuantity(k.EgressThroughputPerSec)
	if err != nil {
		return fmt.Errorf("egressThroughputPerSec for Kafka instance type '%s', size '%s' is invalid: %s", k.Id, instanceTypeId, err.Error())
	}

	ingressThroughputQuantity, err := resource.ParseQuantity(k.IngressThroughputPerSec)
	if err != nil {
		return fmt.Errorf("ingressThroughputPerSec for Kafka instance type '%s', size '%s' is invalid: %s", k.Id, instanceTypeId, err.Error())
	}

	maxDataRetentionSize, err := resource.ParseQuantity(k.MaxDataRetentionSize)
	if err != nil {
		return fmt.Errorf("maxDataRetentionSize for Kafka instance type '%s', size '%s' is invalid: %s", k.Id, instanceTypeId, err.Error())
	}

	maxDataRetentionPeriod, err := duration.ParseISO8601(k.MaxDataRetentionPeriod)
	if err != nil {
		return fmt.Errorf("maxDataRetentionPeriod for Kafka instance type '%s', size '%s' is invalid: %s", k.Id, instanceTypeId, err.Error())
	}

	if maxDataRetentionPeriod.IsZero() || egressThroughputQuantity.CmpInt64(1) < 0 ||
		ingressThroughputQuantity.CmpInt64(1) < 0 || maxDataRetentionSize.CmpInt64(1) < 0 ||
		k.TotalMaxConnections <= 0 || k.MaxPartitions <= 0 || k.MaxConnectionAttemptsPerSec <= 0 ||
		k.QuotaConsumed < 1 {
		return fmt.Errorf("Kafka instance size '%s' for instance type '%s' specifies a property value less than or equals to Zero.", k.Id, instanceTypeId)
	}

	return nil
}

type SupportedKafkaInstanceTypesConfig struct {
	SupportedKafkaInstanceTypes []KafkaInstanceType `yaml:"supported_instance_types"`
}

func (s *SupportedKafkaInstanceTypesConfig) validate() error {
	existingInstanceTypes := make(map[string]int, len(s.SupportedKafkaInstanceTypes))

	for _, KafkaInstanceType := range s.SupportedKafkaInstanceTypes {
		if _, ok := existingInstanceTypes[KafkaInstanceType.Id]; ok {
			return fmt.Errorf("Kafka instance type id '%s' was defined more than once.", KafkaInstanceType.Id)
		}
		existingInstanceTypes[KafkaInstanceType.Id]++

		if err := KafkaInstanceType.validate(); err != nil {
			return err
		}
	}

	return nil
}

type KafkaSupportedInstanceTypesConfig struct {
	Configuration     SupportedKafkaInstanceTypesConfig
	ConfigurationFile string
}

func NewKafkaSupportedInstanceTypesConfig() *KafkaSupportedInstanceTypesConfig {
	return &KafkaSupportedInstanceTypesConfig{
		ConfigurationFile: "config/kafka-instance-types-configuration.yaml",
	}
}
