package services

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm/clause"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/phase"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/vault"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/queryparser"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sso"
	"github.com/golang/glog"
	"gorm.io/gorm"
)

type ConnectorClusterService interface {
	Create(ctx context.Context, resource *dbapi.ConnectorCluster) *errors.ServiceError
	Get(ctx context.Context, id string) (*dbapi.ConnectorCluster, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	List(ctx context.Context, listArgs *services.ListArguments) (dbapi.ConnectorClusterList, *api.PagingMeta, *errors.ServiceError)
	Update(ctx context.Context, resource *dbapi.ConnectorCluster) *errors.ServiceError
	UpdateConnectorClusterStatus(ctx context.Context, id string, status dbapi.ConnectorClusterStatus) *errors.ServiceError
	GetConnectorClusterStatus(ctx context.Context, id string) (dbapi.ConnectorClusterStatus, *errors.ServiceError)

	SaveDeployment(ctx context.Context, resource *dbapi.ConnectorDeployment) *errors.ServiceError
	UpdateDeployment(resource *dbapi.ConnectorDeployment) *errors.ServiceError
	ListConnectorDeployments(ctx context.Context, clusterId string, filterChannelUpdates bool, filterOperatorUpdates bool, includeDanglingDeploymentsOnly bool, listArgs *services.ListArguments, gtVersion int64) (dbapi.ConnectorDeploymentList, *api.PagingMeta, *errors.ServiceError)
	UpdateConnectorDeploymentStatus(ctx context.Context, status dbapi.ConnectorDeploymentStatus) *errors.ServiceError
	FindAvailableNamespace(owner string, orgId string, namespaceId *string) (*dbapi.ConnectorNamespace, *errors.ServiceError)
	GetDeploymentByConnectorId(ctx context.Context, connectorID string) (dbapi.ConnectorDeployment, *errors.ServiceError)
	GetDeployment(ctx context.Context, id string) (dbapi.ConnectorDeployment, *errors.ServiceError)
	CleanupDeployments() *errors.ServiceError
	ReconcileEmptyDeletingClusters(ctx context.Context, clusterIds []string) (int, []*errors.ServiceError)
	ReconcileNonEmptyDeletingClusters(ctx context.Context, clusterIds []string) (int, []*errors.ServiceError)
	GetClusterIds(query string, args ...interface{}) ([]string, error)
	GetClusterOrg(id string) (string, *errors.ServiceError)
	ResetServiceAccount(ctx context.Context, cluster *dbapi.ConnectorCluster) *errors.ServiceError
}

var _ ConnectorClusterService = &connectorClusterService{}
var _ auth.AuthAgentService = &connectorClusterService{}

type connectorClusterService struct {
	connectionFactory         *db.ConnectionFactory
	bus                       signalbus.SignalBus
	connectorTypesService     ConnectorTypesService
	vaultService              vault.VaultService
	keycloakService           sso.KafkaKeycloakService
	connectorsService         ConnectorsService
	connectorNamespaceService ConnectorNamespaceService
}

func NewConnectorClusterService(connectionFactory *db.ConnectionFactory, bus signalbus.SignalBus, vaultService vault.VaultService,
	connectorTypesService ConnectorTypesService, connectorsService ConnectorsService,
	keycloakService sso.KafkaKeycloakService, connectorNamespaceService ConnectorNamespaceService) *connectorClusterService {
	return &connectorClusterService{
		connectionFactory:         connectionFactory,
		bus:                       bus,
		connectorTypesService:     connectorTypesService,
		vaultService:              vaultService,
		connectorsService:         connectorsService,
		keycloakService:           keycloakService,
		connectorNamespaceService: connectorNamespaceService,
	}
}

func (k *connectorClusterService) CleanupDeployments() *errors.ServiceError {
	type Result struct {
		DeploymentID string
	}

	// Find deployments that have not been deleted who's connector has been deleted...
	results := []Result{}
	dbConn := k.connectionFactory.New()
	err := dbConn.Table("connectors").
		Select("connector_deployments.id AS deployment_id").
		Joins("JOIN connector_deployments ON connectors.id = connector_deployments.connector_id").
		Where("connectors.deleted_at IS NOT NULL").
		Where("connector_deployments.deleted_at IS NULL").
		Scan(&results).Error

	if err != nil {
		return errors.GeneralError("unable to list connector deployment who's connectors have been deleted: %s", err)
	}

	for _, result := range results {
		// delete those deployments and associated status
		if err := dbConn.Delete(&dbapi.ConnectorDeploymentStatus{
			Model: db.Model{
				ID: result.DeploymentID,
			},
		}).Error; err != nil {
			return errors.GeneralError("unable to delete connector deployment status who's connectors have been deleted: %s", err)
		}
		if err := dbConn.Delete(&dbapi.ConnectorDeployment{
			Model: db.Model{
				ID: result.DeploymentID,
			},
		}).Error; err != nil {
			return errors.GeneralError("unable to delete connector deployment who's connectors have been deleted: %s", err)
		}
	}

	return nil
}

// Create creates a connector cluster in the database
func (k *connectorClusterService) Create(ctx context.Context, resource *dbapi.ConnectorCluster) *errors.ServiceError {
	dbConn := k.connectionFactory.New()
	if err := dbConn.Transaction(func(tx *gorm.DB) error {
		// NOTE: directly calling Create on cluster fails due to some issue in Gorm,
		// hence this TX to save cluster and annotations in separate Gorm calls
		if err := tx.Omit("Annotations").Create(resource).Error; err != nil {
			return err
		}
		// save annotations
		if err := tx.Create(resource.Annotations).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return services.HandleCreateError("Connector cluster", err)
	}

	// create a default namespace for the new cluster
	if err := k.connectorNamespaceService.CreateDefaultNamespace(ctx, resource); err != nil {
		return err
	}

	return nil
}

func filterClusterToOwnerOrOrg(ctx context.Context, dbConn *gorm.DB) (*gorm.DB, *errors.ServiceError) {

	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return dbConn, errors.Unauthenticated("user not authenticated")
	}
	owner, _ := claims.GetUsername()
	if owner == "" {
		return dbConn, errors.Unauthenticated("user not authenticated")
	}

	orgId, _ := claims.GetOrgId()
	filterByOrganisationId := auth.GetFilterByOrganisationFromContext(ctx)

	// filter by organisationId if a user is part of an organisation and is not allowed as a service account
	if filterByOrganisationId {
		// include namespaces where either user or organisation is the tenant
		dbConn = dbConn.Where("organisation_id = ?", orgId)
	} else {
		dbConn = dbConn.Where("owner = ?", owner)
	}
	return dbConn, nil
}

// Get gets a connector cluster by id from the database
func (k *connectorClusterService) Get(ctx context.Context, id string) (*dbapi.ConnectorCluster, *errors.ServiceError) {

	dbConn := k.connectionFactory.New()
	resource := &dbapi.ConnectorCluster{
		Model: db.Model{
			ID: id,
		},
	}

	if err := dbConn.Unscoped().Preload(clause.Associations).First(resource).Error; err != nil {
		return resource, services.HandleGetError("Connector cluster", "id", id, err)
	}
	if resource.DeletedAt.Valid {
		return resource, services.HandleGoneError("Connector cluster", "id", id)
	}
	return resource, nil
}

// Delete changes connector cluster status phase to `deleting`
func (k *connectorClusterService) Delete(ctx context.Context, id string) *errors.ServiceError {

	var clusterDeleted bool
	if err := k.connectionFactory.New().Transaction(func(dbConn *gorm.DB) error {

		var resource dbapi.ConnectorCluster
		if err := dbConn.Where("id = ?", id).Select("id", "status_phase").
			First(&resource).Error; err != nil {
			return services.HandleGetError("Connector cluster", "id", id, err)
		}

		if _, err := phase.PerformClusterOperation(&resource, phase.DeleteCluster,
			func(cluster *dbapi.ConnectorCluster) *errors.ServiceError {

				// mark cluster for deletion
				if err := dbConn.Model(cluster).Update("status_phase", cluster.Status.Phase).Error; err != nil {
					return services.HandleUpdateError("Connector cluster", err)
				}

				clusterDeleted = true
				return nil

			}); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return services.HandleDeleteError("Connector cluster", "id", id, err)
	}

	if clusterDeleted {
		// reconcile deleting clusters
		_ = db.AddPostCommitAction(ctx, func() {
			k.bus.Notify("reconcile:connector_cluster")
		})
	}

	return nil
}

func GetValidClusterColumns() []string {
	// property state will be replaced with column name status_phase
	return []string{"id", "created_at", "updated_at", "owner", "organisation_id", "name", "state", "client_id"}
}

// List returns all connector clusters visible to the user within the requested paging window.
func (k *connectorClusterService) List(ctx context.Context, listArgs *services.ListArguments) (dbapi.ConnectorClusterList, *api.PagingMeta, *errors.ServiceError) {
	if err := listArgs.Validate(GetValidClusterColumns()); err != nil {
		return nil, nil, errors.NewWithCause(errors.ErrorMalformedRequest, err, "unable to list connector cluster requests: %s", err.Error())
	}
	var resourceList dbapi.ConnectorClusterList
	dbConn := k.connectionFactory.New()
	pagingMeta := &api.PagingMeta{
		Page: listArgs.Page,
		Size: listArgs.Size,
	}

	var err *errors.ServiceError
	// allow admins to list clusters
	admin, err := isAdmin(ctx)
	if err != nil {
		return nil, nil, err
	}
	if !admin {
		dbConn, err = filterClusterToOwnerOrOrg(ctx, dbConn)
		if err != nil {
			return nil, nil, err
		}
	}

	// Apply search query
	if len(listArgs.Search) > 0 {
		queryParser := coreServices.NewQueryParser(GetValidClusterColumns()...)
		searchDbQuery, err := queryParser.Parse(listArgs.Search)
		if err != nil {
			return resourceList, pagingMeta, errors.NewWithCause(errors.ErrorFailedToParseSearch, err, "unable to list connector cluster requests: %s", err.Error())
		}
		dbConn = dbConn.Where(strings.ReplaceAll(searchDbQuery.Query, "state", "status_phase"), searchDbQuery.Values...)
	}

	// set total, limit and paging (based on https://gitlab.cee.redhat.com/service/api-guidelines#user-content-paging)
	total := int64(pagingMeta.Total)
	dbConn.Model(&resourceList).Count(&total)
	pagingMeta.Total = int(total)

	if pagingMeta.Size > pagingMeta.Total {
		pagingMeta.Size = pagingMeta.Total
	}
	dbConn = dbConn.Offset((pagingMeta.Page - 1) * pagingMeta.Size).Limit(pagingMeta.Size)

	// Set the order by arguments if any
	if len(listArgs.OrderBy) == 0 {
		// default orderBy name
		dbConn = dbConn.Order("name ASC")
	} else {
		for _, orderByArg := range listArgs.OrderBy {
			dbConn = dbConn.Order(strings.ReplaceAll(orderByArg, "state", "status_phase"))
		}
	}

	// execute query
	if err := dbConn.Preload(clause.Associations).Find(&resourceList).Error; err != nil {
		return resourceList, pagingMeta, services.HandleGetError(`Connector cluster`, `query`, listArgs.Search, err)
	}

	return resourceList, pagingMeta, nil
}

func isAdmin(ctx context.Context) (bool, *errors.ServiceError) {
	_, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return false, errors.NewWithCause(errors.ErrorUnauthenticated, err, "user not authenticated")
	}

	return auth.GetIsAdminFromContext(ctx), nil
}

func (k *connectorClusterService) Update(ctx context.Context, resource *dbapi.ConnectorCluster) *errors.ServiceError {
	if err := k.connectionFactory.New().Transaction(func(tx *gorm.DB) error {
		// remove old associations first
		if err := tx.Where("connector_cluster_id = ?", resource.ID).
			Delete(&dbapi.ConnectorClusterAnnotation{}).Error; err != nil {
			return errors.GeneralError("failed to remove connector cluster annotations %q: %v", resource.ID, err)
		}
		// create new associations
		if err := tx.Create(&resource.Annotations).Error; err != nil {
			return errors.GeneralError("failed to update connector cluster annotations %q: %v", resource.ID, err)
		}
		if err := tx.Model(resource).Omit("Annotations").
			Where("id = ?", resource.ID).Updates(resource).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return services.HandleUpdateError("Connector cluster", err)
	}
	return nil
}

func (k *connectorClusterService) UpdateConnectorClusterStatus(ctx context.Context, id string, status dbapi.ConnectorClusterStatus) *errors.ServiceError {
	dbConn := k.connectionFactory.New()
	var resource dbapi.ConnectorCluster

	if err := dbConn.Where("id = ?", id).First(&resource).Error; err != nil {
		return services.HandleGetError("Connector cluster", "id", id, err)
	}

	// compare current and requested cluster phases to validate that agent can connect
	updated, err := phase.PerformClusterOperation(&resource, phase.ConnectCluster)
	if err != nil {
		return err
	}

	// agent doesn't directly modify cluster phase, that's done in PerformClusterOperation()
	status.Phase = resource.Status.Phase

	if updated || !reflect.DeepEqual(resource.Status, status) {

		if updated {
			// The phase should not be changing that often.. but when it does,
			// kick off a reconcile to get connectors deployed to the cluster.
			_ = db.AddPostCommitAction(ctx, func() {
				// Wake up the reconcile loop...
				k.bus.Notify("reconcile:connector")
			})
		}

		if err := dbConn.Updates(&dbapi.ConnectorCluster{
			Model: db.Model{ID: id},
			Status: dbapi.ConnectorClusterStatus{
				Phase:      resource.Status.Phase,
				Version:    status.Version,
				Conditions: status.Conditions,
				Operators:  status.Operators,
				Platform:   status.Platform,
			}}).Error; err != nil {
			return errors.NewWithCause(errors.ErrorGeneral, err, "failed to update status")
		}
	}

	return nil
}

// Get gets a connector by id from the database
func (k *connectorClusterService) GetConnectorClusterStatus(ctx context.Context, id string) (dbapi.ConnectorClusterStatus, *errors.ServiceError) {

	dbConn := k.connectionFactory.New()
	var resource dbapi.ConnectorCluster
	dbConn = dbConn.Select("status_phase, status_version, status_conditions, status_operators").Where("id = ?", id)

	if err := dbConn.First(&resource).Error; err != nil {
		return resource.Status, services.HandleGetError("Connector cluster status", "id", id, err)
	}
	return resource.Status, nil
}

func (k *connectorClusterService) GetClientID(clusterID string) (string, error) {
	dbConn := k.connectionFactory.New()
	var resource dbapi.ConnectorCluster
	dbConn = dbConn.Unscoped().Select("client_id").Where("id = ?", clusterID)

	// use Limit(1) and Find() to avoid ErrRecordNotFound
	if err := dbConn.Limit(1).Find(&resource).Error; err != nil {
		return "", services.HandleGetError("Connector cluster client_id", "id", clusterID, err)
	}
	return resource.ClientId, nil
}

// SaveDeployment creates a connector deployment in the database
func (k *connectorClusterService) SaveDeployment(ctx context.Context, resource *dbapi.ConnectorDeployment) *errors.ServiceError {
	dbConn := k.connectionFactory.New()

	if err := dbConn.Save(resource).Error; err != nil {
		return services.HandleCreateError(`Connector deployment`, err)
	}

	if err := dbConn.Where("id = ?", resource.ID).Select("version").First(&resource).Error; err != nil {
		return services.HandleGetError(`Connector deployment`, "id", resource.ID, err)
	}

	if resource.ClusterID != "" {
		_ = db.AddPostCommitAction(ctx, func() {
			k.bus.Notify(fmt.Sprintf("/kafka_connector_clusters/%s/deployments", resource.ClusterID))
		})
	}

	return nil
}

func (k *connectorClusterService) UpdateDeployment(resource *dbapi.ConnectorDeployment) *errors.ServiceError {
	dbConn := k.connectionFactory.New()
	updates := dbConn.Where("id = ?", resource.ID).
		Updates(resource)
	if err := updates.Error; err != nil {
		return services.HandleUpdateError(`Connector namespace`, err)
	}
	return nil
}

func GetValidDeploymentColumns() []string {
	return []string{"connector_id", "connector_version", "cluster_id", "operator_id", "namespace_id"}
}

// ListConnectorDeployments returns all deployments assigned to the cluster
func (k *connectorClusterService) ListConnectorDeployments(ctx context.Context, clusterId string, filterChannelUpdates bool, filterOperatorUpdates bool, includeDanglingDeploymentsOnly bool, listArgs *services.ListArguments, gtVersion int64) (dbapi.ConnectorDeploymentList, *api.PagingMeta, *errors.ServiceError) {
	var resourceList dbapi.ConnectorDeploymentList
	dbConn := k.connectionFactory.New()
	// specify preload for annotations only, to avoid skipping deleted connectors
	dbConn = dbConn.Unscoped().Preload("Annotations").Joins("Status").Joins("ConnectorShardMetadata").Joins("Connector")

	pagingMeta := &api.PagingMeta{
		Page: listArgs.Page,
		Size: listArgs.Size,
	}

	if clusterId != "" {
		dbConn = dbConn.Where("connector_deployments.cluster_id = ?", clusterId)
	}
	if gtVersion != 0 {
		dbConn = dbConn.Where("connector_deployments.version > ?", gtVersion)
	}
	if filterChannelUpdates {
		dbConn = dbConn.Where("\"ConnectorShardMetadata\".\"latest_revision\" IS NOT NULL")
	}
	if filterOperatorUpdates {
		dbConn = dbConn.Where("\"Status\".\"upgrade_available\"")
	}
	if includeDanglingDeploymentsOnly {
		dbConn = dbConn.Where("\"Connector\".\"deleted_at\" IS NOT NULL")
	} else {
		dbConn = dbConn.Where("\"Connector\".\"deleted_at\" IS NULL")
	}

	// Apply search query
	if len(listArgs.Search) > 0 {
		queryParser := coreServices.NewQueryParserWithColumnPrefix("connector_deployments", GetValidDeploymentColumns()...)
		searchDbQuery, err := queryParser.Parse(listArgs.Search)
		if err != nil {
			return resourceList, pagingMeta, errors.NewWithCause(errors.ErrorFailedToParseSearch, err, "unable to list connector deployments requests: %s", err.Error())
		}
		dbConn = dbConn.Where(searchDbQuery.Query, searchDbQuery.Values...)
	}

	// set total, limit and paging (based on https://gitlab.cee.redhat.com/service/api-guidelines#user-content-paging)
	total := int64(pagingMeta.Total)
	dbConn.Session(&gorm.Session{}).Model(&resourceList).Count(&total)
	pagingMeta.Total = int(total)

	if pagingMeta.Size > pagingMeta.Total {
		pagingMeta.Size = pagingMeta.Total
	}
	dbConn = dbConn.Offset((pagingMeta.Page - 1) * pagingMeta.Size).Limit(pagingMeta.Size)

	// default the order by version
	dbConn = dbConn.Order("connector_deployments.version")

	// execute query
	if err := dbConn.Find(&resourceList).Error; err != nil {
		return resourceList, pagingMeta, services.HandleGetError("Connector deployment",
			fmt.Sprintf("filterChannelUpdates='%v' includeDanglingDeploymentsOnly=%v listArgs='%+v' cluster_id",
				filterChannelUpdates, includeDanglingDeploymentsOnly, listArgs), clusterId, err)
	}

	return resourceList, pagingMeta, nil
}

func (k *connectorClusterService) UpdateConnectorDeploymentStatus(ctx context.Context, deploymentStatus dbapi.ConnectorDeploymentStatus) *errors.ServiceError {
	dbConn := k.connectionFactory.New()

	// lets get the connector id of the deployment..
	deployment := dbapi.ConnectorDeployment{}
	if err := dbConn.Unscoped().Select("connector_id", "deleted_at").
		Where("id = ?", deploymentStatus.ID).
		First(&deployment).Error; err != nil {
		return services.HandleGetError("Connector deployment", "id", deploymentStatus.ID, err)
	}
	if deployment.DeletedAt.Valid {
		return services.HandleGoneError("Connector deployment", "id", deploymentStatus.ID)
	}

	// See https://github.com/go-gorm/gorm/issues/6139
	// We can no longer rely on the Save throwing a constraint violation exception when upserting a record that already exists.
	// By first checking that the record existences we ensure the Save only updates an existing record vs trying to create a duplicate.
	existing := dbapi.ConnectorDeploymentStatus{}
	dbConn.Unscoped().Where("id = ?", deploymentStatus.ID).First(&existing)
	// If it was not found existing.Version will be zero.
	if existing.Version > deploymentStatus.Version {
		return errors.Conflict("failed to update deployment status, probably a stale deployment status version was used: %d", deploymentStatus.Version)
	}
	if err := dbConn.Unscoped().Model(&deploymentStatus).Where("id = ? and version <= ?", deploymentStatus.ID, deploymentStatus.Version).Save(&deploymentStatus).Error; err != nil {
		return errors.Conflict("failed to update deployment status: %s, probably a stale deployment status version was used: %d", err.Error(), deploymentStatus.Version)
	}

	connector := dbapi.Connector{}
	if err := dbConn.Select("desired_state").
		Where("id = ?", deployment.ConnectorID).
		First(&connector).Error; err != nil {
		return services.HandleGetError("Connector", "id", deployment.ConnectorID, err)
	}

	connectorStatus := dbapi.ConnectorStatus{}
	if err := dbConn.Select("phase").
		Where("id = ?", deployment.ConnectorID).
		First(&connectorStatus).Error; err != nil {
		return services.HandleGetError("Connector", "id", deployment.ConnectorID, err)
	}

	connectorStatus.Phase = deploymentStatus.Phase
	if deploymentStatus.Phase == dbapi.ConnectorStatusPhaseDeleted {
		// we don't need the deployment anymore...
		if err := deleteConnectorDeployment(dbConn, deploymentStatus.ID); err != nil {
			return err
		}
	}

	// update the connector status
	if err := dbConn.Where("id = ?", deployment.ConnectorID).Updates(&connectorStatus).Error; err != nil {
		return services.HandleUpdateError("Connector status", err)
	}

	return nil
}

func (k *connectorClusterService) FindAvailableNamespace(owner string, orgID string, namespaceID *string) (*dbapi.ConnectorNamespace, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()
	var namespaces dbapi.ConnectorNamespaceList

	if orgID != "" {
		dbConn = dbConn.Where("id = ? AND (tenant_organisation_id = ? OR tenant_user_id = ?) AND status_phase = ?",
			namespaceID, orgID, owner, dbapi.ConnectorNamespacePhaseReady)
	} else {
		dbConn = dbConn.Where("id = ? AND tenant_owner_id = ? AND status_phase = ?",
			namespaceID, owner, dbapi.ConnectorNamespacePhaseReady)
	}

	if err := dbConn.Limit(1).Find(&namespaces).Error; err != nil {
		return nil, services.HandleGetError(`Connector namespace`, `id`, *namespaceID, err)
	}

	if len(namespaces) > 0 {
		return namespaces[0], nil
	}

	return nil, nil
}

func (k *connectorClusterService) GetDeploymentByConnectorId(ctx context.Context, connectorID string) (resource dbapi.ConnectorDeployment, serr *errors.ServiceError) {

	if err := k.connectionFactory.New().Preload(clause.Associations).
		Joins("Status").Joins("ConnectorShardMetadata").Joins("Connector").
		Where("connector_id = ?", connectorID).First(&resource).Error; err != nil {
		return resource, services.HandleGetError("Connector deployment", "connector_id", connectorID, err)
	}
	return
}

func (k *connectorClusterService) GetDeployment(ctx context.Context, id string) (resource dbapi.ConnectorDeployment, serr *errors.ServiceError) {

	dbConn := k.connectionFactory.New()
	if err := dbConn.Unscoped().Preload(clause.Associations).
		Joins("Status").Joins("ConnectorShardMetadata").Joins("Connector").
		Where("connector_deployments.id = ?", id).First(&resource).Error; err != nil {
		return resource, services.HandleGetError("Connector deployment", "id", id, err)
	}

	if resource.DeletedAt.Valid {
		return resource, services.HandleGoneError("Connector deployment", "id", id)
	}

	return
}

// ReconcileDeletingClusters deletes empty clusters with no namespaces that are in phase deleting,
// it also deletes their service accounts to disconnect from the agent gracefully
func (k *connectorClusterService) ReconcileEmptyDeletingClusters(_ context.Context, clusterIds []string) (int, []*errors.ServiceError) {
	count := len(clusterIds)
	var errs []*errors.ServiceError
	if err := k.connectionFactory.New().Transaction(func(dbConn *gorm.DB) error {

		// get client_ids for service accounts
		var clientIds []string
		if err := dbConn.Where("id IN ?", clusterIds).
			Model(&dbapi.ConnectorCluster{}).Select("client_id").
			Find(&clientIds).Error; err != nil {
			return services.HandleDeleteError("Connector cluster", "id", clusterIds, err)
		}

		// remove service account for clusters first, so agents are blocked from connecting
		for _, id := range clientIds {
			glog.V(5).Infof("Removing agent service account for connector cluster %s", id)
			if err := k.keycloakService.DeleteServiceAccountInternal(id); err != nil {
				errs = append(errs, errors.GeneralError(
					"failed to remove connector service account %s: %s", id, err))
			}
		}

		// delete all the empty deleting clusters
		if err := dbConn.Where("id IN ?", clusterIds).
			Delete(&dbapi.ConnectorCluster{}).Error; err != nil {
			return services.HandleDeleteError("Connector cluster", "id", clusterIds, err)
		}

		return nil

	}); err != nil {
		errs = append(errs, services.HandleDeleteError("Connector cluster", "status_phase",
			dbapi.ConnectorClusterPhaseDeleting, err))
		count = 0
	}

	return count, errs
}

// ReconcileNonEmptyDeletingClusters deletes non-deleting namespaces in deleting clusters
func (k *connectorClusterService) ReconcileNonEmptyDeletingClusters(ctx context.Context, clusterIds []string) (int, []*errors.ServiceError) {
	count := len(clusterIds)
	var errs []*errors.ServiceError
	if err := k.connectionFactory.New().Transaction(func(dbConn *gorm.DB) error {

		// cascade delete non-deleting namespaces
		var serr *errors.ServiceError
		_, serr = k.connectorNamespaceService.DeleteNamespaces(ctx, dbConn,
			"connector_namespaces.cluster_id IN ? AND connector_namespaces.status_phase NOT IN ?",
			clusterIds, []string{string(dbapi.ConnectorNamespacePhaseDeleting), string(dbapi.ConnectorNamespacePhaseDeleted)})
		if serr != nil {
			return serr
		}

		return nil

	}); err != nil {
		errs = append(errs, services.HandleDeleteError("Connector cluster", "status_phase",
			dbapi.ConnectorClusterPhaseDeleting, err))
		count = 0
	}

	return count, errs
}

// GetClusterIds gets ids of all clusters that match the query and args
func (k *connectorClusterService) GetClusterIds(query string, args ...interface{}) ([]string, error) {
	var clusterIds []string
	if err := k.connectionFactory.New().Table("connector_clusters").Select("connector_clusters.id").
		Joins("LEFT JOIN connector_namespaces ON connector_namespaces.cluster_id = connector_clusters.id AND "+
			"connector_namespaces.deleted_at IS NULL").
		Group("connector_clusters.id").
		Where(query, args...).
		Find(&clusterIds).Error; err != nil {
		return nil, services.HandleGetError("Connector cluster", query, args, err)
	}
	return clusterIds, nil
}

func (k *connectorClusterService) GetClusterOrg(id string) (string, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()
	cluster := dbapi.ConnectorCluster{}
	if err := dbConn.Unscoped().Select("organisation_id").Where("id = ?", id).Find(&cluster).Error; err != nil {
		return "", services.HandleGetError("Connector cluster", "id", id, err)
	}
	if cluster.DeletedAt.Valid {
		return "", services.HandleGoneError("Connector cluster", "id", id)
	}
	return cluster.OrganisationId, nil
}

func (k *connectorClusterService) ResetServiceAccount(ctx context.Context, cluster *dbapi.ConnectorCluster) *errors.ServiceError {
	secret, serr := k.keycloakService.GetKafkaClientSecret(cluster.ClientId)
	if serr != nil {
		return serr
	}
	cluster.ClientSecret = secret

	if err := k.connectionFactory.New().UpdateColumns(dbapi.ConnectorCluster{
		Model:        db.Model{ID: cluster.ID},
		ClientId:     cluster.ClientId,
		ClientSecret: cluster.ClientSecret,
	}).Error; err != nil {
		return services.HandleGetError(`Connector cluster`, `id`, cluster.ID, err)
	}
	return nil
}
