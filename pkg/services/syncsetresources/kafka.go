package syncsetresources

import (
	"fmt"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	strimzi "gitlab.cee.redhat.com/service/managed-services-api/pkg/api/kafka.strimzi.io/v1beta1"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	numOfZookeepers = 3
	produceQuota    = 4000000
	consumeQuota    = 4000000
)

var (
	deleteClaim  = false
	jbodVolumeId = 0

	// kafka vars
	kafkaVolumeSize      = resource.NewScaledQuantity(225, resource.Giga)
	kafkaContainerMemory = resource.NewScaledQuantity(1, resource.Giga)
	kafkaContainerCpu    = resource.NewMilliQuantity(1000, resource.DecimalSI)
	kafkaJvmOptions      = &strimzi.JvmOptionsSpec{
		Xms: "512m",
		Xmx: "512m",
	}

	// zookeeper vars
	zkVolumeSize      = resource.NewScaledQuantity(10, resource.Giga)
	zkContainerMemory = resource.NewScaledQuantity(1, resource.Giga)
	zkContainerCpu    = resource.NewMilliQuantity(500, resource.DecimalSI)
	zkJvmOptions      = &strimzi.JvmOptionsSpec{
		Xms: "512m",
		Xmx: "512m",
	}
)

// Derive Kafka config based on global constants
var kafkaCRConfig = map[string]string{
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

// BuildKafkaCR builds a Kafka CR
func BuildKafkaCR(kafkaRequest *api.KafkaRequest, kafkaConfig *config.KafkaConfig, keycloakConfig *config.KeycloakConfig, namespace string) *strimzi.Kafka {
	kafkaCR := &strimzi.Kafka{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kafka.strimzi.io/v1beta1",
			Kind:       "Kafka",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kafkaRequest.Name,
			Namespace: namespace,
			Labels:    getKafkaLabels(kafkaConfig),
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
								PersistentClaimStorage: getKafkaStorage(kafkaConfig.KafkaStorageClass),
							},
						},
					},
				},
				Listeners: getKafkaListener(kafkaRequest, kafkaConfig, keycloakConfig),
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
					PersistentClaimStorage: getZookeeperStorage(kafkaConfig.KafkaStorageClass),
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
		kafkaCR = setMultiAZConfig(kafkaCR)
	}

	return kafkaCR
}

func BuildKafkaTLSSecretResource(kafkaRequest *api.KafkaRequest, kafkaConfig *config.KafkaConfig, kafkaNamespace string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getKafkaTLSSecretName(kafkaRequest.Name),
			Namespace: kafkaNamespace,
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte(kafkaConfig.KafkaTLSCert),
			corev1.TLSPrivateKeyKey: []byte(kafkaConfig.KafkaTLSKey),
		},
	}
}

// Need to override the broker route hosts to ensure the length is not above 63 characters which is the max length of the Host on an OpenShift route
func getBrokerOverrides(numOfBrokers int, bootstrapServerHost string) []strimzi.GenericKafkaListenerConfigurationBroker {
	brokerOverrides := []strimzi.GenericKafkaListenerConfigurationBroker{}
	for i := 0; i < numOfBrokers; i++ {
		brokerOverride := strimzi.GenericKafkaListenerConfigurationBroker{
			Host:   fmt.Sprintf("broker-%d-%s", i, bootstrapServerHost),
			Broker: i,
		}
		brokerOverrides = append(brokerOverrides, brokerOverride)
	}

	return brokerOverrides
}

func getKafkaLabels(kafkaConfig *config.KafkaConfig) map[string]string {
	labels := make(map[string]string)
	if kafkaConfig.EnableDedicatedIngress {
		labels["ingressType"] = "sharded" // signal detected by the shared ingress controller
	}

	return labels
}

func getKafkaStorage(storageClass string) strimzi.PersistentClaimStorage {
	kafkaStorage := strimzi.PersistentClaimStorage{
		Size:        kafkaVolumeSize.String(),
		DeleteClaim: &deleteClaim,
	}

	if storageClass != "" {
		kafkaStorage.Class = storageClass
	}

	return kafkaStorage
}

func getZookeeperStorage(storageClass string) strimzi.PersistentClaimStorage {
	zooKeeperStorage := strimzi.PersistentClaimStorage{
		Size:        zkVolumeSize.String(),
		DeleteClaim: &deleteClaim,
	}

	if storageClass != "" {
		zooKeeperStorage.Class = storageClass
	}

	return zooKeeperStorage
}

func getKafkaListener(kafkaRequest *api.KafkaRequest, kafkaConfig *config.KafkaConfig, keycloakConfig *config.KeycloakConfig) []strimzi.GenericKafkaListener {
	var plainOverOauthAuthenticationListener *strimzi.KafkaListenerAuthentication
	var oauthAuthenticationListener *strimzi.KafkaListenerAuthentication

	if keycloakConfig.EnableAuthenticationOnKafka {
		ssoClientID := BuildKeycloakClientNameIdentifier(kafkaRequest.ID)
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

	return []strimzi.GenericKafkaListener{
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
				Brokers:               getBrokerOverrides(kafkaConfig.NumOfBrokers, kafkaRequest.BootstrapServerHost),
				BrokerCertChainAndKey: buildBrokerCertChainAndKeyResource(kafkaConfig, getKafkaTLSSecretName(kafkaRequest.Name)),
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
}

func setMultiAZConfig(kafkaCR *strimzi.Kafka) *strimzi.Kafka {
	kafkaCR.Spec.Kafka.Rack = &strimzi.Rack{
		TopologyKey: "topology.kubernetes.io/zone",
	}
	kafkaCR.Spec.Zookeeper.Template.Pod.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution =
		append(kafkaCR.Spec.Zookeeper.Template.Pod.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution,
			corev1.PodAffinityTerm{
				TopologyKey: "topology.kubernetes.io/zone",
			})
	return kafkaCR
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

func getKafkaTLSSecretName(kafkaRequestName string) string {
	return kafkaRequestName + "-tls-secret"
}
