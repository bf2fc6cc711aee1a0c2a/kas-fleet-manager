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
	KafkaBrokerQuotaHardlimitbytesDesc                       = "Attribute exposed for management (io.strimzi.kafka.quotas<type=StorageChecker, name=HardLimitBytes><>Value)"
	KafkaBrokerQuotaTotalstorageusedbytesDesc                = "Attribute exposed for management (io.strimzi.kafka.quotas<type=StorageChecker, name=TotalStorageUsedBytes><>Value)"
	KafkaBrokerClientQuotaLimitDesc                          = "Produce/fetch/request client quota limits imposed by broker quota-plugin"
	KubeletVolumeStatsAvailableBytesDesc                     = "Number of available bytes in the volume"
	KubeletVolumeStatsUsedBytesDesc                          = "Number of used bytes in the volume"
	HaproxyServerBytesInTotalDesc                            = "Current total of incoming bytes"
	HaproxyServerBytesOutTotalDesc                           = "Current total of outgoing bytes"
	KafkaTopicPartitionsSumDesc                              = "Number of topic partitions for this Kafka"
	KafkaTopicPartitionsCountDesc                            = "Number of Topics for this Kafka"
	KafkaConsumergroupMembersDesc                            = "Amount of members in a consumer group"
	KafkaServerSocketServerMetricsConnectionCountDesc        = "Current number of total kafka connections"
	KafkaServerSocketServerMetricsConnectionCreationRateDesc = "Current rate of connections creation"
	KafkaTopicIncomingMessagesRateDesc                       = "Current rate of incoming messages over the last 5 minutes"
	KafkaTopicTotalIncomingBytesRateDesc                     = "Current total rate of incoming bytes over the last 5 minutes"
	KafkaTopicTotalOutgoingBytesRateDesc                     = "Current total rate of outgoing bytes over the last 5 minutes"
	KafkaInstanceSpecBrokersDesiredCountDesc                 = "Desired number of brokers for this Kafka"
	KafkaInstanceMaxMessageSizeLimitDesc                     = "Maximum message size for this Kafka"
	KafkaInstancePartitionLimitDesc                          = "Maximum number of partitions for this Kafka"
	KafkaInstanceConnectionLimitDesc                         = "Maximum number of connections for this Kafka"
	KafkaInstanceConnectionCreationRateLimitDesc             = "Maximum rate of new connections for this Kafka"
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
		"kafka_broker_quota_hardlimitbytes": {
			Name:           "kafka_broker_quota_hardlimitbytes",
			Help:           KafkaBrokerQuotaHardlimitbytesDesc,
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
		"kafka_broker_client_quota_limit": {
			Name:           "kafka_broker_client_quota_limit",
			Help:           KafkaBrokerClientQuotaLimitDesc,
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
			VariableLabels: []string{},
		},
		"kafka_topic:kafka_topic_partitions:count": {
			Name:           "kafka_topic:kafka_topic_partitions:count",
			Help:           KafkaTopicPartitionsCountDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{},
		},
		"consumergroup:kafka_consumergroup_members:count": {
			Name:           "consumergroup:kafka_consumergroup_members:count",
			Help:           KafkaConsumergroupMembersDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{},
		},
		"kafka_namespace:kafka_server_socket_server_metrics_connection_count:sum": {
			Name:           "kafka_namespace:kafka_server_socket_server_metrics_connection_count:sum",
			Help:           KafkaServerSocketServerMetricsConnectionCountDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{},
		},
		"kafka_namespace:kafka_server_socket_server_metrics_connection_creation_rate:sum": {
			Name:           "kafka_namespace:kafka_server_socket_server_metrics_connection_creation_rate:sum",
			Help:           KafkaServerSocketServerMetricsConnectionCreationRateDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{},
		},
		"kafka_topic:kafka_server_brokertopicmetrics_messages_in_total:rate5m": {
			Name:           "kafka_topic:kafka_server_brokertopicmetrics_messages_in_total:rate5m",
			Help:           KafkaTopicIncomingMessagesRateDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"statefulset_kubernetes_io_pod_name", "strimzi_io_cluster", "topic"},
		},
		"kafka_topic:kafka_server_brokertopicmetrics_bytes_in_total:rate5m": {
			Name:           "kafka_topic:kafka_server_brokertopicmetrics_bytes_in_total:rate5m",
			Help:           KafkaTopicTotalIncomingBytesRateDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"statefulset_kubernetes_io_pod_name", "strimzi_io_cluster", "topic"},
		},
		"kafka_topic:kafka_server_brokertopicmetrics_bytes_out_total:rate5m": {
			Name:           "kafka_topic:kafka_server_brokertopicmetrics_bytes_out_total:rate5m",
			Help:           KafkaTopicTotalOutgoingBytesRateDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"statefulset_kubernetes_io_pod_name", "strimzi_io_cluster", "topic"},
		},
		"kafka_instance_spec_brokers_desired_count": {
			Name:           "kafka_instance_spec_brokers_desired_count",
			Help:           KafkaInstanceSpecBrokersDesiredCountDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"instance_name"},
		},
		"kafka_instance_max_message_size_limit": {
			Name:           "kafka_instance_max_message_size_limit",
			Help:           KafkaInstanceMaxMessageSizeLimitDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"instance_name"},
		},
		"kafka_instance_partition_limit": {
			Name:           "kafka_instance_partition_limit",
			Help:           KafkaInstancePartitionLimitDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"instance_name"},
		},
		"kafka_instance_connection_limit": {
			Name:           "kafka_instance_connection_limit",
			Help:           KafkaInstanceConnectionLimitDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"instance_name"},
		},
		"kafka_instance_connection_creation_rate_limit": {
			Name:           "kafka_instance_connection_creation_rate_limit",
			Help:           KafkaInstanceConnectionCreationRateLimitDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"instance_name"},
		},
	}
}
