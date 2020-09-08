package services

import (
	"time"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/db"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
)

type KafkaService interface {
	Create(kafka *api.Kafka) *errors.ServiceError
	RegisterKafkaJob(kafka *api.Kafka) *errors.ServiceError
}

type kafkaService struct {
	connectionFactory *db.ConnectionFactory
	syncsetService    SyncsetService
	clusterService    ClusterService
}

type kafkaStatus string

var _ KafkaService = &kafkaService{}

const (
	statusAccepted kafkaStatus = "accepted"
)

func NewKafkaService(connectionFactory *db.ConnectionFactory, syncsetService SyncsetService, clusterService ClusterService) *kafkaService {
	return &kafkaService{
		connectionFactory: connectionFactory,
		syncsetService:    syncsetService,
		clusterService:    clusterService,
	}
}

// RegisterKafkaJob registers a new job in the kafka table
func (k *kafkaService) RegisterKafkaJob(kafka *api.Kafka) *errors.ServiceError {
	dbConn := k.connectionFactory.New()
	kafka.Owner = "dummy-owner"
	kafka.Status = string(statusAccepted)
	if err := dbConn.Save(kafka).Error; err != nil {
		return errors.GeneralError("failed to create kafka job:", err)
	}
	return nil
}

// Create will create a new kafka cr with the given configuration,
// and sync it via a syncset to an available cluster with capacity
// in the desired region for the desired cloud provider.
// The kafka object in the database will be updated with a completed_at
// timestamp and the corresponding cluster identifier.
func (k *kafkaService) Create(kafka *api.Kafka) *errors.ServiceError {
	dbConn := k.connectionFactory.New()

	// TODO: Call out to the cluster service
	//  we want to choose the cluster based on current capacity, region
	//  and provider
	cluster := &api.Cluster{
		ClusterID: "000",
	}

	// update the kafka object to point to the cluster id
	kafka.ClusterID = cluster.ClusterID

	// create the syncset builder
	syncsetBuilder, syncsetId, err := newKafkaSyncsetBuilder(kafka)
	if err != nil {
		return errors.GeneralError("error creating kafka syncset builder:", err)
	}

	// create the syncset
	_, err = k.syncsetService.Create(syncsetBuilder, syncsetId, kafka.ClusterID)
	if err != nil {
		return errors.GeneralError("error creating syncset:", err)
	}

	// update kafka completed_at timestamp
	if err := dbConn.Model(&kafka).Update("completed_at", time.Now()).Error; err != nil {
		return errors.GeneralError("failed to update completed at time:", err)
	}

	return nil
}
