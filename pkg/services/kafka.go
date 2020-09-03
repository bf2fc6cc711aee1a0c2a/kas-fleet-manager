package services

import (
	"context"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/db"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"time"
)

type KafkaService interface {
	Create(ctx context.Context, kafka *api.Kafka) *errors.ServiceError
}

type kafkaService struct {
	connectionFactory *db.ConnectionFactory
	syncsetService    SyncsetService
}

var _ KafkaService = &kafkaService{}

func NewKafkaService(connectionFactory *db.ConnectionFactory, syncsetService SyncsetService) *kafkaService {
	return &kafkaService{
		connectionFactory: connectionFactory,
		syncsetService:    syncsetService,
	}
}

// Create will create a new kafka cr with the given configuration,
// and sync it via a syncset to an available cluster with capacity
// in the desired region for the desired cloud provider.
// The kafka object in the database will be updated with a completed_at
// timestamp and the corresponding cluster identifier.
func (k *kafkaService) Create(ctx context.Context, kafka *api.Kafka) *errors.ServiceError {
	dbConn := k.connectionFactory.New()

	// TODO: Interface for finding available clusters
	//  we want to choose the cluster based on current capacity, region
	//  and provider
	cluster := &api.Cluster{
		ClusterID: "0000",
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
