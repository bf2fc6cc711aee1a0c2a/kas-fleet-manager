package config

import "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

type KafkaQuotaConfig struct {
	Type                   string `json:"type"`
	AllowDeveloperInstance bool   `json:"allow_developer_instance"`
}

func NewKafkaQuotaConfig() *KafkaQuotaConfig {
	return &KafkaQuotaConfig{
		Type:                   api.QuotaManagementListQuotaType.String(),
		AllowDeveloperInstance: true,
	}
}
