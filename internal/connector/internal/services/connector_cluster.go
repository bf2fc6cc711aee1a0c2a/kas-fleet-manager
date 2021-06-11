package services

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	goerrors "errors"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"reflect"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/secrets"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/signalbus"
	"github.com/spyzhov/ajson"
	"gorm.io/gorm"
)

type ConnectorClusterService interface {
	Create(ctx context.Context, resource *api.ConnectorCluster) *errors.ServiceError
	Get(ctx context.Context, id string) (api.ConnectorCluster, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	List(ctx context.Context, listArgs *services.ListArguments) (api.ConnectorClusterList, *api.PagingMeta, *errors.ServiceError)
	Update(ctx context.Context, resource *api.ConnectorCluster) *errors.ServiceError
	UpdateConnectorClusterStatus(ctx context.Context, id string, status api.ConnectorClusterStatus) *errors.ServiceError
	GetConnectorClusterStatus(ctx context.Context, id string) (api.ConnectorClusterStatus, *errors.ServiceError)

	SaveDeployment(ctx context.Context, resource *api.ConnectorDeployment) *errors.ServiceError
	GetConnectorWithBase64Secrets(ctx context.Context, resource api.ConnectorDeployment) (api.Connector, *errors.ServiceError)
	ListConnectorDeployments(ctx context.Context, id string, listArgs *services.ListArguments, gtVersion int64) (api.ConnectorDeploymentList, *api.PagingMeta, *errors.ServiceError)
	UpdateConnectorDeploymentStatus(ctx context.Context, status api.ConnectorDeploymentStatus) *errors.ServiceError
	FindReadyCluster(owner string, orgId string, group string) (*api.ConnectorCluster, *errors.ServiceError)
	GetDeploymentByConnectorId(ctx context.Context, connectorID string) (api.ConnectorDeployment, *errors.ServiceError)
	GetDeployment(ctx context.Context, id string) (api.ConnectorDeployment, *errors.ServiceError)
	GetAvailableDeploymentUpgrades() (upgrades []api.ConnectorDeploymentAvailableUpgrades, serr *errors.ServiceError)
}

var _ ConnectorClusterService = &connectorClusterService{}

type connectorClusterService struct {
	connectionFactory     *db.ConnectionFactory
	bus                   signalbus.SignalBus
	connectorTypesService ConnectorTypesService
	vaultService          services.VaultService
}

func NewConnectorClusterService(connectionFactory *db.ConnectionFactory, bus signalbus.SignalBus, vaultService services.VaultService, connectorTypesService ConnectorTypesService) *connectorClusterService {
	return &connectorClusterService{
		connectionFactory:     connectionFactory,
		bus:                   bus,
		connectorTypesService: connectorTypesService,
		vaultService:          vaultService,
	}
}

// Create creates a connector cluster in the database
func (k *connectorClusterService) Create(ctx context.Context, resource *api.ConnectorCluster) *errors.ServiceError {
	dbConn := k.connectionFactory.New()
	if err := dbConn.Save(resource).Error; err != nil {
		return errors.GeneralError("failed to create connector: %v", err)
	}
	// TODO: increment connector cluster metrics
	// metrics.IncreaseStatusCountMetric(constants.KafkaRequestStatusAccepted.String())
	return nil
}

// Get gets a connector by id from the database
func (k *connectorClusterService) Get(ctx context.Context, id string) (api.ConnectorCluster, *errors.ServiceError) {

	dbConn := k.connectionFactory.New()
	var resource api.ConnectorCluster
	dbConn = dbConn.Where("id = ?", id)

	var err *errors.ServiceError
	dbConn, err = filterToOwnerOrOrg(ctx, dbConn)
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

	dbConn := k.connectionFactory.New()
	var resource api.ConnectorCluster
	if err := dbConn.Where("owner = ? AND id = ?", owner, id).First(&resource).Error; err != nil {
		return services.HandleGetError("Connector cluster", "id", id, err)
	}

	// TODO: implement soft delete instead?
	if err := dbConn.Delete(&resource).Error; err != nil {
		return errors.GeneralError("unable to delete connector with id %s: %s", resource.ID, err)
	}

	return nil
}

// List returns all connector clusters visible to the user within the requested paging window.
func (k *connectorClusterService) List(ctx context.Context, listArgs *services.ListArguments) (api.ConnectorClusterList, *api.PagingMeta, *errors.ServiceError) {
	var resourceList api.ConnectorClusterList
	dbConn := k.connectionFactory.New()
	pagingMeta := &api.PagingMeta{
		Page: listArgs.Page,
		Size: listArgs.Size,
	}

	var err *errors.ServiceError
	dbConn, err = filterToOwnerOrOrg(ctx, dbConn)
	if err != nil {
		return nil, nil, err
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
	dbConn := k.connectionFactory.New()
	var resource api.ConnectorCluster

	if err := dbConn.Where("id = ?", id).First(&resource).Error; err != nil {
		return services.HandleGetError("Connector cluster status", "id", id, err)
	}

	if !reflect.DeepEqual(resource.Status, status) {
		resource.Status = status
		if err := dbConn.Save(&resource).Error; err != nil {
			return errors.NewWithCause(errors.ErrorGeneral, err, "failed to update status")
		}
		_ = db.AddPostCommitAction(ctx, func() {
			// Wake up the reconcile loop...
			k.bus.Notify("reconcile:connector")
		})
	}

	return nil
}

// Get gets a connector by id from the database
func (k *connectorClusterService) GetConnectorClusterStatus(ctx context.Context, id string) (api.ConnectorClusterStatus, *errors.ServiceError) {

	dbConn := k.connectionFactory.New()
	var resource api.ConnectorCluster
	dbConn = dbConn.Select("status_phase, status_version, status_conditions, status_operators").Where("id = ?", id)

	if err := dbConn.First(&resource).Error; err != nil {
		return resource.Status, services.HandleGetError("Connector cluster status", "id", id, err)
	}
	return resource.Status, nil
}

// Create creates a connector deployment in the database
func (k *connectorClusterService) SaveDeployment(ctx context.Context, resource *api.ConnectorDeployment) *errors.ServiceError {
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
func (k *connectorClusterService) ListConnectorDeployments(ctx context.Context, id string, listArgs *services.ListArguments, gtVersion int64) (api.ConnectorDeploymentList, *api.PagingMeta, *errors.ServiceError) {
	var resourceList api.ConnectorDeploymentList
	dbConn := k.connectionFactory.New()
	dbConn = dbConn.Preload("Status")
	pagingMeta := &api.PagingMeta{
		Page: listArgs.Page,
		Size: listArgs.Size,
	}

	dbConn = dbConn.Where("cluster_id = ?", id)
	if gtVersion != 0 {
		dbConn = dbConn.Where("version > ?", gtVersion)
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
	dbConn = dbConn.Order("version")

	// execute query
	if err := dbConn.Find(&resourceList).Error; err != nil {
		return resourceList, pagingMeta, errors.GeneralError("Unable to list connector deployments for cluster %s: %s", id, err)
	}

	return resourceList, pagingMeta, nil
}

func (k *connectorClusterService) UpdateConnectorDeploymentStatus(ctx context.Context, resource api.ConnectorDeploymentStatus) *errors.ServiceError {
	dbConn := k.connectionFactory.New()

	// lets get the connector id of the deployment..
	deployment := api.ConnectorDeployment{}
	if err := dbConn.Select("connector_id").
		Where("id = ?", resource.ID).
		First(&deployment).Error; err != nil {
		return services.HandleGetError("connector deployment", "id", resource.ID, err)
	}

	if err := dbConn.Model(&resource).Where("id = ?", resource.ID).Save(&resource).Error; err != nil {
		return errors.GeneralError("failed to update deployment status: %s", err.Error())
	}

	// TODO: use post the deployment status to the type service to simplify the connector status.
	c := api.ConnectorStatus{
		Phase: resource.Phase,
	}
	if err := dbConn.Model(&c).Where("id = ?", deployment.ConnectorID).Updates(&c).Error; err != nil {
		return errors.GeneralError("failed to update connector status: %s", err.Error())
	}

	return nil
}

func (k *connectorClusterService) FindReadyCluster(owner string, orgId string, connectorClusterId string) (*api.ConnectorCluster, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()
	var resource api.ConnectorCluster

	dbConn = dbConn.Where("id = ? AND status_phase = ?", connectorClusterId, api.ConnectorClusterPhaseReady)

	if orgId != "" {
		dbConn = dbConn.Where("organisation_id = ?", orgId)
	} else {
		dbConn = dbConn.Where("owner = ?", owner)
	}

	if err := dbConn.First(&resource).Error; err != nil {
		if goerrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errors.GeneralError("failed to query ready addon connector cluster: %v", err.Error())
	}
	return &resource, nil
}

func Checksum(spec interface{}) (string, error) {
	h := sha1.New()
	err := json.NewEncoder(h).Encode(spec)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func (k *connectorClusterService) GetConnectorWithBase64Secrets(ctx context.Context, resource api.ConnectorDeployment) (api.Connector, *errors.ServiceError) {

	dbConn := k.connectionFactory.New()

	var connector api.Connector
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

func getSecretsFromVaultAsBase64(resource *api.Connector, cts ConnectorTypesService, vault services.VaultService) *errors.ServiceError {
	ct, err := cts.Get(resource.ConnectorTypeId)
	if err != nil {
		return errors.BadRequest("invalid connector type id: %s", resource.ConnectorTypeId)
	}
	// move secrets to a vault.

	if resource.Kafka.ClientSecretRef != "" {
		v, err := vault.GetSecretString(resource.Kafka.ClientSecretRef)
		if err != nil {
			return errors.GeneralError("could not get kafka client secrets from the vault")
		}
		encoded := base64.StdEncoding.EncodeToString([]byte(v))
		resource.Kafka.ClientSecret = encoded
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
			return errors.GeneralError("could not get connectors secrets from the vault")
		}
		resource.ConnectorSpec = updated
	}
	return nil
}

func (k *connectorClusterService) GetDeploymentByConnectorId(ctx context.Context, connectorID string) (resource api.ConnectorDeployment, serr *errors.ServiceError) {

	dbConn := k.connectionFactory.New()
	dbConn = dbConn.Where("connector_id = ?", connectorID)
	if err := dbConn.First(&resource).Error; err != nil {
		return resource, services.HandleGetError("Connector deployment", "connector_id", connectorID, err)
	}
	return
}
func (k *connectorClusterService) GetDeployment(ctx context.Context, id string) (resource api.ConnectorDeployment, serr *errors.ServiceError) {

	dbConn := k.connectionFactory.New()
	dbConn = dbConn.Where("id = ?", id)
	if err := dbConn.First(&resource).Error; err != nil {
		return resource, services.HandleGetError("Connector deployment", "id", id, err)
	}
	return
}

func (k *connectorClusterService) GetAvailableDeploymentUpgrades() (upgrades []api.ConnectorDeploymentAvailableUpgrades, serr *errors.ServiceError) {

	type Result struct {
		DeploymentID             string
		ConnectorTypeUpgrade     bool
		ConnectorOperatorUpgrade bool
		ConnectorTypeUpgradeFrom int64
		ConnectorTypeUpgradeTo   int64
		ConnectorTypeId          string
		Channel                  string
		ConnectorOperators       api.JSON
	}

	results := []Result{}
	dbConn := k.connectionFactory.New()
	dbConn = dbConn.Table("connector_deployments")
	dbConn = dbConn.Select(
		"connector_deployments.id AS deployment_id",
		"connector_shard_metadata.latest_id IS NOT NULL AS connector_type_upgrade",
		"connector_shard_metadata.id AS connector_type_upgrade_from",
		"connector_shard_metadata.latest_id AS connector_type_upgrade_to",
		"connector_shard_metadata.connector_type_id",
		"connector_shard_metadata.channel",
		"connector_deployment_statuses.upgrade_available AS connector_operator_upgrade",
		"connector_deployment_statuses.operators AS connector_operators",
	)
	dbConn = dbConn.Joins("LEFT JOIN connector_shard_metadata ON connector_shard_metadata.id = connector_deployments.connector_type_channel_id")
	dbConn = dbConn.Joins("LEFT JOIN connector_deployment_statuses ON connector_deployment_statuses.id = connector_deployments.id")
	dbConn = dbConn.Where("connector_shard_metadata.latest_id IS NOT NULL")
	dbConn = dbConn.Or("connector_deployment_statuses.upgrade_available")

	if err := dbConn.Scan(&results).Error; err != nil {
		return upgrades, errors.GeneralError("Unable to list connector deployment upgrades: %s", err)
	}

	upgrades = make([]api.ConnectorDeploymentAvailableUpgrades, len(results))
	for i, r := range results {
		upgrades[i] = api.ConnectorDeploymentAvailableUpgrades{
			DeploymentID:    r.DeploymentID,
			ConnectorTypeId: r.ConnectorTypeId,
			Channel:         r.Channel,
		}

		var operators openapi.ConnectorDeploymentStatusOperators
		if r.ConnectorTypeUpgrade {

			upgrades[i].ShardMetadata = &api.ConnectorTypeUpgrade{
				AssignedId:  r.ConnectorTypeUpgradeFrom,
				AvailableId: r.ConnectorTypeUpgradeTo,
			}

		}
		if r.ConnectorOperatorUpgrade {
			err := json.Unmarshal([]byte(r.ConnectorOperators), &operators)
			if err != nil {
				return upgrades, errors.GeneralError("converting ConnectorDeploymentStatusOperators: %s", err)
			}

			upgrades[i].Operator = &api.ConnectorOperatorUpgrade{
				Assigned: api.ConnectorOperator{
					Id:      operators.Assigned.Id,
					Type:    operators.Assigned.Type,
					Version: operators.Assigned.Version,
				},
				Available: api.ConnectorOperator{
					Id:      operators.Available.Id,
					Type:    operators.Available.Type,
					Version: operators.Available.Version,
				},
			}
		}

	}
	return
}
