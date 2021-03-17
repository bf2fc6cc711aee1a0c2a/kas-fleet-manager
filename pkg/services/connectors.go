package services

import (
	"context"
	goerrors "errors"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/signalbus"
	"github.com/jinzhu/gorm"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

//go:generate moq -out connectors_moq.go . ConnectorsService
type ConnectorsService interface {
	Create(ctx context.Context, resource *api.Connector) *errors.ServiceError
	Get(ctx context.Context, id string, tid string) (*api.Connector, *errors.ServiceError)
	List(ctx context.Context, kid string, listArgs *ListArguments, tid string) (api.ConnectorList, *api.PagingMeta, *errors.ServiceError)
	Update(ctx context.Context, resource *api.Connector) *errors.ServiceError
	Delete(ctx context.Context, id string) *errors.ServiceError
	ForEachInStatus(statuses []api.ConnectorStatus, f func(*api.Connector) *errors.ServiceError) *errors.ServiceError
}

var _ ConnectorsService = &connectorsService{}

type connectorsService struct {
	connectionFactory *db.ConnectionFactory
	bus               signalbus.SignalBus
}

func NewConnectorsService(connectionFactory *db.ConnectionFactory, bus signalbus.SignalBus) *connectorsService {
	return &connectorsService{
		connectionFactory: connectionFactory,
		bus:               bus,
	}
}

// Create creates a connector in the database
func (k *connectorsService) Create(ctx context.Context, resource *api.Connector) *errors.ServiceError {
	kid := resource.KafkaID
	if kid == "" {
		return errors.Validation("kafka id is undefined")
	}

	dbConn := k.connectionFactory.New()
	resource.Status = api.ConnectorStatusAssigning
	if err := dbConn.Save(resource).Error; err != nil {
		return errors.GeneralError("failed to create connector: %v", err)
	}

	// read it back.... to get the updated version...
	dbConn = k.connectionFactory.New().Where("id = ?", resource.ID)
	if err := dbConn.First(&resource).Error; err != nil {
		return handleGetError("Connector", "id", resource.ID, err)
	}

	// TODO: increment connector metrics
	// metrics.IncreaseStatusCountMetric(constants.KafkaRequestStatusAccepted.String())
	return nil
}

// Get gets a connector by id from the database
func (k *connectorsService) Get(ctx context.Context, id string, tid string) (*api.Connector, *errors.ServiceError) {
	if id == "" {
		return nil, errors.Validation("connector id is undefined")
	}

	dbConn := k.connectionFactory.New()
	var resource api.Connector
	dbConn = dbConn.Where("id = ?", id)

	var err *errors.ServiceError
	dbConn, err = filterToOwnerOrOrg(ctx, dbConn)
	if err != nil {
		return nil, err
	}

	if tid != "" {
		dbConn = dbConn.Where("connector_type_id = ?", tid)
	}

	if err := dbConn.First(&resource).Error; err != nil {
		return nil, handleGetError("Connector", "id", id, err)
	}
	return &resource, nil
}

func filterToOwnerOrOrg(ctx context.Context, dbConn *gorm.DB) (*gorm.DB, *errors.ServiceError) {
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return dbConn, errors.Unauthenticated("user not authenticated")
	}
	owner := auth.GetUsernameFromClaims(claims)
	if owner == "" {
		return dbConn, errors.Unauthenticated("user not authenticated")
	}

	orgId := auth.GetOrgIdFromClaims(claims)
	userIsAllowedAsServiceAccount := auth.GetUserIsAllowedAsServiceAccountFromContext(ctx)

	// filter by organisationId if a user is part of an organisation and is not allowed as a service account
	filterByOrganisationId := !userIsAllowedAsServiceAccount && orgId != ""
	if filterByOrganisationId {
		dbConn = dbConn.Where("organisation_id = ?", orgId)
	} else {
		dbConn = dbConn.Where("owner = ?", owner)
	}
	return dbConn, nil
}

// Delete deletes a connector from the database.
func (k *connectorsService) Delete(ctx context.Context, id string) *errors.ServiceError {
	if id == "" {
		return errors.Validation("id is undefined")
	}
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return errors.Unauthenticated("user not authenticated")
	}
	owner := auth.GetUsernameFromClaims(claims)
	if owner == "" {
		return errors.Unauthenticated("user not authenticated")
	}

	dbConn := k.connectionFactory.New()
	var resource api.Connector
	if err := dbConn.Where("owner = ? AND id = ?", owner, id).First(&resource).Error; err != nil {
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
	var resourceList api.ConnectorList
	dbConn := k.connectionFactory.New()
	pagingMeta := &api.PagingMeta{
		Page: listArgs.Page,
		Size: listArgs.Size,
	}

	dbConn = dbConn.Where("kafka_id = ?", kid)

	var err *errors.ServiceError
	dbConn, err = filterToOwnerOrOrg(ctx, dbConn)
	if err != nil {
		return nil, nil, err
	}

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
		return resourceList, pagingMeta, errors.GeneralError("Unable to list connectors: %s", err)
	}

	return resourceList, pagingMeta, nil
}

func (k connectorsService) Update(ctx context.Context, resource *api.Connector) *errors.ServiceError {

	// If the version is set, then lets verify that the version has not changed...
	if resource.Version != 0 {
		dbConn := k.connectionFactory.New()
		dbConn = dbConn.Where("id = ? AND version = ?", resource.ID, resource.Version)
		t := api.Connector{}
		if err := dbConn.First(&t).Error; err != nil {
			return errors.BadRequest("resource version changed")
		}
	}

	dbConn := k.connectionFactory.New()
	if err := dbConn.Model(resource).Update(resource).Error; err != nil {
		return errors.GeneralError("failed to update: %s", err.Error())
	}

	// read it back.... to get the updated version...
	dbConn = k.connectionFactory.New().Where("id = ?", resource.ID)
	if err := dbConn.First(&resource).Error; err != nil {
		return handleGetError("Connector", "id", resource.ID, err)
	}

	if resource.ClusterID != "" {
		_ = db.AddPostCommitAction(ctx, func() {
			k.bus.Notify(fmt.Sprintf("/kafka-connector-clusters/%s/connectors", resource.ClusterID))
		})
	}

	return nil
}

func (k connectorsService) ForEachInStatus(statuses []api.ConnectorStatus, f func(*api.Connector) *errors.ServiceError) *errors.ServiceError {
	dbConn := k.connectionFactory.New().Model(&api.Connector{})
	dbConn = dbConn.Where("status IN (?)", statuses)
	rows, err := dbConn.Order("version").Rows()
	if err != nil {
		if goerrors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return errors.GeneralError("Unable to list connectors: %s", err)
	}
	defer rows.Close()

	for rows.Next() {
		resource := api.Connector{}
		// ScanRows is a method of `gorm.DB`, it can be used to scan a row into a struct
		err := dbConn.ScanRows(rows, &resource)
		if err != nil {
			return errors.GeneralError("Unable to scan connector: %s", err)
		}
		if serr := f(&resource); serr != nil {
			return serr
		}
	}
	return nil
}
