package services

import (
	"fmt"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services/syncsetresources"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
)

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
		err := errors.ToServiceError(syncsetErr)
		fmt.Println(syncsetErr)
		return nil, errors.New(err.Code, "failed to create syncset '%s' for cluster id '%s': %s", syncset.ID(), clusterId, err)
	}
	return response, nil
}

// Delete removes the syncset from the specified cluster
func (s syncsetService) Delete(syncsetId, clusterId string) (int, *errors.ServiceError) {
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

	resources := []interface{}{
		syncsetresources.BuildProject(namespaceName),
		syncsetresources.BuildKafkaCR(kafkaRequest, kafkaConfig, keycloakConfig, namespaceName),
		syncsetresources.BuildCanary(kafkaRequest, kafkaConfig, sanitizedKafkaName, namespaceName),
	}

	// include admin server resources
	resources = append(resources, syncsetresources.BuildAdminServerResources(kafkaRequest, kafkaConfig, sanitizedKafkaName, namespaceName)...)

	// include Keycloak resources if authentication is enabled
	if keycloakConfig.EnableAuthenticationOnKafka {
		resources = append(resources, syncsetresources.BuildKeycloakResources(kafkaRequest, keycloakConfig, clientSecretValue, namespaceName)...)
	}

	// include Kafka TLS secret if external certs is enabled
	if kafkaConfig.EnableKafkaExternalCertificate {
		resources = append(resources, syncsetresources.BuildKafkaTLSSecretResource(kafkaRequest, kafkaConfig, namespaceName))
	}

	syncsetBuilder = syncsetBuilder.Resources(resources...)
	syncsetId := buildSyncsetIdentifier(kafkaRequest)
	return syncsetBuilder, syncsetId, nil
}
