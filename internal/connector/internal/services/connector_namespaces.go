package services

import (
	"context"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/queryparser"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	"math/rand"
	"time"
)

type ConnectorNamespaceService interface {
	Create(ctx context.Context, request *dbapi.ConnectorNamespace) *errors.ServiceError
	Update(ctx context.Context, request *dbapi.ConnectorNamespace) *errors.ServiceError
	Get(ctx context.Context, namespaceID string) (*dbapi.ConnectorNamespace, *errors.ServiceError)
	List(ctx context.Context, clusterIDs []string, listArguments *services.ListArguments) (dbapi.ConnectorNamespaceList, *api.PagingMeta, *errors.ServiceError)
	Delete(ctx context.Context, namespaceName string) *errors.ServiceError
	SetEvalClusterId(request *dbapi.ConnectorNamespace) *errors.ServiceError
	GetOwnerClusterIds(userId string, ownerClusterIds *[]string) *errors.ServiceError
	GetOrgClusterIds(userId string, organisationId string, orgClusterIds *[]string) *errors.ServiceError
	CreateDefaultNamespace(ctx context.Context, connectorCluster *dbapi.ConnectorCluster) *errors.ServiceError
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
	// reload namespace to get version update
	if err := dbConn.Select("version").First(request).Error; err != nil {
		return errors.GeneralError("failed to read new connector namespace: %v", err)
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

	// TODO use signal bus and a namespace worker to sync connectors in updated namespace

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
	dbConn := k.connectionFactory.New()
	dbConn = dbConn.Where("id = ?", namespaceId)
	if err := dbConn.Delete(&dbapi.ConnectorNamespace{}).Error; err != nil {
		return errors.GeneralError("failed to delete connector namespace: %v", err)
	}

	// Clear the namespace from any connectors that were using it.
	// TODO do this asynchronously in a namespace reconciler
	dbConn = k.connectionFactory.New()
	{
		rows, err := dbConn.
			Model(&dbapi.Connector{}).
			Select("id").
			Where("namespace_id = ?", namespaceId).
			Rows()
		if err != nil {
			return errors.GeneralError("unable find connector using namespace %s: %s", namespaceId, err)
		}
		defer rows.Close()
		for rows.Next() {
			connector := dbapi.Connector{}
			err := dbConn.ScanRows(rows, &connector)
			if err != nil {
				return errors.GeneralError("Unable to scan connector: %s", err)
			}

			if err := dbConn.Model(&connector).Update("namespace_id", nil).Error; err != nil {
				return errors.GeneralError("failed to update connector: %s", err)
			}

			status := dbapi.ConnectorStatus{}
			status.ID = connector.ID
			status.NamespaceID = nil
			if err := dbConn.Model(&status).Update("phase", "assigning").Error; err != nil {
				return errors.GeneralError("failed to update connector status: %s", err)
			}
		}
	}

	// TODO: increment connector namespace metrics
	// metrics.IncreaseStatusCountMetric(constants.KafkaRequestStatusAccepted.String())
	return nil
}

// TODO make this a configurable property in the future
const defaultNamespaceName = "default-connector-namespace"

func (k *connectorNamespaceService) CreateDefaultNamespace(ctx context.Context, connectorCluster *dbapi.ConnectorCluster) *errors.ServiceError {
	owner := connectorCluster.Owner
	organisationId := connectorCluster.OrganisationId

	kind := presenters.UserKind
	if organisationId != "" {
		kind = presenters.OrganisationKind
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
		Kind: kind,
	}, owner, organisationId)

	if err != nil {
		return err
	}
	return k.Create(ctx, namespaceRequest)
}
