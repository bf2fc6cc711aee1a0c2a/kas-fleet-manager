package config

type KafkaLifespanConfig struct {
	EnableDeletionOfExpiredKafka bool
}

func NewKafkaLifespanConfig() *KafkaLifespanConfig {
	return &KafkaLifespanConfig{
		EnableDeletionOfExpiredKafka: true,
	}
}
