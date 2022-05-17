package mocks

import "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"

const (
	DefaultKafkaInstanceSizeId         = "x1"
	DefaultIngressThroughputPerSec     = "1Mi"
	DefaultEgressThroughputPerSec      = "1Mi"
	DefaultTotalMaxConnections         = 100
	DefaultMaxDataRetentionSize        = "1Gi"
	DefaultMaxDataRetentionPeriod      = "P14D"
	DefaultMaxConnectionAttemptsPerSec = 100
	DefaultQuotaconsumed               = 1
	DefaultQuotaType                   = "RHOSAK"
	DefaultCapacityConsumed            = 1
)

func BuildKafkaInstanceSize(modifyFn func(kafkaInstanceSize *config.KafkaInstanceSize)) *config.KafkaInstanceSize {
	kafkaInstanceSize := &config.KafkaInstanceSize{
		Id:                          DefaultKafkaInstanceSizeId,
		IngressThroughputPerSec:     DefaultIngressThroughputPerSec,
		EgressThroughputPerSec:      DefaultEgressThroughputPerSec,
		TotalMaxConnections:         DefaultTotalMaxConnections,
		MaxDataRetentionSize:        DefaultMaxDataRetentionSize,
		MaxDataRetentionPeriod:      DefaultMaxDataRetentionPeriod,
		MaxConnectionAttemptsPerSec: DefaultMaxConnectionAttemptsPerSec,
		QuotaConsumed:               DefaultQuotaconsumed,
		QuotaType:                   DefaultQuotaType,
		CapacityConsumed:            DefaultCapacityConsumed,
	}
	if modifyFn != nil {
		modifyFn(kafkaInstanceSize)
	}
	return kafkaInstanceSize
}
