package services

import (
	"fmt"
	sdkClient "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	projectv1 "github.com/openshift/api/project/v1"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	strimzi "gitlab.cee.redhat.com/service/managed-services-api/pkg/api/kafka.strimzi.io/v1beta1"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	numOfBrokers    = 3
	numOfZookeepers = 3
)

//go:generate moq -out syncset_mock.go . SyncsetService
type SyncsetService interface {
	Create(syncsetBuilder *cmv1.SyncsetBuilder, syncsetId, clusterId string) (*cmv1.Syncset, *errors.ServiceError)
	Delete(syncsetId, clusterId string) *errors.ServiceError
}

func NewSyncsetService(ocmClient *sdkClient.Connection) SyncsetService {
	return &syncsetService{
		ocmClient: ocmClient,
	}
}

var _ SyncsetService = &syncsetService{}

type syncsetService struct {
	ocmClient *sdkClient.Connection
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
	clustersResource := s.ocmClient.ClustersMgmt().V1().Clusters()
	response, syncsetErr := clustersResource.Cluster(clusterId).
		ExternalConfiguration().
		Syncsets().
		Add().
		Body(syncset).
		Send()
	if syncsetErr != nil {
		fmt.Println(syncsetErr)
		return nil, errors.GeneralError(fmt.Sprintf("failed to create syncset: %s for cluster id: %s", syncset.ID(), clusterId), syncsetErr)
	}
	return response.Body(), nil
}

func (s syncsetService) Delete(syncsetId, clusterId string) *errors.ServiceError {
	// create the syncset on the cluster
	clustersResource := s.ocmClient.ClustersMgmt().V1().Clusters()
	_, syncsetErr := clustersResource.Cluster(clusterId).
		ExternalConfiguration().
		Syncsets().
		Syncset(syncsetId).
		Delete().
		Send()
	if syncsetErr != nil {
		return errors.GeneralError(fmt.Sprintf("failed to delete syncset: %s for cluster id: %s", syncsetId, clusterId), syncsetErr)
	}
	return nil
}

// syncset builder for a kafka/strimzi custom resource
func newKafkaSyncsetBuilder(kafkaRequest *api.KafkaRequest) (*cmv1.SyncsetBuilder, string, *errors.ServiceError) {
	kafkaOwnerID := buildKafkaNamespaceIdentifier(kafkaRequest)
	kafkaIdentifier := validateClusterNameAndReplaceSpecialChar(kafkaOwnerID)


	// Need to override the broker route hosts to ensure the length is not above 63 characters which is the max length of the Host on an OpenShift route
	brokerOverrides := []strimzi.RouteListenerBrokerOverride{}
	for i := 0; i < numOfBrokers; i++ {
		brokerOverride := strimzi.RouteListenerBrokerOverride{
			Host:   fmt.Sprintf("broker-%d-%s", i, kafkaRequest.BootstrapServerHost),
			Broker: i,
		}
		brokerOverrides = append(brokerOverrides, brokerOverride)
	}

	// build array of objects to be created by the syncset
	kafkaCR := &strimzi.Kafka{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kafka.strimzi.io/v1beta1",
			Kind:       "Kafka",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kafkaRequest.Name,
			Namespace: kafkaIdentifier,
		},
		Spec: strimzi.KafkaSpec{
			Kafka: strimzi.KafkaClusterSpec{
				Replicas: numOfBrokers,
				Storage: strimzi.Storage{
					Type: strimzi.Ephemeral,
				},
				Listeners: strimzi.KafkaListeners{
					Plain: &strimzi.KafkaListenerPlain{},
					TLS:   &strimzi.KafkaListenerTLS{},
					External: &strimzi.KafkaListenerExternal{
						Type: strimzi.Route,
						KafkaListenerExternalRoute: strimzi.KafkaListenerExternalRoute{
							Overrides: &strimzi.RouteListenerOverride{
								Bootstrap: &strimzi.RouteListenerBootstrapOverride{
									Host: kafkaRequest.BootstrapServerHost,
								},
								Brokers: brokerOverrides,
							},
						},
					},
				},
			},
			Zookeeper: strimzi.ZookeeperClusterSpec{
				Replicas: numOfZookeepers,
				Storage: strimzi.Storage{
					Type: strimzi.Ephemeral,
				},
			},
		},
	}

	resources := []interface{}{
		&projectv1.Project{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "project.openshift.io/v1",
				Kind:       "Project",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: kafkaIdentifier,
			},
		},
		kafkaCR,
	}

	syncsetBuilder := cmv1.NewSyncset()
	syncsetBuilder = syncsetBuilder.Resources(resources...)
	syncsetId := buildSyncsetIdentifier(kafkaRequest)
	return syncsetBuilder, syncsetId, nil
}
