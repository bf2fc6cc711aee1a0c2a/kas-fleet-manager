package services

import (
	"context"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/phase"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/queryparser"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	"gorm.io/gorm"
	"math/rand"
	"time"
)

type ConnectorNamespaceService interface {
	Create(ctx context.Context, request *dbapi.ConnectorNamespace) *errors.ServiceError
	Update(ctx context.Context, request *dbapi.ConnectorNamespace) *errors.ServiceError
	Get(ctx context.Context, namespaceID string) (*dbapi.ConnectorNamespace, *errors.ServiceError)
	List(ctx context.Context, clusterIDs []string, listArguments *services.ListArguments) (dbapi.ConnectorNamespaceList, *api.PagingMeta, *errors.ServiceError)
	Delete(ctx context.Context, namespaceId string) *errors.ServiceError
	SetEvalClusterId(request *dbapi.ConnectorNamespace) *errors.ServiceError
	GetOwnerClusterIds(userId string, ownerClusterIds *[]string) *errors.ServiceError
	GetOrgClusterIds(userId string, organisationId string, orgClusterIds *[]string) *errors.ServiceError
	CreateDefaultNamespace(ctx context.Context, connectorCluster *dbapi.ConnectorCluster) *errors.ServiceError
	GetExpiredNamespaceIds() ([]string, *errors.ServiceError)
	UpdateConnectorNamespaceStatus(ctx context.Context, namespaceID string, status *dbapi.ConnectorNamespaceStatus) *errors.ServiceError
	DeleteNamespaceAndConnectorDeployments(ctx context.Context, dbConn *gorm.DB, query interface{}, values ...interface{}) (bool, bool, *errors.ServiceError)
	ReconcileDeletingNamespaces() (int, []*errors.ServiceError)
}

var _ ConnectorNamespaceService = &connectorNamespaceService{}

type connectorNamespaceService struct {
	connectionFactory *db.ConnectionFactory
	connectorsConfig  *config.ConnectorsConfig
	bus               signalbus.SignalBus
}

func init() {
	// random seed
	rand.Seed(time.Now().UnixNano())
}

func NewConnectorNamespaceService(factory *db.ConnectionFactory, config *config.ConnectorsConfig, bus signalbus.SignalBus) *connectorNamespaceService {
	return &connectorNamespaceService{
		connectionFactory: factory,
		connectorsConfig:  config,
		bus:               bus,
	}
}

func (k *connectorNamespaceService) GetOrgClusterIds(userId, organisationId string, orgClusterIds *[]string) *errors.ServiceError {
	dbConn := k.connectionFactory.New()
	if err := dbConn.Raw("SELECT id FROM connector_clusters WHERE deleted_at is null AND owner <> ? AND organisation_id = ?", userId, organisationId).
		Scan(orgClusterIds).Error; err != nil {
		return errors.Unauthorized("failed to get cluster id for organisation %v: %v", organisationId, err)
	}
	return nil
}

func (k *connectorNamespaceService) GetOwnerClusterIds(userId string, ownerClusterIds *[]string) *errors.ServiceError {
	dbConn := k.connectionFactory.New()
	if err := dbConn.Raw("SELECT id FROM connector_clusters WHERE deleted_at is null AND owner = ?", userId).
		Scan(ownerClusterIds).Error; err != nil {
		return errors.Unauthorized("failed to get cluster id for user %v: %v", userId, err)
	}
	return nil
}

func (k *connectorNamespaceService) SetEvalClusterId(request *dbapi.ConnectorNamespace) *errors.ServiceError {
	var availableClusters []string

	// get eval clusters
	dbConn := k.connectionFactory.New()
	if err := dbConn.Raw("SELECT id FROM connector_clusters WHERE deleted_at is null AND organisation_id IN ?",
		k.connectorsConfig.ConnectorEvalOrganizations).
		Scan(&availableClusters).Error; err != nil {
		return errors.Unauthorized("failed to get eval cluster id: %v", err)
	}

	numOrgClusters := len(availableClusters)
	if numOrgClusters == 0 {
		return errors.Unauthorized("no eval clusters")
	} else if numOrgClusters == 1 {
		request.ClusterId = availableClusters[0]
	} else {
		// TODO add support for load balancing strategies
		// pick a cluster at random
		request.ClusterId = availableClusters[rand.Intn(numOrgClusters)]
	}

	// also set expiration duration
	expiry := time.Now().Add(k.connectorsConfig.ConnectorEvalDuration)
	request.Expiration = &expiry
	return nil
}

func (k *connectorNamespaceService) Create(ctx context.Context, request *dbapi.ConnectorNamespace) *errors.ServiceError {

	dbConn := k.connectionFactory.New()
	if err := dbConn.Create(request).Error; err != nil {
		return errors.GeneralError("failed to create connector namespace: %v", err)
	}

	// TODO: increment connector namespace metrics
	// metrics.IncreaseStatusCountMetric(constants.KafkaRequestStatusAccepted.String())
	return nil
}

func (k *connectorNamespaceService) Update(ctx context.Context, request *dbapi.ConnectorNamespace) *errors.ServiceError {

	dbConn := k.connectionFactory.New()
	if err := dbConn.Save(request).Error; err != nil {
		return errors.GeneralError("failed to update connector namespace: %v", err)
	}
	// reload namespace to get version update
	if err := dbConn.Select("version").First(request).Error; err != nil {
		return errors.GeneralError("failed to read updated connector namespace: %v", err)
	}

	// TODO: increment connector namespace metrics
	// metrics.IncreaseStatusCountMetric(constants.KafkaRequestStatusAccepted.String())
	return nil
}

func (k *connectorNamespaceService) Get(ctx context.Context, namespaceID string) (*dbapi.ConnectorNamespace, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()
	result := &dbapi.ConnectorNamespace{
		Model: db.Model{
			ID: namespaceID,
		},
	}
	if err := dbConn.First(result).Error; err != nil {
		return nil, errors.GeneralError("failed to get connector namespace: %v", err)
	}

	// TODO: increment connector namespace metrics
	// metrics.IncreaseStatusCountMetric(constants.KafkaRequestStatusAccepted.String())
	return result, nil
}

var validNamespaceColumns = []string{"name", "cluster_id", "owner", "expiration", "tenant_user_id", "tenant_organisation_id"}

func (k *connectorNamespaceService) List(ctx context.Context, clusterIDs []string,
	listArguments *services.ListArguments) (dbapi.ConnectorNamespaceList, *api.PagingMeta, *errors.ServiceError) {
	var resourceList dbapi.ConnectorNamespaceList
	pagingMeta := api.PagingMeta{
		Page:  listArguments.Page,
		Size:  listArguments.Size,
		Total: 0,
	}
	dbConn := k.connectionFactory.New().Model(&resourceList)
	if len(clusterIDs) != 0 {
		dbConn = dbConn.Where("cluster_id IN ?", clusterIDs)
	}

	// Apply search query
	if len(listArguments.Search) > 0 {
		queryParser := coreServices.NewQueryParser(validNamespaceColumns...)
		searchDbQuery, err := queryParser.Parse(listArguments.Search)
		if err != nil {
			return resourceList, &pagingMeta, errors.NewWithCause(errors.ErrorFailedToParseSearch, err, "Unable to list connector namespace requests: %s", err.Error())
		}
		dbConn = dbConn.Where(searchDbQuery.Query, searchDbQuery.Values...)
	}

	// set total, limit and paging (based on https://gitlab.cee.redhat.com/service/api-guidelines#user-content-paging)
	total := int64(pagingMeta.Total)
	dbConn.Count(&total)
	pagingMeta.Total = int(total)
	if pagingMeta.Size > pagingMeta.Total {
		pagingMeta.Size = pagingMeta.Total
	}
	dbConn = dbConn.Offset((pagingMeta.Page - 1) * pagingMeta.Size).Limit(pagingMeta.Size)

	if len(listArguments.OrderBy) == 0 {
		// default orderBy name
		dbConn = dbConn.Order("name ASC")
	}

	// Set the order by arguments if any
	for _, orderByArg := range listArguments.OrderBy {
		dbConn = dbConn.Order(orderByArg)
	}

	// execute query
	result := dbConn.
		Preload("Annotations").
		Preload("TenantUser").
		Preload("TenantOrganisation").
		Find(&resourceList)
	if result.Error != nil {
		return nil, nil, errors.GeneralError("failed to get connector namespaces: %v", result.Error)
	}

	// TODO: increment connector namespace metrics
	// metrics.IncreaseStatusCountMetric(constants.KafkaRequestStatusAccepted.String())
	return resourceList, &pagingMeta, nil
}

func (k *connectorNamespaceService) Delete(ctx context.Context, namespaceId string) *errors.ServiceError {

	var namespaceDeleted, connectorsDeleted bool
	if err := k.connectionFactory.New().Transaction(func(dbConn *gorm.DB) error {

		var resource dbapi.ConnectorNamespace
		if err := dbConn.Where("id = ?", namespaceId).Select("id", "cluster_id", "status_phase").
			First(&resource).Error; err != nil {
			return services.HandleGetError("Connector namespace", "id", namespaceId, err)
		}

		var cluster dbapi.ConnectorCluster
		if err := dbConn.Where("id = ?", resource.ClusterId).Select("id", "status_phase").
			First(&cluster).Error; err != nil {
			return services.HandleGetError("Connector namespace", "id", namespaceId, err)
		}

		if _, err := phase.PerformNamespaceOperation(&cluster, &resource, phase.DeleteNamespace,
			func(ns *dbapi.ConnectorNamespace) *errors.ServiceError {

				var serr *errors.ServiceError
				namespaceDeleted, connectorsDeleted, serr = k.DeleteNamespaceAndConnectorDeployments(ctx, dbConn, "connector_namespaces.id = ?", namespaceId)
				if !namespaceDeleted {
					return errors.NotFound("Connector namespace %s not found", namespaceId)
				}
				return serr

			}); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return services.HandleDeleteError("Connector namespace", "id", namespaceId, err)
	}

	// reconcile deleting namespaces and connectors
	defer func() {
		if namespaceDeleted {
			k.bus.Notify("reconcile:connector_namespace")
		}
	}()
	defer func() {
		if connectorsDeleted {
			k.bus.Notify("reconcile:connector")
		}
	}()

	// TODO: increment connector namespace metrics
	// metrics.IncreaseStatusCountMetric(constants.KafkaRequestStatusAccepted.String())
	return nil
}

// TODO make this a configurable property in the future
const defaultNamespaceName = "default-connector-namespace"

func (k *connectorNamespaceService) CreateDefaultNamespace(ctx context.Context, connectorCluster *dbapi.ConnectorCluster) *errors.ServiceError {
	owner := connectorCluster.Owner
	organisationId := connectorCluster.OrganisationId

	kind := public.CONNECTORNAMESPACETENANTKIND_USER
	if organisationId != "" {
		kind = public.CONNECTORNAMESPACETENANTKIND_ORGANISATION
	}

	namespaceRequest, err := presenters.ConvertConnectorNamespaceRequest(&public.ConnectorNamespaceRequest{
		Name: defaultNamespaceName,
		Annotations: []public.ConnectorNamespaceRequestMetaAnnotations{
			{
				Name:  "connector_mgmt.api.openshift.com/profile",
				Value: "default-profile",
			},
		},
		ClusterId: connectorCluster.ID,
		Kind:      kind,
	}, owner, organisationId)

	if err != nil {
		return err
	}
	return k.Create(ctx, namespaceRequest)
}

func (k *connectorNamespaceService) GetExpiredNamespaceIds() ([]string, *errors.ServiceError) {
	var result []string
	dbConn := k.connectionFactory.New()
	now := time.Now()
	if err := dbConn.Model(&dbapi.ConnectorNamespace{}).Select("id").
		Where("expiration < ?", now).Find(&result).Error; err != nil {
		return nil, services.HandleGetError("Connector namespace", "expiration", now, err)
	}
	// update phase to 'deleting' for all expired namespaces
	if err := dbConn.Model(&dbapi.ConnectorNamespace{}).
		Where("id in ?", result).
		Update("status_phase", dbapi.ConnectorNamespacePhaseDeleting).Error; err != nil {
		return nil, services.HandleUpdateError("Connector namespace", err)
	}
	return result, nil
}

func (k *connectorNamespaceService) UpdateConnectorNamespaceStatus(ctx context.Context, namespaceID string, status *dbapi.ConnectorNamespaceStatus) *errors.ServiceError {

	if err := k.connectionFactory.New().Transaction(func(dbConn *gorm.DB) error {
		var namespace dbapi.ConnectorNamespace
		if err := dbConn.Select("id", "cluster_id", "status_phase").Where("id = ?", namespaceID).First(&namespace).Error; err != nil {
			return services.HandleGetError("Connector namespace", "id", namespaceID, err)
		}
		var cluster dbapi.ConnectorCluster
		if err := dbConn.Select("id", "status_phase").Where("id = ?", namespace.ClusterId).First(&cluster).Error; err != nil {
			return services.HandleGetError("Connector namespace", "id", namespaceID, err)
		}

		if _, err := phase.PerformNamespaceOperation(&cluster, &namespace, phase.ConnectNamespace,
			func(namespace *dbapi.ConnectorNamespace) *errors.ServiceError {

				// copy updated values from agent provided status
				namespace.ID = namespaceID
				namespace.Status.ConnectorsDeployed = status.ConnectorsDeployed
				namespace.Status.Conditions = status.Conditions

				if err := dbConn.Updates(namespace).Error; err != nil {
					return services.HandleUpdateError("Connector namespace", err)
				}
				return nil
			}); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return services.HandleUpdateError("Connector namespace", err)
	}

	return nil
}

func (k *connectorNamespaceService) DeleteNamespaceAndConnectorDeployments(ctx context.Context, dbConn *gorm.DB, query interface{}, values ...interface{}) (bool, bool, *errors.ServiceError) {
	var namespaces []dbapi.ConnectorNamespace

	if err := dbConn.Model(&dbapi.ConnectorNamespace{}).Select("id", "cluster_id").
		Where(query, values).Find(&namespaces).Error; err != nil {
		return false, false, services.HandleGetError("Connector namespace", fmt.Sprintf("%s", query), values, err)
	}

	// no namespaces
	if len(namespaces) == 0 {
		return false, false, nil
	}
	var namespaceIds []string
	clusterIds := make(map[string]struct{})
	var empty struct{}
	for _, ns := range namespaces {
		namespaceIds = append(namespaceIds, ns.ID)
		clusterIds[ns.ClusterId] = empty
	}

	// Set namespace phase to "deleting"
	if err := dbConn.Model(&dbapi.ConnectorNamespace{}).Where("id IN ?", namespaceIds).
		Update("status_phase", dbapi.ConnectorNamespacePhaseDeleting).Error; err != nil {
		return false, false, services.HandleUpdateError("Connector namespace", err)
	}

	// get all connectors that are currently not being deleted
	var connectorIds []string
	if err := dbConn.Model(&dbapi.Connector{}).Select("id").
		Where("namespace_id IN ? AND desired_state != ?", namespaceIds, dbapi.ConnectorDeleted).
		Find(&connectorIds).Error; err != nil {
		return false, false, services.HandleGetError("Connector", "namespace_id", namespaces, err)
	}

	// no connectors
	if len(connectorIds) == 0 {
		return true, false, nil
	}

	// set connector desired state to "unassigned" and status to "deleting" to remove from namespaces
	if err := dbConn.Where("deleted_at IS NULL AND id IN ?", connectorIds).
		Updates(&dbapi.Connector{DesiredState: dbapi.ConnectorUnassigned}).Error; err != nil {
		return false, false, services.HandleUpdateError("Connector", err)
	}
	if err := dbConn.Where("deleted_at IS NULL AND id IN ?", connectorIds).
		Updates(&dbapi.ConnectorStatus{Phase: dbapi.ConnectorStatusPhaseDeleting}).Error; err != nil {
		return false, false, services.HandleUpdateError("Connector", err)
	}
	// mark all deployment statuses as "assigning"
	if err := dbConn.Where("deleted_at IS NULL AND id IN ?", connectorIds).
		Updates(&dbapi.ConnectorDeploymentStatus{Phase: dbapi.ConnectorStatusPhaseDeleting}).Error; err != nil {
		return false, false, services.HandleUpdateError("Connector", err)
	}

	// notify deployment status update
	_ = db.AddPostCommitAction(ctx, func() {
		for cid := range clusterIds {
			k.bus.Notify(fmt.Sprintf("/kafka-connector-clusters/%s/deployments", cid))
		}
	})

	return true, true, nil
}

// ReconcileDeletingNamespaces deletes empty namespaces in phase deleting with no connectors,
func (k *connectorNamespaceService) ReconcileDeletingNamespaces() (int, []*errors.ServiceError) {
	count := 0
	var errs []*errors.ServiceError
	if err := k.connectionFactory.New().Transaction(func(dbConn *gorm.DB) error {

		// get ids of all namespaces in deleting phase that have no connectors
		var namespaceIds []string
		if err := dbConn.Table("connector_namespaces").Select("connector_namespaces.id").
			Joins("LEFT JOIN connector_statuses ON connector_statuses.namespace_id = connector_namespaces.id AND "+
				"connector_statuses.deleted_at IS NULL").
			Group("connector_namespaces.id").
			Having("connector_namespaces.status_phase = ? AND "+
				"connector_namespaces.deleted_at IS NULL AND count(namespace_id) = 0", dbapi.ConnectorStatusPhaseDeleting).
			Find(&namespaceIds).Error; err != nil {
			return services.HandleGetError("Connector namespace",
				"status_phase", dbapi.ConnectorNamespacePhaseDeleting, err)
		}

		count = len(namespaceIds)
		if count == 0 {
			return nil
		}

		// delete all the empty deleting namespaces
		if err := dbConn.Where("id IN ?", namespaceIds).
			Delete(&dbapi.ConnectorNamespace{}).Error; err != nil {
			count = 0
			return services.HandleDeleteError("Connector namespace", "id", namespaceIds, err)
		}

		return nil

	}); err != nil {
		errs = append(errs, services.HandleDeleteError("Connector namespace", "status_phase",
			dbapi.ConnectorNamespacePhaseDeleting, err))
	}

	return count, errs
}
