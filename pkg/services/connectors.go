package services

import (
	"context"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/auth"
	constants "gitlab.cee.redhat.com/service/managed-services-api/pkg/constants"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/db"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
)

//go:generate moq -out connectors_moq.go . ConnectorsService
type ConnectorsService interface {
	Create(ctx context.Context, resource *api.Connector) *errors.ServiceError
	Get(ctx context.Context, kid string, id string, tid string) (*api.Connector, *errors.ServiceError)
	List(ctx context.Context, kid string, listArgs *ListArguments, tid string) (api.ConnectorList, *api.PagingMeta, *errors.ServiceError)
	Update(ctx context.Context, resource *api.Connector) *errors.ServiceError
	Delete(ctx context.Context, kid string, id string) *errors.ServiceError
}

var _ ConnectorsService = &connectorsService{}

type connectorsService struct {
	connectionFactory *db.ConnectionFactory
}

func NewConnectorsService(connectionFactory *db.ConnectionFactory) *connectorsService {
	return &connectorsService{
		connectionFactory: connectionFactory,
	}
}

// Create creates a connector in the database
func (k *connectorsService) Create(ctx context.Context, resource *api.Connector) *errors.ServiceError {
	kid := resource.KafkaID
	if kid == "" {
		return errors.Validation("kafka id is undefined")
	}

	dbConn := k.connectionFactory.New()
	resource.Status = constants.KafkaRequestStatusAccepted.String()
	if err := dbConn.Save(resource).Error; err != nil {
		return errors.GeneralError("failed to create connector: %v", err)
	}
	// TODO: increment connector metrics
	// metrics.IncreaseStatusCountMetric(constants.KafkaRequestStatusAccepted.String())
	return nil
}

// Get gets a connector by id from the database
func (k *connectorsService) Get(ctx context.Context, kid string, id string, tid string) (*api.Connector, *errors.ServiceError) {
	if kid == "" {
		return nil, errors.Validation("kafka id is undefined")
	}
	if id == "" {
		return nil, errors.Validation("connector id is undefined")
	}

	owner := auth.GetUsernameFromContext(ctx)
	if owner == "" {
		return nil, errors.Unauthenticated("user not authenticated")
	}

	dbConn := k.connectionFactory.New()
	var resource api.Connector
	dbConn = dbConn.Where("owner = ? AND id = ? AND kafka_id = ?", owner, id, kid)

	if tid != "" {
		dbConn = dbConn.Where("connector_type_id = ?", tid)
	}

	if err := dbConn.First(&resource).Error; err != nil {
		return nil, handleGetError("Connector", "id", id, err)
	}
	return &resource, nil
}

// Delete deletes a connector from the database.
func (k *connectorsService) Delete(ctx context.Context, kid string, id string) *errors.ServiceError {
	if kid == "" {
		return errors.Validation("kafka id is undefined")
	}
	if id == "" {
		return errors.Validation("id is undefined")
	}

	owner := auth.GetUsernameFromContext(ctx)
	if owner == "" {
		return errors.Unauthenticated("user not authenticated")
	}

	dbConn := k.connectionFactory.New()
	var resource api.Connector
	if err := dbConn.Where("owner = ? AND id = ? AND kafka_id = ?", owner, id, kid).First(&resource).Error; err != nil {
		return handleGetError("Connector", "id", id, err)
	}

	// TODO: implement soft delete instead?
	if err := dbConn.Delete(&resource).Error; err != nil {
		return errors.GeneralError("unable to delete connector with id %s: %s", resource.ID, err)
	}

	return nil
}

// List returns all connectors visible to the user within the requested paging window.
func (k *connectorsService) List(ctx context.Context, kid string, listArgs *ListArguments, tid string) (api.ConnectorList, *api.PagingMeta, *errors.ServiceError) {
	if kid == "" {
		return nil, nil, errors.Validation("kafka id is undefined")
	}

	var resourceList api.ConnectorList
	dbConn := k.connectionFactory.New()
	pagingMeta := &api.PagingMeta{
		Page: listArgs.Page,
		Size: listArgs.Size,
	}

	// filter connectors requests by owner
	owner := auth.GetUsernameFromContext(ctx)
	if owner == "" {
		return nil, nil, errors.Unauthenticated("user not authenticated")
	}
	dbConn = dbConn.Where("owner = ? AND kafka_id = ?", owner, kid)

	if tid != "" {
		dbConn = dbConn.Where("connector_type_id = ?", tid)
	}

	// set total, limit and paging (based on https://gitlab.cee.redhat.com/service/api-guidelines#user-content-paging)
	dbConn.Model(&resourceList).Count(&pagingMeta.Total)
	if pagingMeta.Size > pagingMeta.Total {
		pagingMeta.Size = pagingMeta.Total
	}
	dbConn = dbConn.Offset((pagingMeta.Page - 1) * pagingMeta.Size).Limit(pagingMeta.Size)

	// default the order by name
	dbConn = dbConn.Order("name")

	// execute query
	if err := dbConn.Find(&resourceList).Error; err != nil {
		return resourceList, pagingMeta, errors.GeneralError("Unable to list connectors for %s: %s", owner, err)
	}

	return resourceList, pagingMeta, nil
}

func (k connectorsService) Update(ctx context.Context, resource *api.Connector) *errors.ServiceError {
	kid := resource.KafkaID
	if kid == "" {
		return errors.Validation("kafka id is undefined")
	}

	owner := auth.GetUsernameFromContext(ctx)
	if owner == "" {
		return errors.Unauthenticated("user not authenticated")
	}

	dbConn := k.connectionFactory.New()

	if err := dbConn.Where("owner = ? AND id = ? AND kafka_id = ?", owner, resource.ID, kid).Model(resource).Update(resource).Error; err != nil {
		return errors.GeneralError("failed to update: %s", err.Error())
	}
	return nil
}
