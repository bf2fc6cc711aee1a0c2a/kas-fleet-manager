package config

import "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

type KafkaQuotaConfig struct {
	Type                   string `json:"type"`
	AllowEvaluatorInstance bool   `json:"allow_evaluator_instance"`
}

func NewKafkaQuotaConfig() *KafkaQuotaConfig {
	return &KafkaQuotaConfig{
		Type:                   api.AllowListQuotaType.String(),
		AllowEvaluatorInstance: true,
	}
}
