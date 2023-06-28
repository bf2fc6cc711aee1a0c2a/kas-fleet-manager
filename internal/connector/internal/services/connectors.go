package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gorm/clause"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/vault"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/queryparser"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/secrets"
	goerrors "github.com/pkg/errors"
	"github.com/spyzhov/ajson"

	"gorm.io/gorm"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

type ConnectorsService interface {
	Create(ctx context.Context, resource *dbapi.Connector) *errors.ServiceError
	Get(ctx context.Context, id string) (*dbapi.ConnectorWithConditions, *errors.ServiceError)
	List(ctx context.Context, listArgs *services.ListArguments, clusterId string) (dbapi.ConnectorWithConditionsList, *api.PagingMeta, *errors.ServiceError)
	Update(ctx context.Context, resource *dbapi.Connector) *errors.ServiceError
	SaveStatus(ctx context.Context, resource dbapi.ConnectorStatus) *errors.ServiceError
	Delete(ctx context.Context, id string) *errors.ServiceError
	ForEach(f func(*dbapi.Connector) *errors.ServiceError, joins string, query string, args ...interface{}) []error
	ForceDelete(ctx context.Context, id string) *errors.ServiceError

	ResolveConnectorRefsWithBase64Secrets(resource *dbapi.Connector) (bool, *errors.ServiceError)
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
	if err := dbConn.Create(resource).Error; err != nil {
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
func (k *connectorsService) Get(ctx context.Context, id string) (*dbapi.ConnectorWithConditions, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()
	var resource dbapi.ConnectorWithConditions

	dbConn = selectConnectorWithConditions(dbConn, false)
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

	if err := dbConn.Unscoped().First(&resource).Error; err != nil {
		return nil, services.HandleGetError("Connector", "id", id, err)
	}
	if resource.DeletedAt.Valid {
		return nil, services.HandleGoneError("Connector", "id", id)
	}
	return &resource, nil
}

func filterConnectorsToOwnerOrOrg(ctx context.Context, dbConn *gorm.DB, factory *db.ConnectionFactory) (*gorm.DB, *errors.ServiceError) {

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

func GetValidConnectorColumns() []string {
	// state should be replaced with column name connector_statuses.phase
	return []string{"id", "created_at", "updated_at", "name", "owner", "organisation_id", "kafka_id", "connector_type_id", "desired_state", "state", "channel", "kafka_bootstrap_server", "service_account_client_id", "schema_registry_id", "schema_registry_url", "namespace_id"}
}

var columnRegex = regexp.MustCompile("^(" + strings.Join(GetValidConnectorColumns(), "|") + ")")

// List returns all connectors visible to the user within the requested paging window.
func (k *connectorsService) List(ctx context.Context, listArgs *services.ListArguments, clusterId string) (dbapi.ConnectorWithConditionsList, *api.PagingMeta, *errors.ServiceError) {
	if err := listArgs.Validate(GetValidConnectorColumns()); err != nil {
		return nil, nil, errors.NewWithCause(errors.ErrorMalformedRequest, err, "Unable to list connector requests: %s", err.Error())
	}

	dbConn := k.connectionFactory.New()
	pagingMeta := &api.PagingMeta{
		Page: listArgs.Page,
		Size: listArgs.Size,
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

	if clusterId != "" {
		dbConn = dbConn.Joins("JOIN connector_namespaces ON connectors.namespace_id = connector_namespaces.id and connector_namespaces.cluster_id = ?", clusterId)
	}

	joinedStatus := false
	// Apply search query
	if len(listArgs.Search) > 0 {
		queryParser := coreServices.NewQueryParserWithColumnPrefix("connectors", GetValidConnectorColumns()...)
		searchDbQuery, err := queryParser.Parse(listArgs.Search)
		if err != nil {
			return nil, pagingMeta, errors.NewWithCause(errors.ErrorFailedToParseSearch, err, "Unable to list connector requests: %s", err.Error())
		}
		if strings.Contains(searchDbQuery.Query, "connectors.state") {
			joinedStatus = true
			dbConn = dbConn.Joins("left join connector_statuses on connector_statuses.id = connectors.id")
			// replace connectors.state with connector_statuses.phase
			searchDbQuery.Query = strings.ReplaceAll(searchDbQuery.Query, "connectors.state", "connector_statuses.phase")
		}
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

	// Set the order by arguments if any
	if len(listArgs.OrderBy) == 0 {
		// default orderBy name
		dbConn = dbConn.Order("name ASC")
	} else {
		for _, orderByArg := range listArgs.OrderBy {
			// add connectors. prefix to all orderBy columns
			orderByArg = columnRegex.ReplaceAllString(orderByArg, "connectors.$1")
			if strings.Contains(orderByArg, "connectors.state") {
				if !joinedStatus {
					dbConn = dbConn.Joins("left join connector_statuses on connector_statuses.id = connectors.id")
				}
				orderByArg = strings.ReplaceAll(orderByArg, "connectors.state", "connector_statuses.phase")
			}
			dbConn = dbConn.Order(orderByArg)
		}
	}

	var resourcesWithConditions dbapi.ConnectorWithConditionsList
	// execute query
	dbConn = selectConnectorWithConditions(dbConn, joinedStatus)

	if err := dbConn.Find(&resourcesWithConditions).Error; err != nil {
		return resourcesWithConditions, pagingMeta, errors.GeneralError("unable to list connectors: %s", err)
	}

	return resourcesWithConditions, pagingMeta, nil
}

func selectConnectorWithConditions(dbConn *gorm.DB, joinedStatus bool) *gorm.DB {
	if !joinedStatus {
		dbConn = dbConn.Joins("Status")
	}
	return dbConn.Model(&dbapi.ConnectorWithConditions{}).Table("connectors").
		Select("connectors.*, connector_deployment_statuses.conditions").
		Preload(clause.Associations).
		Joins("left join connector_deployments on connector_deployments.connector_id = connectors.id " +
			"and connector_deployments.deleted_at IS NULL").
		Joins("left join connector_deployment_statuses on connector_deployment_statuses.id = connector_deployments.id " +
			"and connector_deployment_statuses.deleted_at IS NULL")
}

func (k *connectorsService) Update(ctx context.Context, resource *dbapi.Connector) *errors.ServiceError {

	if resource.Version == 0 {
		return errors.BadRequest("resource version is required")
	}

	if err := k.connectionFactory.New().Transaction(func(dbConn *gorm.DB) error {

		// remove old annotations
		if err := dbConn.Where("connector_id = ?", resource.ID).Delete(&dbapi.ConnectorAnnotation{}).Error; err != nil {
			return services.HandleUpdateError("Connector", err)
		}

		update := dbConn.Model(resource).Session(&gorm.Session{FullSaveAssociations: true}).
			Where("id = ? AND version = ?", resource.ID, resource.Version).Updates(resource)
		if err := update.Error; err != nil {
			return services.HandleUpdateError(`Connector`, err)
		}
		if update.RowsAffected == 0 {
			return errors.Conflict("resource version changed")
		}

		return nil

	}); err != nil {
		return errors.ToServiceError(err)
	}

	_ = db.AddPostCommitAction(ctx, func() {
		// Wake up the reconcile loop...
		k.bus.Notify("reconcile:connector")
	})

	// read it back.... to get the updated version...
	dbConn := k.connectionFactory.New().Preload(clause.Associations).Where("id = ?", resource.ID)
	if err := dbConn.First(&resource).Error; err != nil {
		return services.HandleGetError("Connector", "id", resource.ID, err)
	}

	return nil
}

func (k *connectorsService) SaveStatus(ctx context.Context, connectorStatus dbapi.ConnectorStatus) *errors.ServiceError {
	dbConn := k.connectionFactory.New()
	if err := dbConn.Where("deleted_at IS NULL").Save(&connectorStatus).Error; err != nil {
		return errors.GeneralError("failed to update: %s", err.Error())
	}
	return nil
}

func (k *connectorsService) ForEach(f func(*dbapi.Connector) *errors.ServiceError, joins string, query string, args ...interface{}) []error {
	dbConn := k.connectionFactory.New()

	if joins != "" {
		dbConn = dbConn.Joins(joins)
	}

	rows, err := dbConn.
		Model(&dbapi.Connector{}).
		Where(query, args...).
		Joins("left join connector_statuses on connector_statuses.id = connectors.id").
		Order("version").Rows()

	if err != nil {
		if goerrors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return []error{errors.GeneralError("unable to list connectors: %s", err)}
	}
	defer shared.CloseQuietly(rows)

	var errs []error
	for rows.Next() {
		resource := dbapi.Connector{}

		// ScanRows is a method of `gorm.DB`, it can be used to scan a row into a struct
		err := dbConn.ScanRows(rows, &resource)
		if err != nil {
			errs = append(errs, errors.GeneralError("unable to scan connector: %s", err))
		}

		resource.Status.ID = resource.ID
		err = dbConn.Model(&dbapi.ConnectorStatus{}).First(&resource.Status).Error
		if err != nil {
			errs = append(errs, errors.GeneralError("unable to load connector status: %s", err))
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
			if err := deleteConnectorDeployment(tx, deploymentId); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return services.HandleDeleteError("Connector", "id", id, err)
	}

	// delete connector in a separate transaction to allow deleting dangling deployments
	if err := k.Delete(ctx, id); err != nil {
		return err
	}
	return nil
}

func (k *connectorsService) ResolveConnectorRefsWithBase64Secrets(connector *dbapi.Connector) (bool, *errors.ServiceError) {
	err := getSecretsFromVaultAsBase64(connector, k.connectorTypesService, k.vaultService)
	if err != nil {
		return err.Code == errors.ErrorGeneral, err
	}

	return false, nil
}

func getSecretsFromVaultAsBase64(connector *dbapi.Connector, cts ConnectorTypesService, vault vault.VaultService) *errors.ServiceError {
	ct, err := cts.Get(connector.ConnectorTypeId)
	if err != nil {
		return errors.BadRequest("invalid connector type id: %s", connector.ConnectorTypeId)
	}
	// move secrets to a vault.

	if connector.ServiceAccount.ClientSecretRef != "" {
		v, err := vault.GetSecretString(connector.ServiceAccount.ClientSecretRef)
		if err != nil {
			return errors.GeneralError("could not get kafka client secrets from the vault: %v", err.Error())
		}
		encoded := base64.StdEncoding.EncodeToString([]byte(v))
		connector.ServiceAccount.ClientSecret = encoded
	}

	if len(connector.ConnectorSpec) != 0 {
		updated, err := secrets.ModifySecrets(ct.JsonSchema, connector.ConnectorSpec, func(node *ajson.Node) error {
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
		connector.ConnectorSpec = updated
	}
	return nil
}
