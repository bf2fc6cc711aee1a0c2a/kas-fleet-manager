package config

import (
	"fmt"

	"github.com/senseyeio/duration"
	"k8s.io/apimachinery/pkg/api/resource"
)

type KafkaInstanceSize struct {
	Size                        string `yaml:"size"`
	IngressThroughputPerSec     string `yaml:"ingressThroughputPerSec"`
	EgressThroughputPerSec      string `yaml:"egressThroughputPerSec"`
	TotalMaxConnections         int    `yaml:"totalMaxConnections"`
	MaxDataRetentionSize        string `yaml:"maxDataRetentionSize"`
	MaxPartitions               int    `yaml:"maxPartitions"`
	MaxDataRetentionPeriod      string `yaml:"maxDataRetentionPeriod"`
	MaxConnectionAttemptsPerSec int    `yaml:"maxConnectionAttemptsPerSec"`
}

type SupportedKafkaSizesConfig struct {
	SupportedKafkaSizes []KafkaInstanceSize `yaml:"supported_kafka_sizes"`
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

func (k *KafkaSupportedSizesConfig) Validate() error {
	supportedKafkaSizes := k.SupportedKafkaSizesConfig.SupportedKafkaSizes

	existingSizes := make(map[string]int, len(supportedKafkaSizes))

	for _, kafkaInstanceSize := range supportedKafkaSizes {
		if kafkaInstanceSize.EgressThroughputPerSec == "" || kafkaInstanceSize.IngressThroughputPerSec == "" ||
			kafkaInstanceSize.MaxDataRetentionPeriod == "" || kafkaInstanceSize.MaxDataRetentionSize == "" || kafkaInstanceSize.Size == "" {
			return fmt.Errorf("Kafka instance size '%s' is missing required parameters.", kafkaInstanceSize.Size)
		}

		if _, ok := existingSizes[kafkaInstanceSize.Size]; ok {
			return fmt.Errorf("Kafka instance size '%s' was defined more than once.", kafkaInstanceSize.Size)
		}
		existingSizes[kafkaInstanceSize.Size]++

		egressThroughputQuantity, err := resource.ParseQuantity(kafkaInstanceSize.EgressThroughputPerSec)
		if err != nil {
			return fmt.Errorf("egressThroughputPerSec: %s", err.Error())
		}

		ingressThroughputQuantity, err := resource.ParseQuantity(kafkaInstanceSize.IngressThroughputPerSec)
		if err != nil {
			return fmt.Errorf("ingressThroughputPerSec: %s", err.Error())
		}

		maxDataRetentionSize, err := resource.ParseQuantity(kafkaInstanceSize.MaxDataRetentionSize)
		if err != nil {
			return fmt.Errorf("maxDataRetentionSize: %s", err.Error())
		}

		maxDataRetentionPeriod, err := duration.ParseISO8601(kafkaInstanceSize.MaxDataRetentionPeriod)
		if err != nil {
			return fmt.Errorf("maxDataRetentionPeriod: %s", err.Error())
		}

		if maxDataRetentionPeriod.IsZero() || egressThroughputQuantity.CmpInt64(1) < 0 ||
			ingressThroughputQuantity.CmpInt64(1) < 0 || maxDataRetentionSize.CmpInt64(1) < 0 ||
			kafkaInstanceSize.TotalMaxConnections <= 0 || kafkaInstanceSize.MaxPartitions <= 0 || kafkaInstanceSize.MaxConnectionAttemptsPerSec <= 0 {
			return fmt.Errorf("Kafka instance size '%s' specifies a property value less than or equals to Zero.", kafkaInstanceSize.Size)
		}
	}

	return nil
}
