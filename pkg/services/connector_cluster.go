package services

import (
	"context"
	goerrors "errors"
	"github.com/jinzhu/gorm"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

type ConnectorClusterService interface {
	Create(ctx context.Context, resource *api.ConnectorCluster) *errors.ServiceError
	Get(ctx context.Context, id string) (*api.ConnectorCluster, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	List(ctx context.Context, listArgs *ListArguments) (api.ConnectorClusterList, *api.PagingMeta, *errors.ServiceError)
	ListConnectors(ctx context.Context, id string, listArgs *ListArguments, gtVersion int64) (api.ConnectorList, *api.PagingMeta, *errors.ServiceError)
	Update(ctx context.Context, resource *api.ConnectorCluster) *errors.ServiceError
	UpdateConnectorClusterStatus(ctx context.Context, id string, status api.ConnectorClusterStatus) *errors.ServiceError
	UpdateConnectorStatus(ctx context.Context, id string, cid string, status api.ConnectorStatus) *errors.ServiceError
	FindReadyCluster(owner string, group string) (*api.ConnectorCluster, *errors.ServiceError)
}

var _ ConnectorClusterService = &connectorClusterService{}

type connectorClusterService struct {
	connectionFactory *db.ConnectionFactory
}

func NewConnectorClusterService(connectionFactory *db.ConnectionFactory) *connectorClusterService {
	return &connectorClusterService{
		connectionFactory: connectionFactory,
	}
}

// Create creates a connector cluster in the database
func (k *connectorClusterService) Create(ctx context.Context, resource *api.ConnectorCluster) *errors.ServiceError {
	dbConn := k.connectionFactory.New()
	if resource.Name == "" {
		resource.Name = "New Cluster"
	}
	if resource.AddonGroup == "" {
		resource.AddonGroup = "default"
	}
	resource.Status = api.ConnectorClusterStatusUnconnected
	if err := dbConn.Save(resource).Error; err != nil {
		return errors.GeneralError("failed to create connector: %v", err)
	}
	// TODO: increment connector cluster metrics
	// metrics.IncreaseStatusCountMetric(constants.KafkaRequestStatusAccepted.String())
	return nil
}

// Get gets a connector by id from the database
func (k *connectorClusterService) Get(ctx context.Context, id string) (*api.ConnectorCluster, *errors.ServiceError) {
	if id == "" {
		return nil, errors.Validation("connector cluster id is undefined")
	}

	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, errors.Unauthenticated("user not authenticated")
	}
	owner := auth.GetUsernameFromClaims(claims)
	if owner == "" {
		return nil, errors.Unauthenticated("user not authenticated")
	}
	if owner == "" {
		return nil, errors.Unauthenticated("user not authenticated")
	}

	dbConn := k.connectionFactory.New()
	var resource api.ConnectorCluster
	dbConn = dbConn.Where("owner = ? AND id = ?", owner, id)

	if err := dbConn.First(&resource).Error; err != nil {
		return nil, handleGetError("Connector", "id", id, err)
	}
	return &resource, nil
}

// Delete deletes a connector cluster from the database.
func (k *connectorClusterService) Delete(ctx context.Context, id string) *errors.ServiceError {
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
	var resource api.ConnectorCluster
	if err := dbConn.Where("owner = ? AND id = ?", owner, id).First(&resource).Error; err != nil {
		return handleGetError("Connector", "id", id, err)
	}

	// TODO: implement soft delete instead?
	if err := dbConn.Delete(&resource).Error; err != nil {
		return errors.GeneralError("unable to delete connector with id %s: %s", resource.ID, err)
	}

	return nil
}

// List returns all connector clusters visible to the user within the requested paging window.
func (k *connectorClusterService) List(ctx context.Context, listArgs *ListArguments) (api.ConnectorClusterList, *api.PagingMeta, *errors.ServiceError) {
	var resourceList api.ConnectorClusterList
	dbConn := k.connectionFactory.New()
	pagingMeta := &api.PagingMeta{
		Page: listArgs.Page,
		Size: listArgs.Size,
	}

	// filter connectors requests by owner
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, nil, errors.Unauthenticated("user not authenticated")
	}
	owner := auth.GetUsernameFromClaims(claims)
	if owner == "" {
		return nil, nil, errors.Unauthenticated("user not authenticated")
	}

	dbConn = dbConn.Where("owner = ?", owner)

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

func (k connectorClusterService) Update(ctx context.Context, resource *api.ConnectorCluster) *errors.ServiceError {
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return errors.Unauthenticated("user not authenticated")
	}
	owner := auth.GetUsernameFromClaims(claims)
	if owner == "" {
		return errors.Unauthenticated("user not authenticated")
	}

	dbConn := k.connectionFactory.New()
	if err := dbConn.Where("owner = ? AND id = ?", owner, resource.ID).Model(resource).Updates(resource).Error; err != nil {
		return errors.GeneralError("failed to update: %s", err.Error())
	}
	return nil
}

func (k *connectorClusterService) UpdateConnectorClusterStatus(ctx context.Context, id string, status api.ConnectorClusterStatus) *errors.ServiceError {
	if id == "" {
		return errors.Validation("id is undefined")
	}

	dbConn := k.connectionFactory.New()

	m := api.ConnectorCluster{
		Meta: api.Meta{
			ID: id,
		},
	}

	if err := dbConn.Model(&m).Update("status", status).Error; err != nil {
		return errors.GeneralError("failed to update status: %s", err.Error())
	}
	return nil
}

// List returns all connectors assigned to the cluster
func (k *connectorClusterService) ListConnectors(ctx context.Context, id string, listArgs *ListArguments, gtVersion int64) (api.ConnectorList, *api.PagingMeta, *errors.ServiceError) {
	var resourceList api.ConnectorList
	dbConn := k.connectionFactory.New()
	pagingMeta := &api.PagingMeta{
		Page: listArgs.Page,
		Size: listArgs.Size,
	}

	dbConn = dbConn.Where("cluster_id = ?", id)
	if gtVersion != 0 {
		dbConn = dbConn.Where("version > ?", gtVersion)
	}

	// set total, limit and paging (based on https://gitlab.cee.redhat.com/service/api-guidelines#user-content-paging)
	dbConn.Model(&resourceList).Count(&pagingMeta.Total)
	if pagingMeta.Size > pagingMeta.Total {
		pagingMeta.Size = pagingMeta.Total
	}
	dbConn = dbConn.Offset((pagingMeta.Page - 1) * pagingMeta.Size).Limit(pagingMeta.Size)

	// default the order by version
	dbConn = dbConn.Order("version")

	// execute query
	if err := dbConn.Find(&resourceList).Error; err != nil {
		return resourceList, pagingMeta, errors.GeneralError("Unable to list connectors for cluster %s: %s", id, err)
	}

	return resourceList, pagingMeta, nil
}

func (k *connectorClusterService) UpdateConnectorStatus(ctx context.Context, id string, cid string, status api.ConnectorStatus) *errors.ServiceError {
	if id == "" {
		return errors.Validation("id is undefined")
	}
	if cid == "" {
		return errors.Validation("cid is undefined")
	}

	dbConn := k.connectionFactory.New()

	m := api.Connector{
		Meta: api.Meta{
			ID: cid,
		},
	}

	if err := dbConn.Model(&m).Where("cluster_id = ?", id).Update("status", status).Error; err != nil {
		return errors.GeneralError("failed to update status: %s", err.Error())
	}
	return nil
}

func (k *connectorClusterService) FindReadyCluster(owner string, group string) (*api.ConnectorCluster, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()
	var resource api.ConnectorCluster
	dbConn = dbConn.Where("owner = ? AND addon_group = ? AND status = ?", owner, group, api.ConnectorClusterStatusReady)

	if err := dbConn.First(&resource).Error; err != nil {
		if goerrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errors.GeneralError("failed to query ready addon connector cluster: %v", err.Error())
	}
	return &resource, nil
}
