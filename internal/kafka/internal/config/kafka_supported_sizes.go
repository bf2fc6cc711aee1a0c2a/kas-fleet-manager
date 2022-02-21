package config

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
