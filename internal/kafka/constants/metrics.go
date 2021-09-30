package constants

import "github.com/prometheus/client_golang/prometheus"

const (
	KafkaServerBrokertopicmetricsMessagesInTotalDesc         = "Attribute exposed for management (kafka.server<type=BrokerTopicMetrics, name=MessagesInPerSec, topic=__strimzi_canary><>Count)"
	KafkaServerBrokertopicmetricsBytesInTotalDesc            = "Attribute exposed for management (kafka.server<type=BrokerTopicMetrics, name=BytesInPerSec, topic=__strimzi_canary><>Count)"
	KafkaServerBrokertopicmetricsBytesOutTotalDesc           = "Attribute exposed for management (kafka.server<type=BrokerTopicMetrics, name=BytesOutPerSec, topic=__strimzi_canary><>Count)"
	KafkaControllerKafkacontrollerOfflinePartitionsCountDesc = "Attribute exposed for management (kafka.controller<type=KafkaController, name=OfflinePartitionsCount><>Value)"
	KafkaControllerKafkacontrollerGlobalPartitionCountDesc   = "Attribute exposed for management (kafka.controller<type=KafkaController, name=GlobalPartitionCount><>Value)"
	KafkaTopicKafkaLogLogSizeSumDesc                         = "Attribute exposed for management (kafka.log<type=Log, name=Size, topic=__consumer_offsets, partition=18><>Value)"
	KafkaBrokerQuotaSoftlimitbytesDesc                       = "Attribute exposed for management (io.strimzi.kafka.quotas<type=StorageChecker, name=SoftLimitBytes><>Value)"
	KafkaBrokerQuotaTotalstorageusedbytesDesc                = "Attribute exposed for management (io.strimzi.kafka.quotas<type=StorageChecker, name=TotalStorageUsedBytes><>Value)"
	KubeletVolumeStatsAvailableBytesDesc                     = "Number of available bytes in the volume"
	KubeletVolumeStatsUsedBytesDesc                          = "Number of used bytes in the volume"
	HaproxyServerBytesInTotalDesc                            = "Current total of incoming bytes"
	HaproxyServerBytesOutTotalDesc                           = "Current total of outgoing bytes"
)

type MetricsMetadata struct {
	Name           string
	Help           string
	Type           prometheus.ValueType
	VariableLabels []string
	ConstantLabels prometheus.Labels
}

func GetMetricsMetaData() map[string]MetricsMetadata {
	return map[string]MetricsMetadata{
		"kafka_server_brokertopicmetrics_messages_in_total": {
			Name:           "kafka_server_brokertopicmetrics_messages_in_total",
			Help:           KafkaServerBrokertopicmetricsMessagesInTotalDesc,
			Type:           prometheus.GaugeValue,
			VariableLabels: []string{"namespace", "topic"},
		},
		"kafka_server_brokertopicmetrics_bytes_in_total": {
			Name:           "kafka_server_brokertopicmetrics_bytes_in_total",
			Help:           KafkaServerBrokertopicmetricsBytesInTotalDesc,
			Type:           prometheus.GaugeValue,
			VariableLabels: []string{"namespace", "topic"},
		},
		"kafka_server_brokertopicmetrics_bytes_out_total": {
			Name:           "kafka_server_brokertopicmetrics_bytes_out_total",
			Help:           KafkaServerBrokertopicmetricsBytesOutTotalDesc,
			Type:           prometheus.GaugeValue,
			VariableLabels: []string{"namespace", "topic"},
		},
		"kafka_controller_kafkacontroller_offline_partitions_count": {
			Name:           "kafka_controller_kafkacontroller_offline_partitions_count",
			Help:           KafkaControllerKafkacontrollerOfflinePartitionsCountDesc,
			Type:           prometheus.GaugeValue,
			VariableLabels: []string{"namespace"},
		},
		"kafka_controller_kafkacontroller_global_partition_count": {
			Name:           "kafka_controller_kafkacontroller_global_partition_count",
			Help:           KafkaControllerKafkacontrollerGlobalPartitionCountDesc,
			Type:           prometheus.GaugeValue,
			VariableLabels: []string{"namespace"},
		},
		"kafka_topic:kafka_log_log_size:sum": {
			Name:           "kafka_topic:kafka_log_log_size:sum",
			Help:           KafkaTopicKafkaLogLogSizeSumDesc,
			Type:           prometheus.GaugeValue,
			VariableLabels: []string{"namespace", "topic"},
		},
		"kafka_broker_quota_softlimitbytes": {
			Name:           "kafka_broker_quota_softlimitbytes",
			Help:           KafkaBrokerQuotaSoftlimitbytesDesc,
			Type:           prometheus.GaugeValue,
			VariableLabels: []string{"namespace"},
		},
		"kafka_broker_quota_totalstorageusedbytes": {
			Name:           "kafka_broker_quota_totalstorageusedbytes",
			Help:           KafkaBrokerQuotaTotalstorageusedbytesDesc,
			Type:           prometheus.GaugeValue,
			VariableLabels: []string{"namespace"},
		},
		"kubelet_volume_stats_available_bytes": {
			Name:           "kubelet_volume_stats_available_bytes",
			Help:           KubeletVolumeStatsAvailableBytesDesc,
			Type:           prometheus.GaugeValue,
			VariableLabels: []string{"namespace", "persistentvolumeclaim"},
		},
		"kubelet_volume_stats_used_bytes": {
			Name:           "kubelet_volume_stats_used_bytes",
			Help:           KubeletVolumeStatsUsedBytesDesc,
			Type:           prometheus.GaugeValue,
			VariableLabels: []string{"namespace", "persistentvolumeclaim"},
		},
		"kafka_namespace:haproxy_server_bytes_in_total:rate5m": {
			Name:           "kafka_namespace:haproxy_server_bytes_in_total:rate5m",
			Help:           HaproxyServerBytesInTotalDesc,
			Type:           prometheus.GaugeValue,
			VariableLabels: []string{"namespace"},
		},
		"kafka_namespace:haproxy_server_bytes_out_total:rate5m": {
			Name:           "kafka_namespace:haproxy_server_bytes_out_total:rate5m",
			Help:           HaproxyServerBytesOutTotalDesc,
			Type:           prometheus.GaugeValue,
			VariableLabels: []string{"namespace"},
		},
	}
}
