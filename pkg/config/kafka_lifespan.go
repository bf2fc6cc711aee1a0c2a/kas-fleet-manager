package config

import (
	"gopkg.in/yaml.v2"
)

type LongLivedKafas []string

type KafkaLifespanConfig struct {
	LongLivedKafkas              LongLivedKafas
	LongLivedKafkaConfigFile     string
	EnableDeletionOfExpiredKafka bool
	KafkaLifespanInHours         int
}

func NewKafkaLifespanConfig() *KafkaLifespanConfig {
	return &KafkaLifespanConfig{
		LongLivedKafkaConfigFile:     "config/long-lived-kafkas-configuration.yaml",
		EnableDeletionOfExpiredKafka: true,
		KafkaLifespanInHours:         48,
	}
}

func (c *KafkaLifespanConfig) ReadFiles() error {
	if c.EnableDeletionOfExpiredKafka {
		return readLongLivedKafkasFile(c.LongLivedKafkaConfigFile, &c.LongLivedKafkas)
	}
	return nil
}

func readLongLivedKafkasFile(file string, longLivedKafkas *LongLivedKafas) error {
	fileContents, err := readFile(file)
	if err != nil {
		return err
	}

	return yaml.Unmarshal([]byte(fileContents), longLivedKafkas)
}
