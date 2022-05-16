package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/phase"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/queryparser"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sso"
	"github.com/golang/glog"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/vault"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/secrets"
	"github.com/spyzhov/ajson"
	"gorm.io/gorm"
)

type ConnectorClusterService interface {
	Create(ctx context.Context, resource *dbapi.ConnectorCluster) *errors.ServiceError
	Get(ctx context.Context, id string) (dbapi.ConnectorCluster, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	List(ctx context.Context, listArgs *services.ListArguments) (dbapi.ConnectorClusterList, *api.PagingMeta, *errors.ServiceError)
	Update(ctx context.Context, resource *dbapi.ConnectorCluster) *errors.ServiceError
	UpdateConnectorClusterStatus(ctx context.Context, id string, status dbapi.ConnectorClusterStatus) *errors.ServiceError
	GetConnectorClusterStatus(ctx context.Context, id string) (dbapi.ConnectorClusterStatus, *errors.ServiceError)

	SaveDeployment(ctx context.Context, resource *dbapi.ConnectorDeployment) *errors.ServiceError
	GetConnectorWithBase64Secrets(ctx context.Context, resource dbapi.ConnectorDeployment) (dbapi.Connector, bool, *errors.ServiceError)
	ListConnectorDeployments(ctx context.Context, id string, listArgs *services.ListArguments, gtVersion int64) (dbapi.ConnectorDeploymentList, *api.PagingMeta, *errors.ServiceError)
	UpdateConnectorDeploymentStatus(ctx context.Context, status dbapi.ConnectorDeploymentStatus) *errors.ServiceError
	FindAvailableNamespace(owner string, orgId string, namespaceId *string) (*dbapi.ConnectorNamespace, *errors.ServiceError)
	GetDeploymentByConnectorId(ctx context.Context, connectorID string) (dbapi.ConnectorDeployment, *errors.ServiceError)
	GetDeployment(ctx context.Context, id string) (dbapi.ConnectorDeployment, *errors.ServiceError)
	GetAvailableDeploymentTypeUpgrades(listArgs *services.ListArguments) (dbapi.ConnectorDeploymentTypeUpgradeList, *api.PagingMeta, *errors.ServiceError)
	UpgradeConnectorsByType(ctx context.Context, clusterId string, upgrades dbapi.ConnectorDeploymentTypeUpgradeList) *errors.ServiceError
	GetAvailableDeploymentOperatorUpgrades(listArgs *services.ListArguments) (dbapi.ConnectorDeploymentOperatorUpgradeList, *api.PagingMeta, *errors.ServiceError)
	UpgradeConnectorsByOperator(ctx context.Context, clusterId string, upgrades dbapi.ConnectorDeploymentOperatorUpgradeList) *errors.ServiceError
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
		return errors.GeneralError("Unable to list connector deployment who's connectors have been deleted: %s", err)
	}

	for _, result := range results {
		// delete those deployments and associated status
		if err := dbConn.Delete(&dbapi.ConnectorDeploymentStatus{
			Model: db.Model{
				ID: result.DeploymentID,
			},
		}).Error; err != nil {
			return errors.GeneralError("Unable to delete connector deployment status who's connectors have been deleted: %s", err)
		}
		if err := dbConn.Delete(&dbapi.ConnectorDeployment{
			Model: db.Model{
				ID: result.DeploymentID,
			},
		}).Error; err != nil {
			return errors.GeneralError("Unable to delete connector deployment who's connectors have been deleted: %s", err)
		}
	}

	return nil
}

// Create creates a connector cluster in the database
func (k *connectorClusterService) Create(ctx context.Context, resource *dbapi.ConnectorCluster) *errors.ServiceError {
	dbConn := k.connectionFactory.New()
	if err := dbConn.Save(resource).Error; err != nil {
		return services.HandleCreateError("Connector", err)
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
func (k *connectorClusterService) Get(ctx context.Context, id string) (dbapi.ConnectorCluster, *errors.ServiceError) {

	dbConn := k.connectionFactory.New()
	var resource dbapi.ConnectorCluster
	dbConn = dbConn.Where("id = ?", id)

	if err := dbConn.Unscoped().First(&resource).Error; err != nil {
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

// List returns all connector clusters visible to the user within the requested paging window.
func (k *connectorClusterService) List(ctx context.Context, listArgs *services.ListArguments) (dbapi.ConnectorClusterList, *api.PagingMeta, *errors.ServiceError) {
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

	// set total, limit and paging (based on https://gitlab.cee.redhat.com/service/api-guidelines#user-content-paging)
	total := int64(pagingMeta.Total)
	dbConn.Model(&resourceList).Count(&total)
	pagingMeta.Total = int(total)

	if pagingMeta.Size > pagingMeta.Total {
		pagingMeta.Size = pagingMeta.Total
	}
	dbConn = dbConn.Offset((pagingMeta.Page - 1) * pagingMeta.Size).Limit(pagingMeta.Size)

	// default the order by name
	dbConn = dbConn.Order("name")

	// execute query
	if err := dbConn.Find(&resourceList).Error; err != nil {
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

func (k connectorClusterService) Update(ctx context.Context, resource *dbapi.ConnectorCluster) *errors.ServiceError {
	dbConn := k.connectionFactory.New()
	if err := dbConn.Where("id = ?", resource.ID).Model(resource).Updates(resource).Error; err != nil {
		return services.HandleUpdateError("Connector", err)
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

func (k *connectorClusterService) GetClientId(clusterID string) (string, error) {
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

func GetValidDeploymentColumns() []string {
	return []string{"connector_id", "connector_version", "connector_type_channel_id", "cluster_id", "operator_id", "namespace_id"}
}

// ListConnectorDeployments returns all deployments assigned to the cluster
func (k *connectorClusterService) ListConnectorDeployments(ctx context.Context, id string, listArgs *services.ListArguments, gtVersion int64) (dbapi.ConnectorDeploymentList, *api.PagingMeta, *errors.ServiceError) {
	var resourceList dbapi.ConnectorDeploymentList
	dbConn := k.connectionFactory.New()
	dbConn = dbConn.Preload("Status")
	pagingMeta := &api.PagingMeta{
		Page: listArgs.Page,
		Size: listArgs.Size,
	}

	if id != "" {
		dbConn = dbConn.Where("connector_deployments.cluster_id = ?", id)
	}
	if gtVersion != 0 {
		dbConn = dbConn.Where("connector_deployments.version > ?", gtVersion)
	}

	// Apply search query
	if len(listArgs.Search) > 0 {
		queryParser := coreServices.NewQueryParser(GetValidDeploymentColumns()...)
		searchDbQuery, err := queryParser.Parse(listArgs.Search)
		if err != nil {
			return resourceList, pagingMeta, errors.NewWithCause(errors.ErrorFailedToParseSearch, err, "Unable to list connector deployments requests: %s", err.Error())
		}
		dbConn = dbConn.Where(searchDbQuery.Query, searchDbQuery.Values...)
	}

	// set total, limit and paging (based on https://gitlab.cee.redhat.com/service/api-guidelines#user-content-paging)
	total := int64(pagingMeta.Total)
	dbConn.Model(&resourceList).Count(&total)
	pagingMeta.Total = int(total)

	if pagingMeta.Size > pagingMeta.Total {
		pagingMeta.Size = pagingMeta.Total
	}
	dbConn = dbConn.Offset((pagingMeta.Page - 1) * pagingMeta.Size).Limit(pagingMeta.Size)

	// default the order by version
	dbConn = dbConn.Order("connector_deployments.version")

	// execute query
	if err := dbConn.Find(&resourceList).Error; err != nil {
		return resourceList, pagingMeta, services.HandleGetError("Connector deployment", `cluster_id`, id, err)
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

	if err := dbConn.Model(&deploymentStatus).Where("id = ? and version <= ?", deploymentStatus.ID, deploymentStatus.Version).Save(&deploymentStatus).Error; err != nil {
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

func (k *connectorClusterService) GetConnectorWithBase64Secrets(ctx context.Context, resource dbapi.ConnectorDeployment) (dbapi.Connector, bool, *errors.ServiceError) {

	dbConn := k.connectionFactory.New()

	var connector dbapi.Connector
	err := dbConn.Where("id = ?", resource.ConnectorID).First(&connector).Error
	if err != nil {
		return connector, false, services.HandleGetError("Connector", "id", resource.ConnectorID, err)
	}

	serr := getSecretsFromVaultAsBase64(&connector, k.connectorTypesService, k.vaultService)
	if serr != nil {
		return connector, serr.Code == errors.ErrorGeneral, serr
	}

	return connector, false, nil
}

func getSecretsFromVaultAsBase64(resource *dbapi.Connector, cts ConnectorTypesService, vault vault.VaultService) *errors.ServiceError {
	ct, err := cts.Get(resource.ConnectorTypeId)
	if err != nil {
		return errors.BadRequest("invalid connector type id: %s", resource.ConnectorTypeId)
	}
	// move secrets to a vault.

	if resource.ServiceAccount.ClientSecretRef != "" {
		v, err := vault.GetSecretString(resource.ServiceAccount.ClientSecretRef)
		if err != nil {
			return errors.GeneralError("could not get kafka client secrets from the vault: %v", err.Error())
		}
		encoded := base64.StdEncoding.EncodeToString([]byte(v))
		resource.ServiceAccount.ClientSecret = encoded
	}

	if len(resource.ConnectorSpec) != 0 {
		updated, err := secrets.ModifySecrets(ct.JsonSchema, resource.ConnectorSpec, func(node *ajson.Node) error {
			if node.Type() == ajson.Object {
				ref, err := node.GetKey("ref")
				if err != nil {
					return err
				}
				r, err := ref.GetString()
				if err != nil {
					return err
				}
				v, err := vault.GetSecretString(r)
				if err != nil {
					return err
				}

				encoded := base64.StdEncoding.EncodeToString([]byte(v))
				err = node.SetObject(map[string]*ajson.Node{
					"kind":  ajson.StringNode("", "base64"),
					"value": ajson.StringNode("", encoded),
				})
				if err != nil {
					return err
				}
			} else if node.Type() == ajson.Null {
				// don't change..
			} else {
				return fmt.Errorf("secret field must be set to an object: " + node.Path())
			}
			return nil
		})
		if err != nil {
			return errors.GeneralError("could not get connectors secrets from the vault: %v", err.Error())
		}
		resource.ConnectorSpec = updated
	}
	return nil
}

func (k *connectorClusterService) GetDeploymentByConnectorId(ctx context.Context, connectorID string) (resource dbapi.ConnectorDeployment, serr *errors.ServiceError) {

	dbConn := k.connectionFactory.New().Where("connector_id = ?", connectorID)
	if err := dbConn.First(&resource).Error; err != nil {
		return resource, services.HandleGetError("Connector deployment", "connector_id", connectorID, err)
	}
	return
}

func (k *connectorClusterService) GetDeployment(ctx context.Context, id string) (resource dbapi.ConnectorDeployment, serr *errors.ServiceError) {

	dbConn := k.connectionFactory.New()
	if err := dbConn.Unscoped().Where("connector_deployments.id = ?", id).First(&resource).Error; err != nil {
		return resource, services.HandleGetError("Connector deployment", "id", id, err)
	}

	if resource.DeletedAt.Valid {
		return resource, services.HandleGoneError("Connector deployment", "id", id)
	}

	return
}

func (k *connectorClusterService) GetAvailableDeploymentTypeUpgrades(listArgs *services.ListArguments) (upgrades dbapi.ConnectorDeploymentTypeUpgradeList, paging *api.PagingMeta, serr *errors.ServiceError) {

	type Result struct {
		ConnectorID              string
		DeploymentID             string
		ConnectorTypeUpgradeFrom int64
		ConnectorTypeUpgradeTo   int64
		ConnectorTypeID          string
		NamespaceID              string
		Channel                  string
	}

	results := []Result{}
	dbConn := k.connectionFactory.New()
	dbConn = dbConn.Table("connector_deployments")
	dbConn = dbConn.Select(
		"connector_deployments.connector_id AS connector_id",
		"connector_deployments.id AS deployment_id",
		"connector_deployments.namespace_id AS namespace_id",
		"connector_shard_metadata.id AS connector_type_upgrade_from",
		"connector_shard_metadata.latest_id AS connector_type_upgrade_to",
		"connector_shard_metadata.connector_type_id",
		"connector_shard_metadata.channel",
	)
	dbConn = dbConn.Joins("LEFT JOIN connector_shard_metadata ON connector_shard_metadata.id = connector_deployments.connector_type_channel_id")
	dbConn = dbConn.Joins("LEFT JOIN connector_deployment_statuses ON connector_deployment_statuses.id = connector_deployments.id")
	dbConn = dbConn.Where("connector_shard_metadata.latest_id IS NOT NULL")
	dbConn = dbConn.Or("connector_deployment_statuses.upgrade_available")

	if err := dbConn.Scan(&results).Error; err != nil {
		return upgrades, paging, services.HandleGetError(`Connector deployment status`, `latest_id is not null or upgrade_available`, true, err)
	}

	// TODO support paging
	paging = &api.PagingMeta{
		Page:  0,
		Size:  len(results),
		Total: len(results),
	}

	upgrades = make(dbapi.ConnectorDeploymentTypeUpgradeList, len(results))
	for i, r := range results {
		upgrades[i] = dbapi.ConnectorDeploymentTypeUpgrade{
			ConnectorID:     r.ConnectorID,
			DeploymentID:    r.DeploymentID,
			ConnectorTypeId: r.ConnectorTypeID,
			NamespaceID:     r.NamespaceID,
			Channel:         r.Channel,
		}

		upgrades[i].ShardMetadata = &dbapi.ConnectorTypeUpgrade{
			AssignedId:  r.ConnectorTypeUpgradeFrom,
			AvailableId: r.ConnectorTypeUpgradeTo,
		}
	}

	return
}

func (k *connectorClusterService) UpgradeConnectorsByType(ctx context.Context, clusterId string, upgrades dbapi.ConnectorDeploymentTypeUpgradeList) *errors.ServiceError {

	// get deployment ids from available upgrades
	available, _, serr := k.GetAvailableDeploymentTypeUpgrades(&services.ListArguments{})
	if serr != nil {
		return serr
	}

	availableConnectors := toTypeMap(available)
	reqConnectors := toTypeMap(upgrades)

	// validate reqConnectors
	var errorList errors.ErrorList
	for cid, upgrade := range reqConnectors {
		availableUpgrade, ok := availableConnectors[cid]
		if !ok {
			errorList = append(errorList, errors.Conflict("Type upgrade not available for connector %v", cid))
		}
		// make sure other bits match
		upgrade.DeploymentID = availableUpgrade.DeploymentID
		if !reflect.DeepEqual(upgrade, availableUpgrade) {
			errorList = append(errorList, errors.Conflict("Type upgrade is outdated for connector %v", cid))
		}
	}
	if len(errorList) != 0 {
		return errors.Conflict(errorList.Error())
	}

	// upgrade connector type channels
	notificationAdded := false
	dbConn := k.connectionFactory.New()
	for cid, upgrade := range availableConnectors {

		// update connector channel id
		if err := dbConn.Model(&dbapi.ConnectorDeployment{}).
			Where("id = ?", upgrade.DeploymentID).
			Update("ConnectorTypeChannelId", upgrade.ShardMetadata.AvailableId).Error; err != nil {
			errorList = append(errorList,
				services.HandleUpdateError(`Connector deployment id=`+cid, err))
		} else {
			if !notificationAdded {
				_ = db.AddPostCommitAction(ctx, func() {
					k.bus.Notify(fmt.Sprintf("/kafka_connector_clusters/%s/deployments", clusterId))
				})
				notificationAdded = true
			}
		}
	}

	if len(errorList) != 0 {
		return services.HandleUpdateError(`Connector deployment`, errorList)
	}
	return nil
}

func toTypeMap(arr []dbapi.ConnectorDeploymentTypeUpgrade) map[string]dbapi.ConnectorDeploymentTypeUpgrade {
	m := make(map[string]dbapi.ConnectorDeploymentTypeUpgrade)
	for _, upgrade := range arr {
		m[upgrade.ConnectorID] = upgrade
	}
	return m
}

func (k *connectorClusterService) GetAvailableDeploymentOperatorUpgrades(listArgs *services.ListArguments) (upgrades dbapi.ConnectorDeploymentOperatorUpgradeList, paging *api.PagingMeta, serr *errors.ServiceError) {

	type Result struct {
		ConnectorID        string
		DeploymentID       string
		ConnectorTypeID    string
		NamespaceID        string
		Channel            string
		ConnectorOperators api.JSON
	}

	results := []Result{}
	dbConn := k.connectionFactory.New()
	dbConn = dbConn.Table("connector_deployments")
	dbConn = dbConn.Select(
		"connector_deployments.connector_id AS connector_id",
		"connector_deployments.id AS deployment_id",
		"connector_deployments.namespace_id AS namespace_id",
		"connector_shard_metadata.connector_type_id",
		"connector_shard_metadata.channel",
		"connector_deployment_statuses.operators AS connector_operators",
	)
	dbConn = dbConn.Joins("LEFT JOIN connector_shard_metadata ON connector_shard_metadata.id = connector_deployments.connector_type_channel_id")
	dbConn = dbConn.Joins("LEFT JOIN connector_deployment_statuses ON connector_deployment_statuses.id = connector_deployments.id")
	dbConn = dbConn.Where("connector_deployment_statuses.upgrade_available")

	if err := dbConn.Scan(&results).Error; err != nil {
		return upgrades, paging, services.HandleGetError(`Connector deployment`, `upgrade_available`, true, err)
	}

	// TODO support paging
	paging = &api.PagingMeta{
		Page:  0,
		Size:  len(results),
		Total: len(results),
	}
	upgrades = make([]dbapi.ConnectorDeploymentOperatorUpgrade, len(results))
	for i, r := range results {
		upgrades[i] = dbapi.ConnectorDeploymentOperatorUpgrade{
			ConnectorID:     r.ConnectorID,
			DeploymentID:    r.DeploymentID,
			ConnectorTypeId: r.ConnectorTypeID,
			NamespaceID:     r.NamespaceID,
			Channel:         r.Channel,
		}

		var operators private.ConnectorDeploymentStatusOperators
		err := json.Unmarshal([]byte(r.ConnectorOperators), &operators)
		if err != nil {
			return upgrades, paging, errors.GeneralError("converting ConnectorDeploymentStatusOperators: %s", err)
		}

		upgrades[i].Operator = &dbapi.ConnectorOperatorUpgrade{
			Assigned: dbapi.ConnectorOperator{
				Id:      operators.Assigned.Id,
				Type:    operators.Assigned.Type,
				Version: operators.Assigned.Version,
			},
			Available: dbapi.ConnectorOperator{
				Id:      operators.Available.Id,
				Type:    operators.Available.Type,
				Version: operators.Available.Version,
			},
		}

	}

	return
}

func (k *connectorClusterService) UpgradeConnectorsByOperator(ctx context.Context, clusterId string, upgrades dbapi.ConnectorDeploymentOperatorUpgradeList) *errors.ServiceError {
	// get deployment ids from available upgrades
	available, _, serr := k.GetAvailableDeploymentOperatorUpgrades(&services.ListArguments{})
	if serr != nil {
		return serr
	}

	availableConnectors := toOperatorMap(available)
	reqConnectors := toOperatorMap(upgrades)

	// validate reqConnectors
	var errorList errors.ErrorList
	for cid, upgrade := range reqConnectors {
		availableUpgrade, ok := availableConnectors[cid]
		if !ok {
			errorList = append(errorList, errors.Conflict("Operator upgrade not available for connector %s", cid))
		}

		// make sure other bits match
		upgrade.DeploymentID = availableUpgrade.DeploymentID
		upgrade.Operator.Assigned.Type = availableUpgrade.Operator.Assigned.Type
		upgrade.Operator.Assigned.Version = availableUpgrade.Operator.Assigned.Version
		upgrade.Operator.Available.Type = availableUpgrade.Operator.Available.Type
		upgrade.Operator.Available.Version = availableUpgrade.Operator.Available.Version

		if !reflect.DeepEqual(upgrade, availableUpgrade) {
			errorList = append(errorList, errors.Conflict("Operator upgrade is outdated for connector %s", cid))
		}
	}
	if len(errorList) != 0 {
		return errors.Conflict(errorList.Error())
	}

	// update deployments by setting operator_id to available_id
	notificationAdded := false
	dbConn := k.connectionFactory.New()
	for cid, upgrade := range availableConnectors {

		// upgrade operator id
		if err := dbConn.Model(&dbapi.ConnectorDeployment{}).
			Where("id = ?", upgrade.DeploymentID).
			Update("OperatorID", upgrade.Operator.Available.Id).Error; err != nil {
			errorList = append(errorList,
				services.HandleUpdateError("Connector deployment id="+cid, serr))
		} else {
			if !notificationAdded {
				_ = db.AddPostCommitAction(ctx, func() {
					k.bus.Notify(fmt.Sprintf("/kafka_connector_clusters/%s/deployments", clusterId))
				})
				notificationAdded = true
			}
		}
	}

	if len(errorList) != 0 {
		return services.HandleUpdateError(`Connector deployment`, errorList)
	}

	return nil
}

func toOperatorMap(arr []dbapi.ConnectorDeploymentOperatorUpgrade) map[string]dbapi.ConnectorDeploymentOperatorUpgrade {
	m := make(map[string]dbapi.ConnectorDeploymentOperatorUpgrade)
	for _, upgrade := range arr {
		m[upgrade.ConnectorID] = upgrade
	}
	return m
}

// ReconcileDeletingClusters deletes empty clusters with no namespaces that are in phase deleting,
// it also deletes their service accounts to disconnect from the agent gracefully
func (k *connectorClusterService) ReconcileEmptyDeletingClusters(_ context.Context, clusterIds []string) (int, []*errors.ServiceError) {
	count := len(clusterIds)
	var errs []*errors.ServiceError
	if err := k.connectionFactory.New().Transaction(func(dbConn *gorm.DB) error {

		// remove service account for clusters first, so agents are blocked from connecting
		for _, id := range clusterIds {
			glog.V(5).Infof("Removing agent service account for connector cluster %s", id)
			if err := k.keycloakService.DeRegisterConnectorFleetshardOperatorServiceAccount(id); err != nil {
				errs = append(errs, errors.GeneralError(
					"failed to remove connector service account for cluster %s: %s", id, err))
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
