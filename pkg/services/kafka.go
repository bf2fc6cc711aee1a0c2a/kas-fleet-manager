package services

import (
	"fmt"
	"time"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/db"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
)

type KafkaService interface {
	Create(kafkaRequest *api.KafkaRequest) *errors.ServiceError
	Get(id string) (*api.KafkaRequest, *errors.ServiceError)
	Delete(id string) *errors.ServiceError
	RegisterKafkaJob(kafkaRequest *api.KafkaRequest) *errors.ServiceError
}

var _ KafkaService = &kafkaService{}

type kafkaService struct {
	connectionFactory *db.ConnectionFactory
	syncsetService    SyncsetService
	clusterService    ClusterService
}

type kafkaStatus string

func (k kafkaStatus) String() string {
	return string(k)
}

const (
	KafkaRequestStatusAccepted kafkaStatus = "accepted"
)

func NewKafkaService(connectionFactory *db.ConnectionFactory, syncsetService SyncsetService, clusterService ClusterService) *kafkaService {
	return &kafkaService{
		connectionFactory: connectionFactory,
		syncsetService:    syncsetService,
		clusterService:    clusterService,
	}
}

// RegisterKafkaJob registers a new job in the kafka table
func (k *kafkaService) RegisterKafkaJob(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
	dbConn := k.connectionFactory.New()
	kafkaRequest.Status = string(KafkaRequestStatusAccepted)
	if err := dbConn.Save(kafkaRequest).Error; err != nil {
		return errors.GeneralError("failed to create kafka job: %v", err)
	}
	return nil
}

// Create will create a new kafka cr with the given configuration,
// and sync it via a syncset to an available cluster with capacity
// in the desired region for the desired cloud provider.
// The kafka object in the database will be updated with a updated_at
// timestamp and the corresponding cluster identifier.
func (k *kafkaService) Create(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
	dbConn := k.connectionFactory.New()

	clusterDNS, err := k.clusterService.GetClusterDNS(kafkaRequest.ClusterID)
	if err != nil || clusterDNS == "" {
		return errors.GeneralError("error retreiving cluster DNS: %v", err)
	}

	truncatedKafkaIdentifier := buildTruncateKafkaIdentifier(kafkaRequest)
	kafkaRequest.BootstrapServerHost = fmt.Sprintf("%s.%s", truncatedKafkaIdentifier, clusterDNS)

	// create the syncset builder
	syncsetBuilder, syncsetId, err := newKafkaSyncsetBuilder(kafkaRequest)
	if err != nil {
		return errors.GeneralError("error creating kafka syncset builder: %v", err)
	}

	// create the syncset
	_, err = k.syncsetService.Create(syncsetBuilder, syncsetId, kafkaRequest.ClusterID)
	if err != nil {
		return errors.GeneralError("error creating syncset: %v", err)
	}

	kafkaRequest.UpdatedAt = time.Now()
	// update kafka updated_at timestamp
	if err := dbConn.Save(kafkaRequest).Error; err != nil {
		return errors.GeneralError("failed to update kafka request: %v", err)
	}

	return nil
}

func (k *kafkaService) Get(id string) (*api.KafkaRequest, *errors.ServiceError) {
	if id == "" {
		return nil, errors.Validation("id is undefined")
	}

	dbConn := k.connectionFactory.New()
	var kafkaRequest api.KafkaRequest
	if err := dbConn.Where("id = ?", id).First(&kafkaRequest).Error; err != nil {
		return nil, handleGetError("KafkaResource", "id", id, err)
	}
	return &kafkaRequest, nil
}

// Delete deletes a kafka request and its corresponding syncset from
// the associated cluster it was deployed on. Deleting the syncset will
// delete all resources (Kafka CR, Project) associated with the syncset.
// The kafka object in the database will be updated with a deleted_at
// timestamp.
func (k *kafkaService) Delete(id string) *errors.ServiceError {
	if id == "" {
		return errors.Validation("id is undefined")
	}

	dbConn := k.connectionFactory.New()
	var kafkaRequest api.KafkaRequest
	if err := dbConn.Where("id = ?", id).First(&kafkaRequest).Error; err != nil {
		return handleGetError("KafkaResource", "id", id, err)
	}

	// delete the syncset
	syncsetId := buildSyncsetIdentifier(&kafkaRequest)
	err := k.syncsetService.Delete(syncsetId, kafkaRequest.ClusterID)
	if err != nil {
		return errors.GeneralError("error deleting syncset: %v", err)
	}

	// update the deletion timestamp
	if err := dbConn.Model(&kafkaRequest).Update("deleted_at", time.Now()).Error; err != nil {
		return errors.GeneralError("unable to update kafka request with id %s: %s", kafkaRequest.ID, err)
	}

	return nil
}
