package services

import (
	"fmt"

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
	numOfBrokers       = 3
	numOfZookeepers    = 3
	produceQuota       = 4000000
	consumeQuota       = 4000000
	kafkaMaxPartitions = 50
)

var (
	zkVolumeSize         = resource.NewScaledQuantity(10, resource.Giga)
	kafkaVolumeSize      = resource.NewScaledQuantity(100, resource.Giga)
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
	kafkaImageUrl = string("quay.io/lulf/kafka:latest-kafka-2.6.0")

	canaryImageUrl = "quay.io/ppatierno/strimzi-canary:0.0.1"

	adminServerUrl = "quay.io/sknot/strimzi-admin:0.0.1"
)

var deleteClaim = false
var jbodVolumeId = 0

var zookeeperMetrics = &strimzi.Metrics{
	LowercaseOutputName: true,
	Rules: []strimzi.Rule{
		{
			Pattern: "org.apache.ZooKeeperService<name0=ReplicatedServer_id(\\d+)><>(\\w+)",
			Name:    "zookeeper_$2",
			Type:    "GAUGE",
		},
		{
			Pattern: "org.apache.ZooKeeperService<name0=ReplicatedServer_id(\\d+), name1=replica.(\\d+)><>(\\w+)",
			Name:    "zookeeper_$3",
			Type:    "GAUGE",
			Labels: map[string]string{
				"replicaId": "$2",
			},
		},
		{
			Pattern: "org.apache.ZooKeeperService<name0=ReplicatedServer_id(\\d+), name1=replica.(\\d+), name2=(\\w+)><>(Packets\\w+)",
			Name:    "zookeeper_$4",
			Type:    "COUNTER",
			Labels: map[string]string{
				"replicaId":  "$2",
				"memberType": "$3",
			},
		},
		{
			Pattern: "org.apache.ZooKeeperService<name0=ReplicatedServer_id(\\d+), name1=replica.(\\d+), name2=(\\w+)><>(\\w+)",
			Name:    "zookeeper_$4",
			Type:    "GAUGE",
			Labels: map[string]string{
				"replicaId":  "$2",
				"memberType": "$3",
			},
		},
		{
			Pattern: "org.apache.ZooKeeperService<name0=ReplicatedServer_id(\\d+), name1=replica.(\\d+), name2=(\\w+), name3=(\\w+)><>(\\w+)",
			Name:    "zookeeper_$4_$5",
			Type:    "GAUGE",
			Labels: map[string]string{
				"replicaId":  "$2",
				"memberType": "$3",
			},
		},
		{
			Pattern: "org.apache.ZooKeeperService<name0=StandaloneServer_port(\\d+)><>(\\w+)",
			Name:    "zookeeper_$2",
			Type:    "GAUGE",
		},
		{
			Pattern: "org.apache.ZooKeeperService<name0=StandaloneServer_port(\\d+), name1=InMemoryDataTree><>(\\w+)",
			Name:    "zookeeper_$2",
			Type:    "GAUGE",
		},
	},
}

var kafkaMetrics = &strimzi.Metrics{
	LowercaseOutputName: true,
	Rules: []strimzi.Rule{
		{
			Pattern: "kafka.server<type=(.+), name=(.+), clientId=(.+), topic=(.+), partition=(.*)><>Value",
			Name:    "kafka_server_$1_$2",
			Type:    "GAUGE",
			Labels: map[string]string{
				"clientId":  "$3",
				"topic":     "$4",
				"partition": "$5",
			},
		},
		{
			Pattern: "kafka.server<type=(.+), name=(.+), clientId=(.+), brokerHost=(.+), brokerPort=(.+)><>Value",
			Name:    "kafka_server_$1_$2",
			Type:    "GAUGE",
			Labels: map[string]string{
				"clientId": "$3",
				"broker":   "$4:$5",
			},
		},
		{
			Pattern: "kafka.(\\w+)<type=(.+), name=(.+)Percent\\w*><>MeanRate",
			Name:    "kafka_$1_$2_$3_percent",
			Type:    "GAUGE",
		},
		{
			Pattern: "kafka.(\\w+)<type=(.+), name=(.+)Percent\\w*><>Value",
			Name:    "kafka_$1_$2_$3_percent",
			Type:    "GAUGE",
		},
		{
			Pattern: "kafka.(\\w+)<type=(.+), name=(.+)Percent\\w*, (.+)=(.+)><>Value",
			Name:    "kafka_$1_$2_$3_percent",
			Type:    "GAUGE",
			Labels: map[string]string{
				"$4": "$5",
			},
		},
		{
			Pattern: "kafka.(\\w+)<type=(.+), name=(.+)PerSec\\w*, (.+)=(.+), (.+)=(.+)><>Count",
			Name:    "kafka_$1_$2_$3_total",
			Type:    "COUNTER",
			Labels: map[string]string{
				"$4": "$5",
				"$6": "$7",
			},
		},
		{
			Pattern: "kafka.(\\w+)<type=(.+), name=(.+)PerSec\\w*, (.+)=(.+)><>Count",
			Name:    "kafka_$1_$2_$3_total",
			Type:    "COUNTER",
			Labels: map[string]string{
				"$4": "$5",
			},
		},
		{
			Pattern: "kafka.(\\w+)<type=(.+), name=(.+)PerSec\\w*><>Count",
			Name:    "kafka_$1_$2_$3_total",
			Type:    "COUNTER",
		},
		{
			Pattern: "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.+), (.+)=(.+)><>Value",
			Name:    "kafka_$1_$2_$3",
			Type:    "GAUGE",
			Labels: map[string]string{
				"$4": "$5",
				"$6": "$7",
			},
		},
		{
			Pattern: "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.+)><>Value",
			Name:    "kafka_$1_$2_$3",
			Type:    "GAUGE",
			Labels: map[string]string{
				"$4": "$5",
			},
		},
		{
			Pattern: "kafka.(\\w+)<type=(.+), name=(.+)><>Value",
			Name:    "kafka_$1_$2_$3",
			Type:    "GAUGE",
		},
		{
			Pattern: "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.+), (.+)=(.+)><>Count",
			Name:    "kafka_$1_$2_$3_count",
			Type:    "COUNTER",
			Labels: map[string]string{
				"$4": "$5",
				"$6": "$7",
			},
		},
		{
			Pattern: "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.*), (.+)=(.+)><>(\\d+)thPercentile",
			Name:    "kafka_$1_$2_$3",
			Type:    "GAUGE",
			Labels: map[string]string{
				"$4":       "$5",
				"$6":       "$7",
				"quantile": "0.$8",
			},
		},
		{
			Pattern: "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.+)><>Count",
			Name:    "kafka_$1_$2_$3_count",
			Type:    "COUNTER",
			Labels: map[string]string{
				"$4": "$5",
			},
		},
		{
			Pattern: "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.*)><>(\\d+)thPercentile",
			Name:    "kafka_$1_$2_$3",
			Type:    "GAUGE",
			Labels: map[string]string{
				"$4":       "$5",
				"quantile": "0.$6",
			},
		},
		{
			Pattern: "kafka.(\\w+)<type=(.+), name=(.+)><>Count",
			Name:    "kafka_$1_$2_$3_count",
			Type:    "COUNTER",
		},
		{
			Pattern: "kafka.(\\w+)<type=(.+), name=(.+)><>(\\d+)thPercentile",
			Name:    "kafka_$1_$2_$3",
			Type:    "GAUGE",
			Labels: map[string]string{
				"quantile": "0.$4",
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
func newKafkaSyncsetBuilder(kafkaRequest *api.KafkaRequest) (*cmv1.SyncsetBuilder, string, *errors.ServiceError) {
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

	// Need to override the broker route hosts to ensure the length is not above 63 characters which is the max length of the Host on an OpenShift route
	brokerOverrides := []strimzi.GenericKafkaListenerConfigurationBroker{}
	for i := 0; i < numOfBrokers; i++ {
		brokerOverride := strimzi.GenericKafkaListenerConfigurationBroker{
			Host:   fmt.Sprintf("broker-%d-%s", i, kafkaRequest.BootstrapServerHost),
			Broker: i,
		}
		brokerOverrides = append(brokerOverrides, brokerOverride)
	}

	// Derive Kafka config based on global constants
	kafkaConfig := map[string]string{
		"offsets.topic.replication.factor": "3",
		// Retention and segment size is set to disk capacity
		"log.retention.ms":                         fmt.Sprintf("%d", 1000*kafkaVolumeSize.Value()/produceQuota),
		"log.segment.bytes":                        fmt.Sprintf("%d", kafkaVolumeSize.Value()/kafkaMaxPartitions),
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
	}

	// build array of objects to be created by the syncset
	kafkaCR := &strimzi.Kafka{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kafka.strimzi.io/v1beta1",
			Kind:       "Kafka",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kafkaRequest.Name,
			Namespace: namespaceName,
		},
		Spec: strimzi.KafkaSpec{
			Kafka: strimzi.KafkaClusterSpec{
				Image:    &kafkaImageUrl,
				Config:   kafkaConfig,
				Replicas: numOfBrokers,
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
								ID:   &jbodVolumeId,
								Type: strimzi.PersistentClaim,
								PersistentClaimStorage: strimzi.PersistentClaimStorage{
									Size:        kafkaVolumeSize.String(),
									DeleteClaim: &deleteClaim,
								},
							},
						},
					},
				},
				Listeners: []strimzi.GenericKafkaListener{
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
						Name: "external",
						Type: strimzi.Route,
						TLS:  true,
						Port: 9094,
						Configuration: &strimzi.GenericKafkaListenerConfiguration{
							Bootstrap: &strimzi.GenericKafkaListenerConfigurationBootstrap{
								Host: kafkaRequest.BootstrapServerHost,
							},
							Brokers: brokerOverrides,
						},
					},
				},
				Metrics: kafkaMetrics,
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
					Type: strimzi.PersistentClaim,
					PersistentClaimStorage: strimzi.PersistentClaimStorage{
						Size:        zkVolumeSize.String(),
						DeleteClaim: &deleteClaim,
					},
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
							Image: canaryImageUrl,
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
							Image: adminServerUrl,
							Env: []corev1.EnvVar{
								{
									Name:  "KAFKA_ADMIN_BOOTSTRAP_SERVERS",
									Value: kafkaRequest.Name + "-kafka-bootstrap:9092",
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

	syncsetBuilder = syncsetBuilder.Resources(resources...)
	syncsetId := buildSyncsetIdentifier(kafkaRequest)
	return syncsetBuilder, syncsetId, nil
}
