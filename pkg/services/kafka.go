package services

import (
	"time"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/db"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
)

type KafkaService interface {
	Create(kafkaRequest *api.KafkaRequest) *errors.ServiceError
	Get(id string) (*api.KafkaRequest, *errors.ServiceError)
	RegisterKafkaJob(kafkaRequest *api.KafkaRequest) *errors.ServiceError
}

var _ KafkaService = &kafkaService{}

type kafkaService struct {
	connectionFactory *db.ConnectionFactory
	syncsetService    SyncsetService
}

type kafkaStatus string

func (k kafkaStatus) String() string {
	return string(k)
}

const (
	KafkaRequestStatusAccepted kafkaStatus = "accepted"
)

func NewKafkaService(connectionFactory *db.ConnectionFactory, syncsetService SyncsetService) *kafkaService {
	return &kafkaService{
		connectionFactory: connectionFactory,
		syncsetService:    syncsetService,
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
// The kafka object in the database will be updated with a completed_at
// timestamp and the corresponding cluster identifier.
func (k *kafkaService) Create(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
	dbConn := k.connectionFactory.New()

	// TODO: Call out to the cluster service
	//  we want to choose the cluster based on current capacity, region
	//  and provider
	cluster := &api.Cluster{
		ClusterID: "000",
	}

	// update the kafka object to point to the cluster id
	kafkaRequest.ClusterID = cluster.ClusterID

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

	// update kafka completed_at timestamp
	if err := dbConn.Model(&kafkaRequest).Update("completed_at", time.Now()).Error; err != nil {
		return errors.GeneralError("failed to update completed at time: %v", err)
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
