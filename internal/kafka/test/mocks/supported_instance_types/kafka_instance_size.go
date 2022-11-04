package supportedinstancetypes

import "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"

const (
	DefaultKafkaInstanceSizeId          = "x1"
	DefaultKafkaInstanceSizeDisplayName = "x1"
	DefaultIngressThroughputPerSec      = "1Mi"
	DefaultEgressThroughputPerSec       = "1Mi"
	DefaultTotalMaxConnections          = 100
	DefaultMaxDataRetentionSize         = "1Gi"
	DefaultMaxDataRetentionPeriod       = "P14D"
	DefaultMaxConnectionAttemptsPerSec  = 100
	DefaultMaxMessageSize               = "1Mi"
	DefaultQuotaconsumed                = 1
	DefaultQuotaType                    = "RHOSAK"
	DefaultCapacityConsumed             = 1
	DefaultMinInSyncReplicas            = 1
	DefaultReplicationFactor            = 1
)

var (
	DefaultSupportedAZModes = []string{"single", "multi"}
	DefaultMaturityStatus   = config.MaturityStatusStable
)

var lifespanSeconds int = 172800
var DefaultLifespanSeconds *int = &lifespanSeconds

type KafkaInstanceSizeAttribute int

const (
	SIZE_ID KafkaInstanceSizeAttribute = iota
	DISPLAY_NAME
	INGRESS_THROUGHPUT_PER_SEC
	EGRESS_THROUGHPUT_PER_SEC
	MAX_DATA_RETENTION_SIZE
	MAX_DATA_RETENTION_PERIOD
	MAX_CONNECTION_ATTEMPTS_PER_SEC
	MAX_MESSAGE_SIZE
	QUOTA_TYPE
	MATURITY_STATUS
)

func With(attribute KafkaInstanceSizeAttribute, value string) KafkaInstanceSizeBuildOption {
	return func(kafkaInstanceSize *config.KafkaInstanceSize) {
		switch attribute {
		case SIZE_ID:
			kafkaInstanceSize.Id = value
		case DISPLAY_NAME:
			kafkaInstanceSize.DisplayName = value
		case INGRESS_THROUGHPUT_PER_SEC:
			kafkaInstanceSize.IngressThroughputPerSec = config.Quantity(value)
		case EGRESS_THROUGHPUT_PER_SEC:
			kafkaInstanceSize.EgressThroughputPerSec = config.Quantity(value)
		case MAX_DATA_RETENTION_SIZE:
			kafkaInstanceSize.MaxDataRetentionSize = config.Quantity(value)
		case MAX_DATA_RETENTION_PERIOD:
			kafkaInstanceSize.MaxDataRetentionPeriod = value
		case MAX_MESSAGE_SIZE:
			kafkaInstanceSize.MaxMessageSize = config.Quantity(value)
		case QUOTA_TYPE:
			kafkaInstanceSize.DeprecatedQuotaType = value
		case MATURITY_STATUS:
			kafkaInstanceSize.MaturityStatus = config.MaturityStatus(value)
		}
	}
}

func WithTotalMaxConnections(totalMaxConnections int) KafkaInstanceSizeBuildOption {
	return func(kafkaInstanceSize *config.KafkaInstanceSize) {
		kafkaInstanceSize.TotalMaxConnections = totalMaxConnections
	}
}

func WithQuotaConsumed(quotaConsumed int) KafkaInstanceSizeBuildOption {
	return func(kafkaInstanceSize *config.KafkaInstanceSize) {
		kafkaInstanceSize.QuotaConsumed = quotaConsumed
	}
}

func WithCapacityConsumed(capacityConsumed int) KafkaInstanceSizeBuildOption {
	return func(kafkaInstanceSize *config.KafkaInstanceSize) {
		kafkaInstanceSize.CapacityConsumed = capacityConsumed
	}
}

func WithSupportedAZModes(supportedAZModes []string) KafkaInstanceSizeBuildOption {
	return func(kafkaInstanceSize *config.KafkaInstanceSize) {
		kafkaInstanceSize.SupportedAZModes = supportedAZModes
	}
}

func WithMinInSyncReplicas(minInSyncReplicas int) KafkaInstanceSizeBuildOption {
	return func(kafkaInstanceSize *config.KafkaInstanceSize) {
		kafkaInstanceSize.MinInSyncReplicas = minInSyncReplicas
	}
}

func WithReplicationFactor(replicationFactor int) KafkaInstanceSizeBuildOption {
	return func(kafkaInstanceSize *config.KafkaInstanceSize) {
		kafkaInstanceSize.ReplicationFactor = replicationFactor
	}
}

func WithLifespanSeconds(lifespanSeconds *int) KafkaInstanceSizeBuildOption {
	return func(kafkaInstanceSize *config.KafkaInstanceSize) {
		kafkaInstanceSize.LifespanSeconds = lifespanSeconds
	}
}

type KafkaInstanceSizeBuildOption func(*config.KafkaInstanceSize)

func BuildKafkaInstanceSize(options ...KafkaInstanceSizeBuildOption) *config.KafkaInstanceSize {
	kafkaInstanceSize := &config.KafkaInstanceSize{
		Id:                          DefaultKafkaInstanceSizeId,
		DisplayName:                 DefaultKafkaInstanceSizeDisplayName,
		IngressThroughputPerSec:     DefaultIngressThroughputPerSec,
		EgressThroughputPerSec:      DefaultEgressThroughputPerSec,
		TotalMaxConnections:         DefaultTotalMaxConnections,
		MaxDataRetentionSize:        DefaultMaxDataRetentionSize,
		MaxDataRetentionPeriod:      DefaultMaxDataRetentionPeriod,
		MaxConnectionAttemptsPerSec: DefaultMaxConnectionAttemptsPerSec,
		MaxMessageSize:              DefaultMaxMessageSize,
		QuotaConsumed:               DefaultQuotaconsumed,
		DeprecatedQuotaType:         DefaultQuotaType,
		CapacityConsumed:            DefaultCapacityConsumed,
		SupportedAZModes:            DefaultSupportedAZModes,
		MinInSyncReplicas:           DefaultMinInSyncReplicas,
		ReplicationFactor:           DefaultReplicationFactor,
		LifespanSeconds:             DefaultLifespanSeconds,
		MaturityStatus:              DefaultMaturityStatus,
	}

	for _, option := range options {
		option(kafkaInstanceSize)
	}
	return kafkaInstanceSize
}
