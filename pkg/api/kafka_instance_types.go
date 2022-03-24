package api

type SupportedKafkaInstanceType struct {
	Id                  string               `json:"id"`
	SupportedKafkaSizes []SupportedKafkaSize `json:"sizes"`
}

type SupportedKafkaSize struct {
	Id                          string `json:"id"`
	IngressThroughputPerSec     string `json:"ingress_throughput_per_sec"`
	EgressThroughputPerSec      string `json:"egress_throughput_per_sec"`
	TotalMaxConnections         int32  `json:"total_max_connections"`
	MaxDataRetentionSize        string `json:"max_data_retention_size"`
	MaxPartitions               int32  `json:"max_partitions"`
	MaxDataRetentionPeriod      string `json:"max_data_retention_period"`
	MaxConnectionAttemptsPerSec int32  `json:"max_connection_attempts_per_sec"`
	QuotaConsumed               int32  `json:"quota_consumed"`
	QuotaType                   string `json:"quota_type"`
	CapacityConsumed            int32  `json:"capacity_consumed"`
}

type SupportedKafkaInstanceTypesList []SupportedKafkaInstanceType
