package config

type KafkaProfile struct {
	Id    string              `yaml:"id"`
	Sizes []KafkaInstanceSize `yaml:"sizes"`
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

type SupportedKafkaSizesConfig struct {
	SupportedKafkaProfiles []KafkaProfile `yaml:"supported_kafka_sizes"`
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
