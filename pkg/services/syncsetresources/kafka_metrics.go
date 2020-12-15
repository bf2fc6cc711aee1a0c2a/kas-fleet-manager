package syncsetresources

import (
	strimzi "gitlab.cee.redhat.com/service/managed-services-api/pkg/api/kafka.strimzi.io/v1beta1"
)

// Metrics configuration for Kafka Brokers
var kafkaMetrics = &strimzi.Metrics{
	LowercaseOutputName: true,
	Rules: []strimzi.Rule{
		{
			Pattern: "kafka.controller<type=KafkaController, name=OfflinePartitionsCount><>Value",
			Name:    "kafka_controller_kafkacontroller_offline_partitions_count",
			Type:    "GAUGE",
		},
		{
			Pattern: "kafka.server<type=ReplicaManager, name=UnderReplicatedPartitions><>Value",
			Name:    "kafka_server_replicamanager_under_replicated_partitions",
			Type:    "GAUGE",
		},
		{
			Pattern: "kafka.server<type=ReplicaManager, name=AtMinIsrPartitionCount><>Value",
			Name:    "kafka_server_replicamanager_at_min_isr_partition_count",
			Type:    "GAUGE",
		},
		{
			Pattern: "kafka.cluster<type=Partition, name=AtMinIsr, topic=(.+), partition=(.*)><>Value",
			Name:    "kafka_cluster_partition_at_min_isr",
			Type:    "GAUGE",
			Labels: map[string]string{
				"topic":     "$1",
				"partition": "$2",
			},
		},
		{
			Pattern: "kafka.server<type=ReplicaManager, name=UnderMinIsrPartitionCount><>Value",
			Name:    "kafka_server_replicamanager_under_min_isr_partition_count",
			Type:    "GAUGE",
		},
		{
			Pattern: "kafka.cluster<type=Partition, name=UnderMinIsr, topic=(.+), partition=(.*)><>Value",
			Name:    "kafka_cluster_partition_under_min_isr",
			Type:    "GAUGE",
			Labels: map[string]string{
				"topic":     "$1",
				"partition": "$2",
			},
		},
		{
			Pattern: "kafka.controller<type=KafkaController, name=ActiveControllerCount><>Value",
			Name:    "kafka_controller_kafkacontroller_active_controller_count",
			Type:    "GAUGE",
		},
		{
			Pattern: "kafka.server<type=ReplicaManager, name=LeaderCount><>Value",
			Name:    "kafka_server_replicamanager_leader_count",
			Type:    "GAUGE",
		},
		{
			Pattern: "kafka.server<type=BrokerTopicMetrics, name=BytesInPerSec, topic=(.+)><>Count",
			Name:    "kafka_server_brokertopicmetrics_bytes_in_total",
			Type:    "GAUGE",
			Labels: map[string]string{
				"topic": "$1",
			},
		},
		{
			Pattern: "kafka.server<type=BrokerTopicMetrics, name=BytesOutPerSec, topic=(.+)><>Count",
			Name:    "kafka_server_brokertopicmetrics_bytes_out_total",
			Type:    "GAUGE",
			Labels: map[string]string{
				"topic": "$1",
			},
		},
		{
			Pattern: "kafka.server<type=BrokerTopicMetrics, name=MessagesInPerSec, topic=(.+)><>Count",
			Name:    "kafka_server_brokertopicmetrics_messages_in_total",
			Type:    "GAUGE",
			Labels: map[string]string{
				"topic": "$1",
			},
		},
		{
			Pattern: "kafka.controller<type=KafkaController, name=GlobalPartitionCount><>Value",
			Name:    "kafka_controller_kafkacontroller_global_partition_count",
			Type:    "GAUGE",
		},
		{
			Pattern: "kafka.log<type=Log, name=Size, topic=(.+), partition=(.*)><>Value",
			Name:    "kafka_log_log_size",
			Type:    "GAUGE",
			Labels: map[string]string{
				"topic":     "$1",
				"partition": "$2",
			},
		},
		{
			Pattern: "kafka.log<type=LogManager, name=OfflineLogDirectoryCount><>Value",
			Name:    "kafka_log_logmanager_offline_log_directory_count",
			Type:    "GAUGE",
		},
		{
			Pattern: "kafka.controller<type=ControllerStats, name=UncleanLeaderElectionsPerSec><>Count",
			Name:    "kafka_controller_controllerstats_unclean_leader_elections_total",
			Type:    "GAUGE",
		},
		{
			Pattern: "kafka.server<type=ReplicaManager, name=PartitionCount><>Value",
			Name:    "kafka_server_replicamanager_partition_count",
			Type:    "GAUGE",
		},
		{
			Pattern: "kafka.server<type=BrokerTopicMetrics, name=TotalProduceRequestsPerSec, topic=(.+)><>Count",
			Name:    "kafka_server_brokertopicmetrics_total_produce_requests_total",
			Type:    "COUNTER",
			Labels: map[string]string{
				"topic": "$1",
			},
		},
		{
			Pattern: "kafka.server<type=BrokerTopicMetrics, name=FailedProduceRequestsPerSec><>Count",
			Name:    "kafka_server_brokertopicmetrics_failed_produce_requests_total",
			Type:    "COUNTER",
		},
		{
			Pattern: "kafka.server<type=BrokerTopicMetrics, name=TotalFetchRequestsPerSec, topic=(.+)><>Count",
			Name:    "kafka_server_brokertopicmetrics_total_fetch_requests_total",
			Type:    "COUNTER",
			Labels: map[string]string{
				"topic": "$1",
			},
		},
		{
			Pattern: "kafka.server<type=BrokerTopicMetrics, name=FailedFetchRequestsPerSec><>Count",
			Name:    "kafka_server_brokertopicmetrics_failed_fetch_requests_total",
			Type:    "COUNTER",
		},
		{
			Pattern: "kafka.network<type=SocketServer, name=NetworkProcessorAvgIdlePercent><>Value",
			Name:    "kafka_network_socketserver_network_processor_avg_idle_percent",
			Type:    "GAUGE",
		},
		{
			Pattern: "kafka.server<type=KafkaRequestHandlerPool, name=RequestHandlerAvgIdlePercent><>MeanRate",
			Name:    "kafka_server_kafkarequesthandlerpool_request_handler_avg_idle_percent",
			Type:    "GAUGE",
		},
		{
			Pattern: "kafka.cluster<type=Partition, name=ReplicasCount, topic=(.+), partition=(.*)><>Value",
			Name:    "kafka_cluster_partition_replicas_count",
			Type:    "GAUGE",
			Labels: map[string]string{
				"topic":     "$1",
				"partition": "$2",
			},
		},
		{
			Pattern: "kafka.network<type=RequestMetrics, name=TotalTimeMs, (.+)=(.+)><>(\\d+)thPercentile",
			Name:    "kafka_network_requestmetrics_total_time_ms",
			Type:    "GAUGE",
			Labels: map[string]string{
				"$1":       "$2",
				"quantile": "0.$3",
			},
		},
	},
}

// Metrics configuration for Zookeeper
var zookeeperMetrics = &strimzi.Metrics{
	LowercaseOutputName: true,
	Rules: []strimzi.Rule{
		{
			Pattern: "org.apache.ZooKeeperService<(name0=ReplicatedServer_id(\\d+), name1=replica.(\\d+), name2=(\\w+))><>OutstandingRequests",
			Name:    "zookeeper_outstanding_requests",
			Type:    "GAUGE",
			Labels: map[string]string{
				"replicaId":  "$2",
				"memberType": "$3",
			},
		},
		{
			Pattern: "org.apache.ZooKeeperService<(name0=ReplicatedServer_id(\\d+), name1=replica.(\\d+), name2=(\\w+))><>AvgRequestLatency",
			Name:    "zookeeper_avg_request_latency",
			Type:    "GAUGE",
			Labels: map[string]string{
				"replicaId":  "$2",
				"memberType": "$3",
			},
		},
		{
			Pattern: "org.apache.ZooKeeperService<(name0=ReplicatedServer_id(\\d+))><>QuorumSize",
			Name:    "zookeeper_quorum_size",
			Type:    "GAUGE",
		},
		{
			Pattern: "org.apache.ZooKeeperService<(name0=ReplicatedServer_id(\\d+), name1=replica.(\\d+), name2=(\\w+))><>NumAliveConnections",
			Name:    "zookeeper_num_alive_connections",
			Type:    "GAUGE",
			Labels: map[string]string{
				"replicaId":  "$2",
				"memberType": "$3",
			},
		},
		{
			Pattern: "org.apache.ZooKeeperService<(name0=ReplicatedServer_id(\\d+), name1=replica.(\\d+), name2=(\\w+), name3=InMemoryDataTree)><>NodeCount",
			Name:    "zookeeper_in_memory_data_tree_node_count",
			Type:    "GAUGE",
			Labels: map[string]string{
				"replicaId":  "$2",
				"memberType": "$3",
			},
		},
		{
			Pattern: "org.apache.ZooKeeperService<(name0=ReplicatedServer_id(\\d+), name1=replica.(\\d+), name2=(\\w+), name3=InMemoryDataTree)><>WatchCount",
			Name:    "zookeeper_in_memory_data_tree_watch_count",
			Type:    "GAUGE",
			Labels: map[string]string{
				"replicaId":  "$2",
				"memberType": "$3",
			},
		},
		{
			Pattern: "org.apache.ZooKeeperService<(name0=ReplicatedServer_id(\\d+), name1=replica.(\\d+), name2=(\\w+))><>MinRequestLatency",
			Name:    "zookeeper_min_request_latency",
			Type:    "GAUGE",
			Labels: map[string]string{
				"replicaId":  "$2",
				"memberType": "$3",
			},
		},
		{
			Pattern: "org.apache.ZooKeeperService<(name0=ReplicatedServer_id(\\d+), name1=replica.(\\d+), name2=(\\w+))><>MaxRequestLatency",
			Name:    "zookeeper_max_request_latency",
			Type:    "GAUGE",
			Labels: map[string]string{
				"replicaId":  "$2",
				"memberType": "$3",
			},
		},
	},
}
