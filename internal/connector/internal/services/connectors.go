package services

import (
	"context"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/vault"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/queryparser"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/secrets"
	goerrors "github.com/pkg/errors"
	"github.com/spyzhov/ajson"
	"strings"

	"gorm.io/gorm"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

type ConnectorsService interface {
	Create(ctx context.Context, resource *dbapi.Connector) *errors.ServiceError
	Get(ctx context.Context, id string, tid string) (*dbapi.ConnectorWithConditions, *errors.ServiceError)
	List(ctx context.Context, kid string, listArgs *services.ListArguments, tid string) (dbapi.ConnectorWithConditionsList, *api.PagingMeta, *errors.ServiceError)
	Update(ctx context.Context, resource *dbapi.Connector) *errors.ServiceError
	SaveStatus(ctx context.Context, resource dbapi.ConnectorStatus) *errors.ServiceError
	Delete(ctx context.Context, id string) *errors.ServiceError
	ForEach(f func(*dbapi.Connector) *errors.ServiceError, query string, args ...interface{}) []error
	ForceDelete(ctx context.Context, id string) *errors.ServiceError
}

var _ ConnectorsService = &connectorsService{}

type connectorsService struct {
	connectionFactory     *db.ConnectionFactory
	bus                   signalbus.SignalBus
	vaultService          vault.VaultService
	connectorTypesService ConnectorTypesService
}

func NewConnectorsService(connectionFactory *db.ConnectionFactory, bus signalbus.SignalBus,
	vaultService vault.VaultService, connectorTypesService ConnectorTypesService) *connectorsService {
	return &connectorsService{
		connectionFactory:     connectionFactory,
		bus:                   bus,
		vaultService:          vaultService,
		connectorTypesService: connectorTypesService,
	}
}

// Create creates a connector in the database
func (k *connectorsService) Create(ctx context.Context, resource *dbapi.Connector) *errors.ServiceError {
	//kid := resource.KafkaID
	//if kid == "" {
	//	return errors.Validation("kafka id is undefined")
	//}

	dbConn := k.connectionFactory.New()
	if err := dbConn.Save(resource).Error; err != nil {
		return errors.GeneralError("failed to create connector: %v", err)
	}

	// read it back.... to get the updated version...
	if err := dbConn.Where("id = ?", resource.ID).First(&resource).Error; err != nil {
		return services.HandleGetError("Connector", "id", resource.ID, err)
	}

	resource.Status.ID = resource.ID
	resource.Status.Phase = dbapi.ConnectorStatusPhaseAssigning
	if err := dbConn.Save(&resource.Status).Error; err != nil {
		return errors.GeneralError("failed to save status: %v", err)
	}

	_ = db.AddPostCommitAction(ctx, func() {
		// Wake up the reconcile loop...
		k.bus.Notify("reconcile:connector")
	})

	return nil
}

// Get gets a connector by id from the database
func (k *connectorsService) Get(ctx context.Context, id string, tid string) (*dbapi.ConnectorWithConditions, *errors.ServiceError) {
	if id == "" {
		return nil, errors.Validation("connector id is undefined")
	}

	dbConn := k.connectionFactory.New()
	var resource dbapi.ConnectorWithConditions
	dbConn = dbConn.Model(&dbapi.Connector{})
	dbConn = selectConnectorWithConditions(dbConn)
	dbConn = dbConn.Where("connectors.id = ?", id)

	var err *errors.ServiceError
	admin, err := isAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if !admin {
		dbConn, err = filterConnectorsToOwnerOrOrg(ctx, dbConn, k.connectionFactory)
		if err != nil {
			return nil, err
		}
	}

	if tid != "" {
		dbConn = dbConn.Where("connectors.connector_type_id = ?", tid)
	}

	dbConn = dbConn.Limit(1)

	if err := dbConn.First(&resource).Error; err != nil {
		return nil, services.HandleGetError("Connector", "id", id, err)
	}
	return &resource, nil
}

func filterConnectorsToOwnerOrOrg(ctx context.Context, dbConn *gorm.DB, factory *db.ConnectionFactory) (*gorm.DB, *errors.ServiceError) {

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
		// unassigned connectors with no namespace_id use owner and org
		// assigned connectors use tenant user or organisation
		dbConn = dbConn.Where("(connectors.namespace_id is null AND (owner = ? OR organisation_id = ?)) OR (connectors.namespace_id IS NOT NULL AND connectors.namespace_id IN (?))",
			owner,
			orgId,
			factory.New().Table("connector_namespaces").Select("id").
				Where("deleted_at is null AND (tenant_user_id = ? OR tenant_organisation_id = ?)", owner, orgId))
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
	dbConn := k.connectionFactory.New()

	var resource dbapi.Connector
	if err := dbConn.Where("id = ?", id).First(&resource).Error; err != nil {
		return services.HandleGetError("Connector", "id", id, err)
	}
	if err := dbConn.Delete(&resource).Error; err != nil {
		return errors.GeneralError("unable to delete connector with id %s: %s", resource.ID, err)
	}

	// delete the associated relations
	if err := dbConn.Where("id = ?", id).Delete(&dbapi.ConnectorStatus{}).Error; err != nil {
		return services.HandleGetError("ConnectorStatus", "id", id, err)
	}

	_ = db.AddPostCommitAction(ctx, func() {
		// delete related distributed resources...

		if resource.ServiceAccount.ClientSecretRef != "" {
			err := k.vaultService.DeleteSecretString(resource.ServiceAccount.ClientSecretRef)
			if err != nil {
				logger.Logger.Errorf("failed to delete vault secret key '%s': %v", resource.ServiceAccount.ClientSecretRef, err)
			}
		}

		if len(resource.ConnectorSpec) != 0 {
			if ct, err := k.connectorTypesService.Get(resource.ConnectorTypeId); err == nil {
				_, _ = secrets.ModifySecrets(ct.JsonSchema, resource.ConnectorSpec, func(node *ajson.Node) error {
					if node.Type() != ajson.Object {
						return nil
					}
					ref, err := node.GetKey("ref")
					if err != nil {
						return nil
					}
					r, err := ref.GetString()
					if err != nil {
						return nil
					}
					err = k.vaultService.DeleteSecretString(r)
					if err != nil {
						logger.Logger.Errorf("failed to delete vault secret key '%s': %v", r, err)
					}
					return nil
				})
			}
		}
	})

	return nil
}

var validConnectorColumns = []string{"name", "owner", "kafka_id", "connector_type_id", "desired_state", "channel", "namespace_id"}

// List returns all connectors visible to the user within the requested paging window.
func (k *connectorsService) List(ctx context.Context, kafka_id string, listArgs *services.ListArguments, tid string) (dbapi.ConnectorWithConditionsList, *api.PagingMeta, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()
	pagingMeta := &api.PagingMeta{
		Page: listArgs.Page,
		Size: listArgs.Size,
	}

	if kafka_id != "" {
		dbConn = dbConn.Where("kafka_id = ?", kafka_id)
	}

	var err *errors.ServiceError
	admin, err := isAdmin(ctx)
	if err != nil {
		return nil, nil, err
	}
	if !admin {
		dbConn, err = filterConnectorsToOwnerOrOrg(ctx, dbConn, k.connectionFactory)
		if err != nil {
			return nil, nil, err
		}
	}

	if tid != "" {
		dbConn = dbConn.Where("connector_type_id = ?", tid)
	}

	// Apply search query
	if len(listArgs.Search) > 0 {
		queryParser := coreServices.NewQueryParser(validConnectorColumns...)
		searchDbQuery, err := queryParser.Parse(listArgs.Search)
		if err != nil {
			return nil, pagingMeta, errors.NewWithCause(errors.ErrorFailedToParseSearch, err, "Unable to list connector requests: %s", err.Error())
		}
		// add connectors. prefix to namespace_id
		searchDbQuery.Query = strings.ReplaceAll(searchDbQuery.Query, "namespace_id", "connectors.namespace_id")
		dbConn = dbConn.Where(searchDbQuery.Query, searchDbQuery.Values...)
	}

	// set total, limit and paging (based on https://gitlab.cee.redhat.com/service/api-guidelines#user-content-paging)
	total := int64(pagingMeta.Total)
	dbConn.Model(&dbapi.ConnectorList{}).Count(&total)
	pagingMeta.Total = int(total)
	if pagingMeta.Size > pagingMeta.Total {
		pagingMeta.Size = pagingMeta.Total
	}
	dbConn = dbConn.Offset((pagingMeta.Page - 1) * pagingMeta.Size).Limit(pagingMeta.Size)

	// default the order by name
	dbConn = dbConn.Order("name")

	// Set the order by arguments if any
	for _, orderByArg := range listArgs.OrderBy {
		dbConn = dbConn.Order(orderByArg)
	}

	var resourcesWithConditions dbapi.ConnectorWithConditionsList
	// execute query
	dbConn = selectConnectorWithConditions(dbConn)

	if err := dbConn.Find(&resourcesWithConditions).Error; err != nil {
		return resourcesWithConditions, pagingMeta, errors.GeneralError("Unable to list connectors: %s", err)
	}

	return resourcesWithConditions, pagingMeta, nil
}

func selectConnectorWithConditions(dbConn *gorm.DB) *gorm.DB {
	return dbConn.Select("connectors.*, connector_deployment_statuses.conditions").
		Joins("Status").
		Joins("left join connector_deployments on connector_deployments.connector_id = connectors.id " +
			"and connector_deployments.deleted_at IS NULL").
		Joins("left join connector_deployment_statuses on connector_deployment_statuses.id = connector_deployments.id " +
			"and connector_deployment_statuses.deleted_at IS NULL")
}

func (k connectorsService) Update(ctx context.Context, resource *dbapi.Connector) *errors.ServiceError {

	// If the version is set, then lets verify that the version has not changed...
	if resource.Version == 0 {
		return errors.BadRequest("resource version is required")
	}

	dbConn := k.connectionFactory.New()
	update := dbConn.Model(resource).Where("id = ? AND version = ?", resource.ID, resource.Version).Updates(resource)
	if err := update.Error; err != nil {
		return errors.GeneralError("failed to update: %s", err.Error())
	}
	if update.RowsAffected == 0 {
		return errors.BadRequest("resource version changed")
	}

	// read it back.... to get the updated version...
	dbConn = k.connectionFactory.New().Where("id = ?", resource.ID)
	if err := dbConn.First(&resource).Error; err != nil {
		return services.HandleGetError("Connector", "id", resource.ID, err)
	}

	_ = db.AddPostCommitAction(ctx, func() {
		// Wake up the reconcile loop...
		k.bus.Notify("reconcile:connector")
	})

	return nil
}

func (k connectorsService) SaveStatus(ctx context.Context, resource dbapi.ConnectorStatus) *errors.ServiceError {
	dbConn := k.connectionFactory.New()
	if err := dbConn.Model(resource).Save(resource).Error; err != nil {
		return errors.GeneralError("failed to update: %s", err.Error())
	}
	return nil
}

func (k connectorsService) ForEach(f func(*dbapi.Connector) *errors.ServiceError, query string, args ...interface{}) []error {
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
		return []error{errors.GeneralError("Unable to list connectors: %s", err)}
	}
	defer rows.Close()

	var errs []error
	for rows.Next() {
		resource := dbapi.Connector{}

		// ScanRows is a method of `gorm.DB`, it can be used to scan a row into a struct
		err := dbConn.ScanRows(rows, &resource)
		if err != nil {
			errs = append(errs, errors.GeneralError("Unable to scan connector: %s", err))
		}

		resource.Status.ID = resource.ID
		err = dbConn.Model(&dbapi.ConnectorStatus{}).First(&resource.Status).Error
		if err != nil {
			errs = append(errs, errors.GeneralError("Unable to load connector status: %s", err))
		}

		if serr := f(&resource); serr != nil {
			errs = append(errs, serr)
		}

	}
	return errs
}

func (k *connectorsService) ForceDelete(ctx context.Context, id string) *errors.ServiceError {
	if err := k.connectionFactory.New().Transaction(func(tx *gorm.DB) error {
		// delete deployment status, deployment, connector status and connector
		var deploymentId string
		if err := tx.Model(&dbapi.ConnectorDeployment{}).Where("connector_id = ?", id).
			Select("id").First(&deploymentId).Error; err != nil {
			if !services.IsRecordNotFoundError(err) {
				return services.HandleGetError("Connector deployment", "connector_id", id, err)
			}
		}
		if deploymentId != "" {
			if err := tx.Where("id = ?", deploymentId).
				Delete(&dbapi.ConnectorDeploymentStatus{}).Error; err != nil {
				return services.HandleDeleteError("Connector deployment status", "id", deploymentId, err)
			}
			if err := tx.Where("id = ?", deploymentId).
				Delete(&dbapi.ConnectorDeployment{}).Error; err != nil {
				return services.HandleDeleteError("Connector deployment", "id", deploymentId, err)
			}
		}
		if err := tx.Where("id = ?", id).
			Delete(&dbapi.ConnectorStatus{}).Error; err != nil {
			return services.HandleDeleteError("Connector status", "id", id, err)
		}
		if err := tx.Where("id = ?", id).
			Delete(&dbapi.Connector{}).Error; err != nil {
			return services.HandleDeleteError("Connector", "id", id, err)
		}

		return nil
	}); err != nil {
		return services.HandleDeleteError("Connector", "id", id, err)
	}
	return nil
}
