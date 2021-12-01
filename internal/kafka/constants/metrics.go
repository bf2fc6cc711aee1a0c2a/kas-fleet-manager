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
	KafkaTopicPartitionsSumDesc                              = "Number of partitions for this Topic"
	KafkaTopicPartitionsCountDesc                            = "Number of Topics for this Kafka"
	KafkaConsumergroupMembersDesc                            = "Amount of members in a consumer group"
	KafkaServerSocketServerMetricsConnectionCountDesc        = "Current number of total kafka connections"
	KafkaServerSocketServerMetricsConnectionCreationRateDesc = "Current rate of connections creation"
)

type MetricsMetadata struct {
	Name           string
	Help           string
	Type           prometheus.ValueType
	TypeName       string
	VariableLabels []string
	ConstantLabels prometheus.Labels
}

func GetMetricsMetaData() map[string]MetricsMetadata {
	return map[string]MetricsMetadata{
		"kafka_server_brokertopicmetrics_messages_in_total": {
			Name:           "kafka_server_brokertopicmetrics_messages_in_total",
			Help:           KafkaServerBrokertopicmetricsMessagesInTotalDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"topic", "statefulset_kubernetes_io_pod_name", "strimzi_io_cluster"},
		},
		"kafka_server_brokertopicmetrics_bytes_in_total": {
			Name:           "kafka_server_brokertopicmetrics_bytes_in_total",
			Help:           KafkaServerBrokertopicmetricsBytesInTotalDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"topic", "statefulset_kubernetes_io_pod_name", "strimzi_io_cluster"},
		},
		"kafka_server_brokertopicmetrics_bytes_out_total": {
			Name:           "kafka_server_brokertopicmetrics_bytes_out_total",
			Help:           KafkaServerBrokertopicmetricsBytesOutTotalDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"topic", "statefulset_kubernetes_io_pod_name", "strimzi_io_cluster"},
		},
		"kafka_controller_kafkacontroller_offline_partitions_count": {
			Name:           "kafka_controller_kafkacontroller_offline_partitions_count",
			Help:           KafkaControllerKafkacontrollerOfflinePartitionsCountDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"statefulset_kubernetes_io_pod_name", "strimzi_io_cluster"},
		},
		"kafka_controller_kafkacontroller_global_partition_count": {
			Name:           "kafka_controller_kafkacontroller_global_partition_count",
			Help:           KafkaControllerKafkacontrollerGlobalPartitionCountDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"statefulset_kubernetes_io_pod_name", "strimzi_io_cluster"},
		},
		"kafka_topic:kafka_log_log_size:sum": {
			Name:           "kafka_topic:kafka_log_log_size:sum",
			Help:           KafkaTopicKafkaLogLogSizeSumDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"topic"},
		},
		"kafka_broker_quota_softlimitbytes": {
			Name:           "kafka_broker_quota_softlimitbytes",
			Help:           KafkaBrokerQuotaSoftlimitbytesDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"statefulset_kubernetes_io_pod_name", "strimzi_io_cluster"},
		},
		"kafka_broker_quota_totalstorageusedbytes": {
			Name:           "kafka_broker_quota_totalstorageusedbytes",
			Help:           KafkaBrokerQuotaTotalstorageusedbytesDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"statefulset_kubernetes_io_pod_name", "strimzi_io_cluster"},
		},
		"kubelet_volume_stats_available_bytes": {
			Name:           "kubelet_volume_stats_available_bytes",
			Help:           KubeletVolumeStatsAvailableBytesDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"persistentvolumeclaim"},
		},
		"kubelet_volume_stats_used_bytes": {
			Name:           "kubelet_volume_stats_used_bytes",
			Help:           KubeletVolumeStatsUsedBytesDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"persistentvolumeclaim"},
		},
		"kafka_namespace:haproxy_server_bytes_in_total:rate5m": {
			Name:           "kafka_namespace:haproxy_server_bytes_in_total:rate5m",
			Help:           HaproxyServerBytesInTotalDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{},
		},
		"kafka_namespace:haproxy_server_bytes_out_total:rate5m": {
			Name:           "kafka_namespace:haproxy_server_bytes_out_total:rate5m",
			Help:           HaproxyServerBytesOutTotalDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{},
		},
		"kafka_topic:kafka_topic_partitions:sum": {
			Name:           "kafka_topic:kafka_topic_partitions:sum",
			Help:           KafkaTopicPartitionsSumDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"node_ip", "pod", "strimzi_io_cluster", "strimzi_io_name"},
		},
		"kafka_topic:kafka_topic_partitions:count": {
			Name:           "kafka_topic:kafka_topic_partitions:count",
			Help:           KafkaTopicPartitionsCountDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"node_ip", "pod", "strimzi_io_cluster", "strimzi_io_name"},
		},
		"consumergroup:kafka_consumergroup_members:count": {
			Name:           "consumergroup:kafka_consumergroup_members:count",
			Help:           KafkaConsumergroupMembersDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"node_ip", "pod", "strimzi_io_cluster", "strimzi_io_name"},
		},
		"kafka_namespace:kafka_server_socket_server_metrics_connection_count:sum": {
			Name:           "kafka_namespace:kafka_server_socket_server_metrics_connection_count:sum",
			Help:           KafkaServerSocketServerMetricsConnectionCountDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"statefulset_kubernetes_io_pod_name", "strimzi_io_cluster", "strimzi_io_cluster"},
		},
		"kafka_namespace:kafka_server_socket_server_metrics_connection_creation_rate:sum": {
			Name:           "kafka_namespace:kafka_server_socket_server_metrics_connection_creation_rate:sum",
			Help:           KafkaServerSocketServerMetricsConnectionCreationRateDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"statefulset_kubernetes_io_pod_name", "strimzi_io_cluster", "strimzi_io_cluster"},
		},
	}
}
