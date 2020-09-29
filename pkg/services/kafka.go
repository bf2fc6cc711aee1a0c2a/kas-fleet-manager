package services

import (
	"context"
	"fmt"
	"time"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/auth"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/db"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
)

//go:generate moq -out kafkaservice_moq.go . KafkaService
type KafkaService interface {
	Create(kafkaRequest *api.KafkaRequest) *errors.ServiceError
	Get(id string) (*api.KafkaRequest, *errors.ServiceError)
	Delete(id string) *errors.ServiceError
	List(ctx context.Context, listArgs *ListArguments) (api.KafkaList, *api.PagingMeta, *errors.ServiceError)
	RegisterKafkaJob(kafkaRequest *api.KafkaRequest) *errors.ServiceError
	ListByStatus(status KafkaStatus) ([]*api.KafkaRequest, *errors.ServiceError)
	UpdateStatus(id string, status KafkaStatus) *errors.ServiceError
	Update(kafkaRequest *api.KafkaRequest) *errors.ServiceError
}

var _ KafkaService = &kafkaService{}

type kafkaService struct {
	connectionFactory *db.ConnectionFactory
	syncsetService    SyncsetService
	clusterService    ClusterService
}

type KafkaStatus string

func (k KafkaStatus) String() string {
	return string(k)
}

const (
	KafkaRequestStatusAccepted     KafkaStatus = "accepted"
	KafkaRequestStatusProvisioning KafkaStatus = "provisioning"
	KafkaRequestStatusComplete     KafkaStatus = "complete"
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

func (k *kafkaService) ListByStatus(status KafkaStatus) ([]*api.KafkaRequest, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()

	var kafkas []*api.KafkaRequest

	if err := dbConn.Model(&api.KafkaRequest{}).Where("status = ?", status).Scan(&kafkas).Error; err != nil {
		return nil, errors.GeneralError(err.Error())
	}

	return kafkas, nil
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

	// soft delete the kafka request
	if err := dbConn.Delete(&kafkaRequest).Error; err != nil {
		return errors.GeneralError("unable to delete kafka request with id %s: %s", kafkaRequest.ID, err)
	}

	return nil
}

// List returns all Kafka requests belonging to a user.
func (k *kafkaService) List(ctx context.Context, listArgs *ListArguments) (api.KafkaList, *api.PagingMeta, *errors.ServiceError) {
	var kafkaRequestList api.KafkaList
	dbConn := k.connectionFactory.New()
	pagingMeta := &api.PagingMeta{
		Page: listArgs.Page,
		Size: listArgs.Size,
	}

	// filter kafka requests by owner
	user := auth.GetUsernameFromContext(ctx)
	if user == "" {
		return nil, nil, errors.Unauthenticated("user not authenticated")
	}
	dbConn = dbConn.Where("owner = ?", user)

	// set total, limit and paging (based on https://gitlab.cee.redhat.com/service/api-guidelines#user-content-paging)
	dbConn.Model(&kafkaRequestList).Count(&pagingMeta.Total)
	if pagingMeta.Size > pagingMeta.Total {
		pagingMeta.Size = pagingMeta.Total
	}
	dbConn = dbConn.Offset((pagingMeta.Page - 1) * pagingMeta.Size).Limit(pagingMeta.Size)

	// default the order by name
	dbConn = dbConn.Order("name")

	// execute query
	if err := dbConn.Find(&kafkaRequestList).Error; err != nil {
		return kafkaRequestList, pagingMeta, errors.GeneralError("Unable to list kafka requests for %s: %s", user, err)
	}

	return kafkaRequestList, pagingMeta, nil
}

func (k kafkaService) Update(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
	dbConn := k.connectionFactory.New()

	if err := dbConn.Model(kafkaRequest).Update(kafkaRequest).Error; err != nil {
		return errors.GeneralError("failed to update: %s", err.Error())
	}
	return nil
}

func (k kafkaService) UpdateStatus(id string, status KafkaStatus) *errors.ServiceError {
	dbConn := k.connectionFactory.New()

	if err := dbConn.Table("kafka_requests").Where("id = ?", id).Update("status", status).Error; err != nil {
		return errors.GeneralError("failed to update status: %s", err.Error())
	}

	return nil
}
