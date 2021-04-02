package services

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	goerrors "errors"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/secrets"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/signalbus"
	"github.com/jinzhu/gorm"
	"github.com/spyzhov/ajson"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

type ConnectorClusterService interface {
	Create(ctx context.Context, resource *api.ConnectorCluster) *errors.ServiceError
	Get(ctx context.Context, id string) (api.ConnectorCluster, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	List(ctx context.Context, listArgs *ListArguments) (api.ConnectorClusterList, *api.PagingMeta, *errors.ServiceError)
	Update(ctx context.Context, resource *api.ConnectorCluster) *errors.ServiceError
	UpdateConnectorClusterStatus(ctx context.Context, id string, status api.ConnectorClusterStatus) *errors.ServiceError
	GetConnectorClusterStatus(ctx context.Context, id string) (api.ConnectorClusterStatus, *errors.ServiceError)

	CreateDeployment(ctx context.Context, resource *api.ConnectorDeployment) *errors.ServiceError
	GetConnectorClusterSpec(ctx context.Context, resource api.ConnectorDeployment) (openapi.ConnectorDeploymentSpec, *errors.ServiceError)
	ListConnectorDeployments(ctx context.Context, id string, listArgs *ListArguments, gtVersion int64) (api.ConnectorDeploymentList, *api.PagingMeta, *errors.ServiceError)
	UpdateConnectorDeploymentStatus(ctx context.Context, id string, cid string, status api.ConnectorDeploymentStatus) *errors.ServiceError
	FindReadyCluster(owner string, orgId string, group string) (*api.ConnectorCluster, *errors.ServiceError)
}

var _ ConnectorClusterService = &connectorClusterService{}

type connectorClusterService struct {
	connectionFactory     *db.ConnectionFactory
	bus                   signalbus.SignalBus
	connectorTypesService ConnectorTypesService
	vaultService          VaultService
}

func NewConnectorClusterService(connectionFactory *db.ConnectionFactory, bus signalbus.SignalBus, vaultService VaultService, connectorTypesService ConnectorTypesService) *connectorClusterService {
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
		return resource, handleGetError("Connector cluster", "id", id, err)
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
		return handleGetError("Connector cluster", "id", id, err)
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

	var err *errors.ServiceError
	dbConn, err = filterToOwnerOrOrg(ctx, dbConn)
	if err != nil {
		return nil, nil, err
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
		return handleGetError("Connector cluster status", "id", id, err)
	}

	if !reflect.DeepEqual(resource.Status, status) {
		resource.Status = status
		if err := dbConn.Save(&resource).Error; err != nil {
			return errors.GeneralError("failed to update status: %s", err.Error())
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
		return resource.Status, handleGetError("Connector cluster status", "id", id, err)
	}
	return resource.Status, nil
}

// Create creates a connector cluster in the database
func (k *connectorClusterService) CreateDeployment(ctx context.Context, resource *api.ConnectorDeployment) *errors.ServiceError {
	dbConn := k.connectionFactory.New()
	if err := dbConn.Save(resource).Error; err != nil {
		return errors.GeneralError("failed to create connector: %v", err)
	}

	if resource.ClusterID != "" {
		_ = db.AddPostCommitAction(ctx, func() {
			k.bus.Notify(fmt.Sprintf("/kafka-connector-clusters/%s/deployments", resource.ClusterID))
		})
	}

	// TODO: increment connector cluster metrics
	// metrics.IncreaseStatusCountMetric(constants.KafkaRequestStatusAccepted.String())
	return nil
}

// List returns all connectors assigned to the cluster
func (k *connectorClusterService) ListConnectorDeployments(ctx context.Context, id string, listArgs *ListArguments, gtVersion int64) (api.ConnectorDeploymentList, *api.PagingMeta, *errors.ServiceError) {
	var resourceList api.ConnectorDeploymentList
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
		return resourceList, pagingMeta, errors.GeneralError("Unable to list connector deployments for cluster %s: %s", id, err)
	}

	return resourceList, pagingMeta, nil
}

func (k *connectorClusterService) UpdateConnectorDeploymentStatus(ctx context.Context, clusterId string, deploymentId string, status api.ConnectorDeploymentStatus) *errors.ServiceError {
	dbConn := k.connectionFactory.New()

	resource := api.ConnectorDeployment{
		Status: status,
	}
	if err := dbConn.Model(&resource).Where("cluster_id = ?", clusterId).Where("id = ?", deploymentId).Update(&resource).Error; err != nil {
		return errors.GeneralError("failed to update deployment status: %s", err.Error())
	}

	// lets get the connector id of the deployment..
	if err := dbConn.Select("connector_id").
		Where("cluster_id = ?", clusterId).
		Where("id = ?", deploymentId).
		First(&resource).Error; err != nil {
		return handleGetError("connector deployment", "id", deploymentId, err)
	}

	c := api.Connector{
		Status: status.Phase,
	}
	if err := dbConn.Model(&c).Where("id = ?", resource.ConnectorID).Update(&c).Error; err != nil {
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

var StaleDeployment = errors.BadRequest("deployment needs to be updated")

func Checksum(spec interface{}) (string, error) {
	h := sha1.New()
	err := json.NewEncoder(h).Encode(spec)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func (k *connectorClusterService) GetConnectorClusterSpec(ctx context.Context, resource api.ConnectorDeployment) (result openapi.ConnectorDeploymentSpec, serr *errors.ServiceError) {

	dbConn := k.connectionFactory.New()

	var connector api.Connector
	err := dbConn.Select("connector_type_id, connector_spec").Where("id = ? AND version=?", resource.ConnectorID, resource.ConnectorVersion).First(&connector).Error
	if err != nil {
		fmt.Println(err)
		return result, StaleDeployment
	}

	serr = getSecretsFromVaultAsBase64(&connector, k.connectorTypesService, k.vaultService)
	if serr != nil {
		return
	}

	connectorSpec, err := connector.ConnectorSpec.Object()
	if err != nil {
		return result, errors.BadRequest("could decode connector spec %s: %v", resource.ConnectorID, err)
	}

	result, err = reify(context.Background(), resource.ConnectorTypeService, connector.ConnectorTypeId, openapi.ConnectorReifyRequest{
		ConnectorId:     resource.ConnectorID,
		ResourceVersion: resource.ConnectorVersion,
		KafkaId:         resource.ClusterID,
		ClusterId:       resource.ClusterID,
		//ClusterStatus:   clusterStatus,
		ConnectorSpec: connectorSpec,
	})
	if err != nil {
		return result, errors.GeneralError("failed to reify deployment for connector %s: %v", resource.ConnectorID, err)
	}

	return
}

func getSecretsFromVaultAsBase64(resource *api.Connector, cts ConnectorTypesService, vault VaultService) *errors.ServiceError {
	ct, err := cts.Get(resource.ConnectorTypeId)
	if err != nil {
		return errors.BadRequest("invalid connector type id: %s", resource.ConnectorTypeId)
	}
	// move secrets to a vault.
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
			return errors.GeneralError("could not store connectors secrets in the vault")
		}
		resource.ConnectorSpec = updated
	}
	return nil
}

func reify(ctx context.Context, serviceUrl string, typeId string, reifyRequest openapi.ConnectorReifyRequest) (openapi.ConnectorDeploymentSpec, error) {

	result := openapi.ConnectorDeploymentSpec{}
	endpoint := fmt.Sprintf("%s/api/managed-services-api/v1/kafka-connector-types/%s/reify/spec", serviceUrl, typeId)

	requestBytes, err := json.Marshal(reifyRequest)
	if err != nil {
		return result, err
	}

	// glog.Infoln("reify request: ", string(requestBytes))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestBytes))
	if err != nil {
		return result, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	c := &http.Client{}

	resp, err := c.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		all, err := ioutil.ReadAll(io.LimitReader(resp.Body, 400))
		if err != nil {
			return result, err
		}
		return result, fmt.Errorf("expected 200: but got %d, body: %s", resp.StatusCode, string(all))
	}
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		return result, fmt.Errorf("expected Content-Type: application/json, but got %s", contentType)
	}

	//all, err := io.ReadAll(resp.Body)
	//if err != nil {
	//	return result, err
	//}
	//glog.Infoln("reify response: ", string(all))
	//
	//err = json.Unmarshal(all, &result)
	//if err != nil {
	//	return result, err
	//}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return result, err
	}

	return result, nil

}
