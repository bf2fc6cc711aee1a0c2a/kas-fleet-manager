package config

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/senseyeio/duration"
	"k8s.io/apimachinery/pkg/api/resource"
)

var validKafkaProfileIds = []string{
	"eval",
	"standard",
}

type KafkaProfile struct {
	Id    string              `yaml:"id"`
	Sizes []KafkaInstanceSize `yaml:"sizes"`
}

// validates kafka profile config to ensure the following:
// - id must be defined and included in the valid profile id list
// - sizes cannot be an empty list and each size id must be unique
func (kp *KafkaProfile) validate() error {
	if kp.Id == "" || len(kp.Sizes) == 0 {
		return fmt.Errorf("Kafka instance profile '%s' is missing required parameters.", kp.Id)
	}

	if !shared.Contains(validKafkaProfileIds, kp.Id) {
		return fmt.Errorf("kafka instance profile id '%s' is not valid. Valid kafka profiles are: '%v'", kp.Id, validKafkaProfileIds)
	}

	existingSizes := make(map[string]int, len(kp.Sizes))

	for _, kafkaInstanceSize := range kp.Sizes {
		if _, ok := existingSizes[kafkaInstanceSize.Id]; ok {
			return fmt.Errorf("Kafka instance size '%s' for profile '%s' was defined more than once.", kafkaInstanceSize.Id, kp.Id)
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
	Cost                        int    `yaml:"cost"`
}

// validates Kafka instance size configuration to ensure the following:
//
// - all properties must be defined
// - any non-id string values must be parseable
// - any int values must not be less than or equal to zero
func (k *KafkaInstanceSize) validate(profileId string) error {
	if k.EgressThroughputPerSec == "" || k.IngressThroughputPerSec == "" ||
		k.MaxDataRetentionPeriod == "" || k.MaxDataRetentionSize == "" || k.Id == "" {
		return fmt.Errorf("Kafka instance size '%s' for profile '%s' is missing required parameters.", k.Id, profileId)
	}

	egressThroughputQuantity, err := resource.ParseQuantity(k.EgressThroughputPerSec)
	if err != nil {
		return fmt.Errorf("egressThroughputPerSec for Kafka profile '%s', size '%s' is invalid: %s", k.Id, profileId, err.Error())
	}

	ingressThroughputQuantity, err := resource.ParseQuantity(k.IngressThroughputPerSec)
	if err != nil {
		return fmt.Errorf("ingressThroughputPerSec for Kafka profile '%s', size '%s' is invalid: %s", k.Id, profileId, err.Error())
	}

	maxDataRetentionSize, err := resource.ParseQuantity(k.MaxDataRetentionSize)
	if err != nil {
		return fmt.Errorf("maxDataRetentionSize for Kafka profile '%s', size '%s' is invalid: %s", k.Id, profileId, err.Error())
	}

	maxDataRetentionPeriod, err := duration.ParseISO8601(k.MaxDataRetentionPeriod)
	if err != nil {
		return fmt.Errorf("maxDataRetentionPeriod for Kafka profile '%s', size '%s' is invalid: %s", k.Id, profileId, err.Error())
	}

	if maxDataRetentionPeriod.IsZero() || egressThroughputQuantity.CmpInt64(1) < 0 ||
		ingressThroughputQuantity.CmpInt64(1) < 0 || maxDataRetentionSize.CmpInt64(1) < 0 ||
		k.TotalMaxConnections <= 0 || k.MaxPartitions <= 0 || k.MaxConnectionAttemptsPerSec <= 0 ||
		k.Cost < 1 {
		return fmt.Errorf("Kafka instance size '%s' for profile '%s' specifies a property value less than or equals to Zero.", k.Id, profileId)
	}

	return nil
}

type SupportedKafkaSizesConfig struct {
	SupportedKafkaProfiles []KafkaProfile `yaml:"supported_kafka_sizes"`
}

func (s *SupportedKafkaSizesConfig) validate() error {
	existingProfiles := make(map[string]int, len(s.SupportedKafkaProfiles))

	for _, kafkaProfile := range s.SupportedKafkaProfiles {
		if _, ok := existingProfiles[kafkaProfile.Id]; ok {
			return fmt.Errorf("Kafka instance profile id '%s' was defined more than once.", kafkaProfile.Id)
		}
		existingProfiles[kafkaProfile.Id]++

		if err := kafkaProfile.validate(); err != nil {
			return err
		}
	}

	return nil
}

type KafkaSupportedSizesConfig struct {
	SupportedKafkaSizesConfig     SupportedKafkaSizesConfig
	SupportedKafkaSizesConfigFile string
}

func NewKafkaSupportedSizesConfig() *KafkaSupportedSizesConfig {
	return &KafkaSupportedSizesConfig{
		SupportedKafkaSizesConfigFile: "config/kafka-sizes-configuration.yaml",
	}
}
