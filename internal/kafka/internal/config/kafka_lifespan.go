package config

type KafkaLifespanConfig struct {
	EnableDeletionOfExpiredKafka bool
	KafkaLifespanInHours         int
}

func NewKafkaLifespanConfig() *KafkaLifespanConfig {
	return &KafkaLifespanConfig{
		EnableDeletionOfExpiredKafka: true,
		KafkaLifespanInHours:         48,
	}
}
