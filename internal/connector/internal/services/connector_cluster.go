package services

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	goerrors "errors"
	"fmt"
	"github.com/golang/glog"
	"math/rand"
	"reflect"

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
	GetConnectorWithBase64Secrets(ctx context.Context, resource dbapi.ConnectorDeployment) (dbapi.Connector, *errors.ServiceError)
	ListConnectorDeployments(ctx context.Context, id string, listArgs *services.ListArguments, gtVersion int64) (dbapi.ConnectorDeploymentList, *api.PagingMeta, *errors.ServiceError)
	UpdateConnectorDeploymentStatus(ctx context.Context, status dbapi.ConnectorDeploymentStatus) *errors.ServiceError
	FindReadyNamespace(owner string, orgId string, namespaceId *string) (*dbapi.ConnectorNamespace, *errors.ServiceError)
	GetDeploymentByConnectorId(ctx context.Context, connectorID string) (dbapi.ConnectorDeployment, *errors.ServiceError)
	GetDeployment(ctx context.Context, id string) (dbapi.ConnectorDeployment, *errors.ServiceError)
	GetAvailableDeploymentTypeUpgrades(listArgs *services.ListArguments) (dbapi.ConnectorDeploymentTypeUpgradeList, *api.PagingMeta, *errors.ServiceError)
	UpgradeConnectorsByType(ctx context.Context, clusterId string, upgrades dbapi.ConnectorDeploymentTypeUpgradeList) *errors.ServiceError
	GetAvailableDeploymentOperatorUpgrades(listArgs *services.ListArguments) (dbapi.ConnectorDeploymentOperatorUpgradeList, *api.PagingMeta, *errors.ServiceError)
	UpgradeConnectorsByOperator(ctx context.Context, clusterId string, upgrades dbapi.ConnectorDeploymentOperatorUpgradeList) *errors.ServiceError
	CleanupDeployments() *errors.ServiceError
	UpdateClientId(clusterId string, clientID string) *errors.ServiceError
}

var _ ConnectorClusterService = &connectorClusterService{}
var _ auth.AuthAgentService = &connectorClusterService{}

type connectorClusterService struct {
	connectionFactory         *db.ConnectionFactory
	bus                       signalbus.SignalBus
	connectorTypesService     ConnectorTypesService
	vaultService              vault.VaultService
	keycloakService           services.KafkaKeycloakService
	connectorsService         ConnectorsService
	connectorNamespaceService ConnectorNamespaceService
	deploymentColumns         []string
}

func NewConnectorClusterService(connectionFactory *db.ConnectionFactory, bus signalbus.SignalBus, vaultService vault.VaultService,
	connectorTypesService ConnectorTypesService, connectorsService ConnectorsService,
	keycloakService services.KafkaKeycloakService, connectorNamespaceService ConnectorNamespaceService) *connectorClusterService {
	types, _ := connectionFactory.New().Migrator().ColumnTypes(&dbapi.ConnectorDeployment{})
	nTypes := len(types)
	columnNames := make([]string, nTypes+1)
	for i, columnType := range types {
		columnNames[i] = "connector_deployments." + columnType.Name()
	}
	columnNames[nTypes] = "connector_namespaces.name AS namespace_name"
	return &connectorClusterService{
		connectionFactory:         connectionFactory,
		bus:                       bus,
		connectorTypesService:     connectorTypesService,
		vaultService:              vaultService,
		connectorsService:         connectorsService,
		keycloakService:           keycloakService,
		connectorNamespaceService: connectorNamespaceService,
		deploymentColumns:         columnNames,
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
			Meta: api.Meta{
				ID: result.DeploymentID,
			},
		}).Error; err != nil {
			return errors.GeneralError("Unable to delete connector deployment status who's connectors have been deleted: %s", err)
		}
		if err := dbConn.Delete(&dbapi.ConnectorDeployment{
			Meta: api.Meta{
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
		return errors.GeneralError("failed to create connector: %v", err)
	}

	// create a default namespace for the new cluster
	if err := k.connectorNamespaceService.CreateDefaultNamespace(ctx, resource); err != nil {
		return err
	}
	// TODO: increment connector cluster metrics
	// metrics.IncreaseStatusCountMetric(constants.KafkaRequestStatusAccepted.String())
	return nil
}

func filterClusterToOwnerOrOrg(ctx context.Context, dbConn *gorm.DB) (*gorm.DB, *errors.ServiceError) {

	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return dbConn, errors.Unauthenticated("user not authenticated")
	}
	owner := auth.GetUsernameFromClaims(claims)
	if owner == "" {
		return dbConn, errors.Unauthenticated("user not authenticated")
	}

	orgId := auth.GetOrgIdFromClaims(claims)
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

// Get gets a connector by id from the database
func (k *connectorClusterService) Get(ctx context.Context, id string) (dbapi.ConnectorCluster, *errors.ServiceError) {

	dbConn := k.connectionFactory.New()
	var resource dbapi.ConnectorCluster
	dbConn = dbConn.Where("id = ?", id)

	var err *errors.ServiceError
	dbConn, err = filterClusterToOwnerOrOrg(ctx, dbConn)
	if err != nil {
		return resource, err
	}

	if err := dbConn.First(&resource).Error; err != nil {
		return resource, services.HandleGetError("Connector cluster", "id", id, err)
	}
	return resource, nil
}

// Delete deletes a connector cluster from the database.
func (k *connectorClusterService) Delete(ctx context.Context, id string) *errors.ServiceError {
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return errors.Unauthenticated("user not authenticated")
	}
	owner := auth.GetUsernameFromClaims(claims)
	if owner == "" {
		return errors.Unauthenticated("user not authenticated")
	}

	if err := k.connectionFactory.New().Transaction(func(dbConn *gorm.DB) error {

		var resource dbapi.ConnectorCluster
		if err := dbConn.Where("owner = ? AND id = ?", owner, id).First(&resource).Error; err != nil {
			return services.HandleGetError("Connector cluster", "id", id, err)
		}

		// delete cluster
		if err := dbConn.Delete(&dbapi.ConnectorCluster{}, "id = ?", id).Error; err != nil {
			return services.HandleDeleteError("Connector cluster", "id", id, err)
		}

		// Delete all deployments assigned to the cluster..
		if err := dbConn.Delete(&dbapi.ConnectorDeploymentStatus{}, "id IN (?)",
			dbConn.Table("connector_deployments").Select("id").Where("cluster_id = ?", id)).Error; err != nil {
			return services.HandleDeleteError("Connector deployment status", "cluster_id", id, err)
		}
		if err := dbConn.Delete(&dbapi.ConnectorDeployment{}, "cluster_id = ?", id).Error; err != nil {
			return services.HandleDeleteError("Connector deployment", "cluster_id", id, err)
		}

		// Delete all namespaces assigned to the cluster..
		if err := dbConn.Delete(&dbapi.ConnectorNamespace{}, "cluster_id = ?", id).Error; err != nil {
			return services.HandleDeleteError("Connector namespace", "cluster_id", id, err)
		}

		// Clear the cluster from any connectors that were using it.
		if err := removeConnectorsFromNamespace(dbConn, "connector_namespaces.cluster_id = ?", id); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return services.HandleDeleteError("Connector cluster", "id", id, err)
	}

	glog.V(5).Infof("Removing agent service account for connector cluster %s", id)
	if err := k.keycloakService.DeRegisterConnectorFleetshardOperatorServiceAccount(id); err != nil {
		return errors.GeneralError("failed to remove connector service account for cluster %s: %s", id, err)
	}

	return nil
}

func (k connectorClusterService) ForEach(f func(*dbapi.Connector) *errors.ServiceError, query string, args ...interface{}) *errors.ServiceError {
	dbConn := k.connectionFactory.New()
	rows, err := dbConn.
		Model(&dbapi.Connector{}).
		Where(query, args...).
		Joins("left join connector_statuses on connector_statuses.id = connectors.id").
		Order("version").Rows()

	if err != nil {
		if goerrors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return errors.GeneralError("Unable to list connectors: %s", err)
	}
	defer rows.Close()

	for rows.Next() {
		resource := dbapi.Connector{}

		// ScanRows is a method of `gorm.DB`, it can be used to scan a row into a struct
		err := dbConn.ScanRows(rows, &resource)
		if err != nil {
			return errors.GeneralError("Unable to scan connector: %s", err)
		}

		resource.Status.ID = resource.ID
		err = dbConn.Model(&dbapi.ConnectorStatus{}).First(&resource.Status).Error
		if err != nil {
			return errors.GeneralError("Unable to load connector status: %s", err)
		}

		if serr := f(&resource); serr != nil {
			return serr
		}

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
		return resourceList, pagingMeta, errors.GeneralError("Unable to list connectors: %s", err)
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

func (k *connectorClusterService) UpdateConnectorClusterStatus(ctx context.Context, id string, status dbapi.ConnectorClusterStatus) *errors.ServiceError {
	dbConn := k.connectionFactory.New()
	var resource dbapi.ConnectorCluster

	if err := dbConn.Where("id = ?", id).First(&resource).Error; err != nil {
		return services.HandleGetError("Connector cluster status", "id", id, err)
	}

	if !reflect.DeepEqual(resource.Status, status) {

		if resource.Status.Phase != status.Phase {
			// The phase should not be changing that often.. but when it does,
			// kick off a reconcile to get connectors deployed to the cluster.
			_ = db.AddPostCommitAction(ctx, func() {
				// Wake up the reconcile loop...
				k.bus.Notify("reconcile:connector")
			})
		}

		resource.Status = status
		if err := dbConn.Save(&resource).Error; err != nil {
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
	dbConn = dbConn.Select("client_id").Where("id = ?", clusterID)

	// use Limit(1) and Find() to avoid ErrRecordNotFound
	if err := dbConn.Limit(1).Find(&resource).Error; err != nil {
		return "", services.HandleGetError("Connector cluster client_id", "id", clusterID, err)
	}
	return resource.ClientId, nil
}

func (k *connectorClusterService) UpdateClientId(clusterId string, clientID string) *errors.ServiceError {
	dbConn := k.connectionFactory.New()

	cluster := dbapi.ConnectorCluster{Meta: api.Meta{ID: clusterId}, ClientId: clientID}
	if err := dbConn.Updates(cluster).Error; err != nil {
		return services.HandleGetError("Connector cluster client_id", "id", clusterId, err)
	}
	return nil
}

// Create creates a connector deployment in the database
func (k *connectorClusterService) SaveDeployment(ctx context.Context, resource *dbapi.ConnectorDeployment) *errors.ServiceError {
	dbConn := k.connectionFactory.New()

	if err := dbConn.Save(resource).Error; err != nil {
		return errors.GeneralError("failed to create the connector deployment: %v", err)
	}

	if err := dbConn.Where("id = ?", resource.ID).Select("version").First(&resource).Error; err != nil {
		return services.HandleGetError("Connector Deployment", "id", resource.ID, err)
	}

	if resource.ClusterID != "" {
		_ = db.AddPostCommitAction(ctx, func() {
			k.bus.Notify(fmt.Sprintf("/kafka-connector-clusters/%s/deployments", resource.ClusterID))
		})
	}

	// TODO: increment connector deployment metrics
	// metrics.IncreaseStatusCountMetric(constants.KafkaRequestStatusAccepted.String())
	return nil
}

// List returns all connectors assigned to the cluster
func (k *connectorClusterService) ListConnectorDeployments(ctx context.Context, id string, listArgs *services.ListArguments, gtVersion int64) (dbapi.ConnectorDeploymentList, *api.PagingMeta, *errors.ServiceError) {
	var resourceList dbapi.ConnectorDeploymentList
	dbConn := k.connectionFactory.New()
	dbConn = dbConn.Preload("Status")
	pagingMeta := &api.PagingMeta{
		Page: listArgs.Page,
		Size: listArgs.Size,
	}

	dbConn = dbConn.
		Where("connector_deployments.cluster_id = ?", id)
	if gtVersion != 0 {
		dbConn = dbConn.Where("connector_deployments.version > ?", gtVersion)
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

	// add namespace_name column
	dbConn = k.selectDeploymentColumns(dbConn)

	// execute query
	if err := dbConn.Find(&resourceList).Error; err != nil {
		return resourceList, pagingMeta, errors.GeneralError("Unable to list connector deployments for cluster %s: %s", id, err)
	}

	return resourceList, pagingMeta, nil
}

func (k *connectorClusterService) selectDeploymentColumns(dbConn *gorm.DB) *gorm.DB {
	// join connector_namespaces for namespace name
	return dbConn.Select(k.deploymentColumns).
		Joins("INNER JOIN connector_namespaces ON connector_namespaces.id = namespace_id")
}

func (k *connectorClusterService) UpdateConnectorDeploymentStatus(ctx context.Context, deploymentStatus dbapi.ConnectorDeploymentStatus) *errors.ServiceError {
	dbConn := k.connectionFactory.New()

	// lets get the connector id of the deployment..
	deployment := dbapi.ConnectorDeployment{}
	if err := dbConn.Unscoped().Select("connector_id", "deleted_at").
		Where("id = ?", deploymentStatus.ID).
		First(&deployment).Error; err != nil {
		return services.HandleGetError("connector deployment", "id", deploymentStatus.ID, err)
	}
	if deployment.DeletedAt.Valid {
		return services.HandleGoneError("connector deployment", "id", deploymentStatus.ID)
	}

	if err := dbConn.Model(&deploymentStatus).Where("id = ?", deploymentStatus.ID).Save(&deploymentStatus).Error; err != nil {
		return errors.GeneralError("failed to update deployment status: %s", err.Error())
	}

	connector := dbapi.Connector{}
	if err := dbConn.Select("desired_state").
		Where("id = ?", deployment.ConnectorID).
		First(&connector).Error; err != nil {
		return services.HandleGetError("connector", "id", deployment.ConnectorID, err)
	}

	// TODO: use post the deployment status to the type service to simplify the connector status.
	c := dbapi.ConnectorStatus{
		Phase: deploymentStatus.Phase,
	}

	if deploymentStatus.Phase == dbapi.ConnectorStatusPhaseDeleted {
		// we don't need the deployment anymore...
		if err := deleteConnectorDeployment(dbConn, deploymentStatus.ID); err != nil {
			return err
		}

		// we might need to also delete the connector...
		if connector.DesiredState == dbapi.ConnectorStatusPhaseDeleted {
			if err := k.connectorsService.Delete(ctx, deployment.ConnectorID); err != nil {
				return err
			}
			return nil // return now since we don't need to update the status of the connector
		}
	}

	// update the connector status
	if err := dbConn.Model(&c).Where("id = ?", deployment.ConnectorID).Updates(&c).Error; err != nil {
		return errors.GeneralError("failed to update connector status: %s", err.Error())
	}

	return nil
}

func (k *connectorClusterService) FindReadyNamespace(owner string, orgID string, namespaceID *string) (*dbapi.ConnectorNamespace, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()
	var namespaces dbapi.ConnectorNamespaceList

	if namespaceID != nil {
		dbConn = dbConn.Where("connector_namespaces.id = ?", namespaceID)
	}
	dbConn = dbConn.Joins("INNER JOIN connector_clusters"+
		" ON cluster_id = connector_clusters.id and connector_clusters.status_phase = ?", dbapi.ConnectorClusterPhaseReady)

	if orgID != "" {
		dbConn = dbConn.Where("tenant_organisation_id = ? or tenant_user_id = ?", orgID, owner)
	} else {
		dbConn = dbConn.Where("tenant_owner_id = ?", owner)
	}

	if err := dbConn.Find(&namespaces).Error; err != nil {
		return nil, errors.GeneralError("failed to query ready connector namespace: %v", err.Error())
	}

	n := len(namespaces)
	if n == 1 {
		return namespaces[0], nil
	} else if n > 1 {
		// prioritise and select from multiple namespaces
		ownerNamespaces := dbapi.ConnectorNamespaceList{}
		orgNamespaces := dbapi.ConnectorNamespaceList{}
		for _, namespace := range namespaces {
			if namespace.TenantUserId != nil {
				ownerNamespaces = append(ownerNamespaces, namespace)
			} else {
				orgNamespaces = append(orgNamespaces, namespace)
			}
		}

		// TODO replace with more sophisticated load balancing logic in the future
		// prefer owner tenant
		n := len(ownerNamespaces)
		if n > 0 {
			return ownerNamespaces[rand.Intn(n)], nil
		} else {
			return orgNamespaces[rand.Intn(len(orgNamespaces))], nil
		}
	}

	return nil, nil
}

func Checksum(spec interface{}) (string, error) {
	h := sha1.New()
	err := json.NewEncoder(h).Encode(spec)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func (k *connectorClusterService) GetConnectorWithBase64Secrets(ctx context.Context, resource dbapi.ConnectorDeployment) (dbapi.Connector, *errors.ServiceError) {

	dbConn := k.connectionFactory.New()

	var connector dbapi.Connector
	err := dbConn.Where("id = ?", resource.ConnectorID).First(&connector).Error
	if err != nil {
		return connector, services.HandleGetError("Connector", "id", resource.ConnectorID, err)
	}

	serr := getSecretsFromVaultAsBase64(&connector, k.connectorTypesService, k.vaultService)
	if serr != nil {
		return connector, serr
	}

	return connector, nil
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

	dbConn := k.connectionFactory.New()
	dbConn = k.selectDeploymentColumns(dbConn).Where("connector_id = ?", connectorID)
	if err := dbConn.First(&resource).Error; err != nil {
		return resource, services.HandleGetError("Connector deployment", "connector_id", connectorID, err)
	}
	return
}

func (k *connectorClusterService) GetDeployment(ctx context.Context, id string) (resource dbapi.ConnectorDeployment, serr *errors.ServiceError) {

	dbConn := k.connectionFactory.New()
	dbConn = dbConn.Unscoped().Where("connector_deployments.id = ?", id)
	if err := k.selectDeploymentColumns(dbConn).First(&resource).Error; err != nil {
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
		return upgrades, paging, errors.GeneralError("Unable to list connector deployment upgrades: %s", err)
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
	errorList := errors.ErrorList{}
	for cid, upgrade := range reqConnectors {
		availableUpgrade, ok := availableConnectors[cid]
		if !ok {
			errorList = append(errorList, errors.GeneralError("Type upgrade not available for connector %v", cid))
		}
		// make sure other bits match
		upgrade.DeploymentID = availableUpgrade.DeploymentID
		if !reflect.DeepEqual(upgrade, availableUpgrade) {
			errorList = append(errorList, errors.GeneralError("Type upgrade is outdated for connector %v", cid))
		}
	}
	if len(errorList) != 0 {
		return errors.GeneralError(errorList.Error())
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
				errors.GeneralError("Error updating deployment for connector %s", cid))
		} else {
			if !notificationAdded {
				_ = db.AddPostCommitAction(ctx, func() {
					k.bus.Notify(fmt.Sprintf("/kafka-connector-clusters/%s/deployments", clusterId))
				})
				notificationAdded = true
			}
		}
	}

	if len(errorList) != 0 {
		return errors.GeneralError(errorList.Error())
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
		return upgrades, paging, errors.GeneralError("Unable to list connector deployment upgrades: %s", err)
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
	errorList := errors.ErrorList{}
	for cid, upgrade := range reqConnectors {
		availableUpgrade, ok := availableConnectors[cid]
		if !ok {
			errorList = append(errorList, errors.GeneralError("Operator upgrade not available for connector %s", cid))
		}

		// make sure other bits match
		upgrade.DeploymentID = availableUpgrade.DeploymentID
		upgrade.Operator.Assigned.Type = availableUpgrade.Operator.Assigned.Type
		upgrade.Operator.Assigned.Version = availableUpgrade.Operator.Assigned.Version
		upgrade.Operator.Available.Type = availableUpgrade.Operator.Available.Type
		upgrade.Operator.Available.Version = availableUpgrade.Operator.Available.Version

		if !reflect.DeepEqual(upgrade, availableUpgrade) {
			errorList = append(errorList, errors.GeneralError("Operator upgrade is outdated for connector %s", cid))
		}
	}
	if len(errorList) != 0 {
		return errors.GeneralError(errorList.Error())
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
				errors.GeneralError("Error upgrading deployment for connector %s: %v", cid, serr))
		} else {
			if !notificationAdded {
				_ = db.AddPostCommitAction(ctx, func() {
					k.bus.Notify(fmt.Sprintf("/kafka-connector-clusters/%s/deployments", clusterId))
				})
				notificationAdded = true
			}
		}
	}

	if len(errorList) != 0 {
		return errors.GeneralError(errorList.Error())
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
