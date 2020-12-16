package services

import (
	"fmt"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/constants"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	strimzi "gitlab.cee.redhat.com/service/managed-services-api/pkg/api/kafka.strimzi.io/v1beta1"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	numOfZookeepers = 3
	produceQuota    = 4000000
	consumeQuota    = 4000000
)

var (
	zkVolumeSize         = resource.NewScaledQuantity(10, resource.Giga)
	kafkaVolumeSize      = resource.NewScaledQuantity(225, resource.Giga)
	kafkaContainerMemory = resource.NewScaledQuantity(1, resource.Giga)
	kafkaContainerCpu    = resource.NewMilliQuantity(1000, resource.DecimalSI)
	kafkaJvmOptions      = &strimzi.JvmOptionsSpec{
		Xms: "512m",
		Xmx: "512m",
	}
	zkContainerMemory = resource.NewScaledQuantity(1, resource.Giga)
	zkContainerCpu    = resource.NewMilliQuantity(500, resource.DecimalSI)
	zkJvmOptions      = &strimzi.JvmOptionsSpec{
		Xms: "512m",
		Xmx: "512m",
	}
)

var deleteClaim = false
var jbodVolumeId = 0

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

//go:generate moq -out syncset_moq.go . SyncsetService
type SyncsetService interface {
	Create(syncsetBuilder *cmv1.SyncsetBuilder, syncsetId, clusterId string) (*cmv1.Syncset, *errors.ServiceError)
	Delete(syncsetId, clusterId string) (int, *errors.ServiceError)
}

func NewSyncsetService(ocmClient ocm.Client) SyncsetService {
	return &syncsetService{
		ocmClient: ocmClient,
	}
}

var _ SyncsetService = &syncsetService{}

type syncsetService struct {
	ocmClient ocm.Client
}

// Create builds the syncset and syncs it to the desired cluster
func (s syncsetService) Create(syncsetBuilder *cmv1.SyncsetBuilder, syncsetId, clusterId string) (*cmv1.Syncset, *errors.ServiceError) {
	//max syncset Id Length is 50
	syncsetBuilder.ID(truncateString(syncsetId, 50))
	syncset, buildErr := syncsetBuilder.Build()
	if buildErr != nil {
		return nil, errors.GeneralError("failed to build syncset: %s", buildErr)
	}
	// create the syncset on the cluster
	response, syncsetErr := s.ocmClient.CreateSyncSet(clusterId, syncset)

	if syncsetErr != nil {
		fmt.Println(syncsetErr)
		return nil, errors.GeneralError(fmt.Sprintf("failed to create syncset: %s for cluster id: %s", syncset.ID(), clusterId), syncsetErr)
	}
	return response, nil
}

func (s syncsetService) Delete(syncsetId, clusterId string) (int, *errors.ServiceError) {
	// create the syncset on the cluster
	statusCode, syncsetErr := s.ocmClient.DeleteSyncSet(clusterId, syncsetId)
	if syncsetErr != nil {
		return statusCode, errors.GeneralError(fmt.Sprintf("failed to delete syncset: %s for cluster id: %s returned status code: %d", syncsetId, clusterId, statusCode), syncsetErr)
	}
	return statusCode, nil
}

// syncset builder for a kafka/strimzi custom resource
func newKafkaSyncsetBuilder(kafkaRequest *api.KafkaRequest, kafkaConfig *config.KafkaConfig, keycloakConfig *config.KeycloakConfig, clientSecretValue string) (*cmv1.SyncsetBuilder, string, *errors.ServiceError) {
	syncsetBuilder := cmv1.NewSyncset()

	namespaceName := buildKafkaNamespaceIdentifier(kafkaRequest)
	namespaceName, err := replaceNamespaceSpecialChar(namespaceName)
	if err != nil {
		return syncsetBuilder, "", errors.GeneralError(fmt.Sprintf("unable to create syncset for kafka id: %s", kafkaRequest.ID), err)
	}

	sanitizedKafkaName := buildTruncateKafkaIdentifier(kafkaRequest)
	sanitizedKafkaName, err = replaceNamespaceSpecialChar(sanitizedKafkaName)
	if err != nil {
		return syncsetBuilder, "", errors.GeneralError(fmt.Sprintf("unable to create syncset for kafka id: %s", kafkaRequest.ID), err)
	}

	kafkaTLSSecretName := kafkaRequest.Name + "-tls-secret"

	// Need to override the broker route hosts to ensure the length is not above 63 characters which is the max length of the Host on an OpenShift route
	brokerOverrides := []strimzi.GenericKafkaListenerConfigurationBroker{}
	for i := 0; i < kafkaConfig.NumOfBrokers; i++ {
		brokerOverride := strimzi.GenericKafkaListenerConfigurationBroker{
			Host:   fmt.Sprintf("broker-%d-%s", i, kafkaRequest.BootstrapServerHost),
			Broker: i,
		}
		brokerOverrides = append(brokerOverrides, brokerOverride)
	}

	// Derive Kafka config based on global constants
	kafkaCRConfig := map[string]string{
		"offsets.topic.replication.factor":         "3",
		"transaction.state.log.min.isr":            "2",
		"transaction.state.log.replication.factor": "3",
		"client.quota.callback.class":              "org.apache.kafka.server.quota.StaticQuotaCallback",
		// Throttle at 4 MB/sec
		"client.quota.callback.static.produce": fmt.Sprintf("%d", produceQuota),
		"client.quota.callback.static.consume": fmt.Sprintf("%d", consumeQuota),
		// Start throttling when disk is above 90%. Full stop at 95%.
		"client.quota.callback.static.storage.soft": fmt.Sprintf("%d", int64(0.9*float64(kafkaVolumeSize.Value()))),
		"client.quota.callback.static.storage.hard": fmt.Sprintf("%d", int64(0.95*float64(kafkaVolumeSize.Value()))),
		// Check storage every 30 seconds
		"client.quota.callback.static.storage.check-interval": "30",
		"quota.window.num":          "30",
		"quota.window.size.seconds": "2",
		// Limit client connections to 500 per broker
		"max.connections": "500",
	}

	var plainOverOauthAuthenticationListener *strimzi.KafkaListenerAuthentication
	var oauthAuthenticationListener *strimzi.KafkaListenerAuthentication
	if keycloakConfig.EnableAuthenticationOnKafka {
		ssoClientID := buildKeycloakClientNameIdentifier(kafkaRequest)
		secretSources := []strimzi.CertSecretSource{
			{
				Certificate: config.NewKeycloakConfig().TLSTrustedCertificatesKey,
				SecretName:  kafkaRequest.Name + "-sso-cert",
			},
		}
		secretSource := strimzi.GenericSecretSource{
			Key:        config.NewKeycloakConfig().MASClientSecretKey,
			SecretName: kafkaRequest.Name + "-sso-secret",
		}
		plainOverOauthAuthenticationListener = &strimzi.KafkaListenerAuthentication{
			Type: strimzi.OAuth,
			KafkaListenerAuthenticationOAuth: strimzi.KafkaListenerAuthenticationOAuth{
				ClientID:               ssoClientID,
				JwksEndpointURI:        keycloakConfig.JwksEndpointURI,
				UserNameClaim:          keycloakConfig.UserNameClaim,
				ValidIssuerURI:         keycloakConfig.ValidIssuerURI,
				TLSTrustedCertificates: secretSources,
				ClientSecret:           secretSource,
				EnablePlain:            true,
				TokenEndpointURI:       keycloakConfig.TokenEndpointURI,
			},
		}

		oauthAuthenticationListener = &strimzi.KafkaListenerAuthentication{
			Type: strimzi.OAuth,
			KafkaListenerAuthenticationOAuth: strimzi.KafkaListenerAuthenticationOAuth{
				ClientID:               ssoClientID,
				JwksEndpointURI:        keycloakConfig.JwksEndpointURI,
				UserNameClaim:          keycloakConfig.UserNameClaim,
				ValidIssuerURI:         keycloakConfig.ValidIssuerURI,
				TLSTrustedCertificates: secretSources,
				ClientSecret:           secretSource,
			},
		}
	}

	listeners := []strimzi.GenericKafkaListener{
		{
			Name: "plain",
			Type: strimzi.Internal,
			TLS:  false,
			Port: 9092,
		},
		{
			Name: "tls",
			Type: strimzi.Internal,
			TLS:  true,
			Port: 9093,
		},
		{
			Name:           "external",
			Type:           strimzi.Route,
			TLS:            true,
			Port:           9094,
			Authentication: plainOverOauthAuthenticationListener,
			Configuration: &strimzi.GenericKafkaListenerConfiguration{
				Bootstrap: &strimzi.GenericKafkaListenerConfigurationBootstrap{
					Host: kafkaRequest.BootstrapServerHost,
				},
				Brokers:               brokerOverrides,
				BrokerCertChainAndKey: buildBrokerCertChainAndKeyResource(kafkaConfig, kafkaTLSSecretName),
			},
		},
		{
			Name:           "oauth",
			Type:           strimzi.Internal,
			TLS:            false,
			Port:           9095,
			Authentication: oauthAuthenticationListener,
		},
	}

	labels := make(map[string]string)
	if kafkaConfig.EnableDedicatedIngress {
		labels["ingressType"] = "sharded" // signal detected by the shared ingress controller
	}

	// build array of objects to be created by the syncset
	kafkaStorage := strimzi.PersistentClaimStorage{
		Size:        kafkaVolumeSize.String(),
		DeleteClaim: &deleteClaim,
	}
	zooKeeperStorage := strimzi.PersistentClaimStorage{
		Size:        zkVolumeSize.String(),
		DeleteClaim: &deleteClaim,
	}

	if kafkaConfig.KafkaStorageClass != "" {
		kafkaStorage.Class = kafkaConfig.KafkaStorageClass
		zooKeeperStorage.Class = kafkaConfig.KafkaStorageClass
	}

	kafkaCR := &strimzi.Kafka{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kafka.strimzi.io/v1beta1",
			Kind:       "Kafka",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kafkaRequest.Name,
			Namespace: namespaceName,
			Labels:    labels,
		},
		Spec: strimzi.KafkaSpec{
			Kafka: strimzi.KafkaClusterSpec{
				Config:   kafkaCRConfig,
				Replicas: kafkaConfig.NumOfBrokers,
				Resources: &corev1.ResourceRequirements{
					Requests: map[corev1.ResourceName]resource.Quantity{
						"memory": *kafkaContainerMemory,
						"cpu":    *kafkaContainerCpu,
					},
					Limits: map[corev1.ResourceName]resource.Quantity{
						"memory": *kafkaContainerMemory,
						"cpu":    *kafkaContainerCpu,
					},
				},
				JvmOptions: kafkaJvmOptions,
				Storage: strimzi.Storage{
					Type: strimzi.Jbod,
					JbodStorage: strimzi.JbodStorage{
						Volumes: []strimzi.JbodVolume{
							{
								ID:                     &jbodVolumeId,
								Type:                   strimzi.PersistentClaim,
								PersistentClaimStorage: kafkaStorage,
							},
						},
					},
				},
				Listeners: listeners,
				Metrics:   kafkaMetrics,
				Template: &strimzi.KafkaTemplate{
					Pod: &strimzi.PodTemplate{
						Affinity: &corev1.Affinity{
							PodAntiAffinity: &corev1.PodAntiAffinity{
								RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
									{
										TopologyKey: "kubernetes.io/hostname",
									},
								},
							},
						},
					},
				},
			},
			Zookeeper: strimzi.ZookeeperClusterSpec{
				Replicas: numOfZookeepers,
				Storage: strimzi.Storage{
					Type:                   strimzi.PersistentClaim,
					PersistentClaimStorage: zooKeeperStorage,
				},
				Resources: &corev1.ResourceRequirements{
					Requests: map[corev1.ResourceName]resource.Quantity{
						"memory": *zkContainerMemory,
						"cpu":    *zkContainerCpu,
					},
					Limits: map[corev1.ResourceName]resource.Quantity{
						"memory": *zkContainerMemory,
						"cpu":    *zkContainerCpu,
					},
				},
				JvmOptions: zkJvmOptions,
				Metrics:    zookeeperMetrics,
				Template: &strimzi.ZookeeperTemplate{
					Pod: &strimzi.PodTemplate{
						Affinity: &corev1.Affinity{
							PodAntiAffinity: &corev1.PodAntiAffinity{
								RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
									{
										TopologyKey: "kubernetes.io/hostname",
									},
								},
							},
						},
					},
				},
			},
			KafkaExporter: &strimzi.KafkaExporterSpec{
				TopicRegex: ".*",
				GroupRegex: ".*",
			},
		},
	}

	// Set rack awareness configuration if requested Kafka is multiAZ
	if kafkaRequest.MultiAZ {
		kafkaCR.Spec.Kafka.Rack = &strimzi.Rack{
			TopologyKey: "topology.kubernetes.io/zone",
		}
		kafkaCR.Spec.Zookeeper.Template.Pod.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution =
			append(kafkaCR.Spec.Zookeeper.Template.Pod.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution,
				corev1.PodAffinityTerm{
					TopologyKey: "topology.kubernetes.io/zone",
				})
	}

	canaryName := sanitizedKafkaName + "-canary"
	// build the canary deployment
	canaryLabels := map[string]string{
		"app":                                    canaryName,
		constants.ObservabilityCanaryPodLabelKey: constants.ObservabilityCanaryPodLabelValue,
	}

	canary := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      canaryName,
			Namespace: namespaceName,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: canaryLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: canaryLabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  canaryName,
							Image: kafkaConfig.KafkaCanaryImage,
							Env: []corev1.EnvVar{
								{
									Name:  "KAFKA_BOOTSTRAP_SERVERS",
									Value: kafkaRequest.Name + "-kafka-bootstrap:9092",
								},
								{
									Name:  "TOPIC_RECONCILE_MS",
									Value: "5000",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "metrics",
									ContainerPort: 8080,
								},
							},
						},
					},
				},
			},
		},
	}

	adminServerName := sanitizedKafkaName + "-admin-server"
	adminServer := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      adminServerName,
			Namespace: namespaceName,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": adminServerName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": adminServerName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  adminServerName,
							Image: kafkaConfig.KafkaAdminServerImage,
							Env: []corev1.EnvVar{
								{
									Name:  "KAFKA_ADMIN_BOOTSTRAP_SERVERS",
									Value: kafkaRequest.Name + "-kafka-bootstrap:9095",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 8080,
								},
							},
						},
					},
				},
			},
		},
	}

	adminServerService := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      adminServerName,
			Namespace: namespaceName,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": adminServerName,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       8080,
					TargetPort: intstr.FromString("http"),
				},
			},
		},
	}

	adminServerRoute := &routev1.Route{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "route.openshift.io/v1",
			Kind:       "Route",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      adminServerName,
			Namespace: namespaceName,
			Labels:    labels,
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: adminServerService.Name,
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString("http"),
			},
			Host: "admin-server-" + kafkaRequest.BootstrapServerHost,
			TLS: &routev1.TLSConfig{
				Termination: routev1.TLSTerminationEdge,
			},
		},
	}

	if kafkaConfig.EnableKafkaExternalCertificate {
		adminServerRoute.Spec.TLS.Certificate = kafkaConfig.KafkaTLSCert
		adminServerRoute.Spec.TLS.Key = kafkaConfig.KafkaTLSKey
	}

	resources := []interface{}{
		&projectv1.Project{
			TypeMeta: metav1.TypeMeta{
				APIVersion: projectv1.SchemeGroupVersion.String(),
				Kind:       "Project",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   namespaceName,
				Labels: constants.NamespaceLabels,
			},
		},
		kafkaCR,
		canary,
		adminServer,
		adminServerService,
		adminServerRoute,
	}

	if keycloakConfig.EnableAuthenticationOnKafka {
		clientSecret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      kafkaRequest.Name + "-sso-secret",
				Namespace: namespaceName,
			},
			Type: corev1.SecretType("Opaque"),
			Data: map[string][]byte{
				keycloakConfig.MASClientSecretKey: []byte(clientSecretValue),
			},
		}
		caSecret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      kafkaRequest.Name + "-sso-cert",
				Namespace: namespaceName,
			},
			Type: corev1.SecretType("Opaque"),
			Data: map[string][]byte{
				keycloakConfig.TLSTrustedCertificatesKey: []byte(keycloakConfig.TLSTrustedCertificatesValue),
			},
		}

		resources = append(resources, clientSecret, caSecret)
	}

	if kafkaConfig.EnableKafkaExternalCertificate {
		resources = append(resources, buildKafkaTLSSecretResource(kafkaConfig, kafkaTLSSecretName, namespaceName))
	}

	syncsetBuilder = syncsetBuilder.Resources(resources...)
	syncsetId := buildSyncsetIdentifier(kafkaRequest)
	return syncsetBuilder, syncsetId, nil
}

func buildKafkaTLSSecretResource(kafkaConfig *config.KafkaConfig, kafkaTLSSecretName string, kafkaNamespace string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kafkaTLSSecretName,
			Namespace: kafkaNamespace,
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte(kafkaConfig.KafkaTLSCert),
			corev1.TLSPrivateKeyKey: []byte(kafkaConfig.KafkaTLSKey),
		},
	}
}

func buildBrokerCertChainAndKeyResource(kafkaConfig *config.KafkaConfig, kafkaTLSSecretName string) *strimzi.CertAndKeySecretSource {
	if !kafkaConfig.EnableKafkaExternalCertificate {
		return nil
	}

	return &strimzi.CertAndKeySecretSource{
		Certificate: corev1.TLSCertKey,
		Key:         corev1.TLSPrivateKeyKey,
		SecretName:  kafkaTLSSecretName,
	}
}
