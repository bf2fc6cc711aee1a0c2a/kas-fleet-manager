package services

import (
	"context"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/vault"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/queryparser"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	goerrors "github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

type ProcessorsService interface {
	Create(ctx context.Context, resource *dbapi.Processor) *errors.ServiceError
	Get(ctx context.Context, id string) (*dbapi.ProcessorWithConditions, *errors.ServiceError)
	List(ctx context.Context, listArgs *services.ListArguments, clusterId string) (dbapi.ProcessorWithConditionsList, *api.PagingMeta, *errors.ServiceError)
	Update(ctx context.Context, resource *dbapi.Processor) *errors.ServiceError
	Delete(ctx context.Context, id string) *errors.ServiceError
	ForEach(f func(*dbapi.Processor) *errors.ServiceError, query string, args ...interface{}) []error
	ForceDelete(ctx context.Context, id string) *errors.ServiceError
	SaveStatus(ctx context.Context, resource dbapi.ProcessorStatus) *errors.ServiceError
}

var _ ProcessorsService = &processorsService{}

type processorsService struct {
	connectionFactory     *db.ConnectionFactory
	bus                   signalbus.SignalBus
	vaultService          vault.VaultService
	connectorTypesService ConnectorTypesService
}

func NewProcessorsService(connectionFactory *db.ConnectionFactory, bus signalbus.SignalBus,
	vaultService vault.VaultService, connectorTypesService ConnectorTypesService) *processorsService {
	return &processorsService{
		connectionFactory:     connectionFactory,
		bus:                   bus,
		vaultService:          vaultService,
		connectorTypesService: connectorTypesService,
	}
}

// Create creates a processor in the database
func (k *processorsService) Create(ctx context.Context, resource *dbapi.Processor) *errors.ServiceError {
	dbConn := k.connectionFactory.New()
	if err := dbConn.Create(resource).Error; err != nil {
		return errors.GeneralError("failed to create processor: %v", err)
	}

	// read it back.... to get the updated version after triggers have run.
	if err := dbConn.Where("id = ?", resource.ID).First(&resource).Error; err != nil {
		return services.HandleGetError("Processor", "id", resource.ID, err)
	}

	resource.Status.ID = resource.ID
	resource.Status.Phase = dbapi.ProcessorStatusPhaseAssigning
	if err := dbConn.Save(&resource.Status).Error; err != nil {
		return errors.GeneralError("failed to save status: %v", err)
	}

	_ = db.AddPostCommitAction(ctx, func() {
		// Wake up the reconcile loop...
		k.bus.Notify("reconcile:processor")
	})

	return nil
}

// Get gets a processor by id from the database
func (k *processorsService) Get(ctx context.Context, id string) (*dbapi.ProcessorWithConditions, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()
	var resource dbapi.ProcessorWithConditions

	dbConn = selectProcessorWithConditions(dbConn, false)
	dbConn = dbConn.Where("processors.id = ?", id)

	var err *errors.ServiceError
	admin, err := isAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if !admin {
		dbConn, err = filterProcessorsToOwnerOrOrg(ctx, dbConn, k.connectionFactory)
		if err != nil {
			return nil, err
		}
	}

	if err := dbConn.Unscoped().First(&resource).Error; err != nil {
		return nil, services.HandleGetError("Processor", "id", id, err)
	}
	if resource.DeletedAt.Valid {
		return nil, services.HandleGoneError("Processor", "id", id)
	}
	return &resource, nil
}

func GetValidProcessorColumns() []string {
	// state should be replaced with column name processor_statuses.phase
	return []string{"id", "created_at", "updated_at", "name", "owner", "organisation_id", "desired_state", "state", "service_account_client_id", "namespace_id"}
}

// List returns all processors visible to the user within the requested paging window.
func (k *processorsService) List(ctx context.Context, listArgs *services.ListArguments, clusterId string) (dbapi.ProcessorWithConditionsList, *api.PagingMeta, *errors.ServiceError) {
	if err := listArgs.Validate(GetValidProcessorColumns()); err != nil {
		return nil, nil, errors.NewWithCause(errors.ErrorMalformedRequest, err, "Unable to list processor requests: %s", err.Error())
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
		dbConn, err = filterProcessorsToOwnerOrOrg(ctx, dbConn, k.connectionFactory)
		if err != nil {
			return nil, nil, err
		}
	}

	if clusterId != "" {
		dbConn = dbConn.Joins("JOIN connector_namespaces ON processors.namespace_id = connector_namespaces.id and connector_namespaces.cluster_id = ?", clusterId)
	}

	joinedStatus := false
	// Apply search query
	if len(listArgs.Search) > 0 {
		queryParser := coreServices.NewQueryParserWithColumnPrefix("processors", GetValidProcessorColumns()...)
		searchDbQuery, err := queryParser.Parse(listArgs.Search)
		if err != nil {
			return nil, pagingMeta, errors.NewWithCause(errors.ErrorFailedToParseSearch, err, "Unable to list processor requests: %s", err.Error())
		}
		if strings.Contains(searchDbQuery.Query, "processors.state") {
			joinedStatus = true
			dbConn = dbConn.Joins("left join processor_statuses on processor_statuses.id = processors.id")
			// replace processors.state with processor_statuses.phase
			searchDbQuery.Query = strings.ReplaceAll(searchDbQuery.Query, "processors.state", "processor_statuses.phase")
		}
		dbConn = dbConn.Where(searchDbQuery.Query, searchDbQuery.Values...)
	}

	// set total, limit and paging (based on https://gitlab.cee.redhat.com/service/api-guidelines#user-content-paging)
	total := int64(pagingMeta.Total)
	dbConn.Model(&dbapi.ProcessorList{}).Count(&total)
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
			// add processors. prefix to all orderBy columns
			orderByArg = columnRegex.ReplaceAllString(orderByArg, "processors.$1")
			if strings.Contains(orderByArg, "processors.state") {
				if !joinedStatus {
					dbConn = dbConn.Joins("left join processor_statuses on processor_statuses.id = processors.id")
				}
				orderByArg = strings.ReplaceAll(orderByArg, "processors.state", "processor_statuses.phase")
			}
			dbConn = dbConn.Order(orderByArg)
		}
	}

	var resourcesWithConditions dbapi.ProcessorWithConditionsList
	// execute query
	dbConn = selectProcessorWithConditions(dbConn, joinedStatus)

	if err := dbConn.Find(&resourcesWithConditions).Error; err != nil {
		return resourcesWithConditions, pagingMeta, errors.GeneralError("unable to list processors: %s", err)
	}

	return resourcesWithConditions, pagingMeta, nil
}

// Update update a processor in the database.
func (k *processorsService) Update(ctx context.Context, resource *dbapi.Processor) *errors.ServiceError {

	if resource.Version == 0 {
		return errors.BadRequest("resource version is required")
	}

	if err := k.connectionFactory.New().Transaction(func(dbConn *gorm.DB) error {

		// remove old annotations
		if err := dbConn.Where("processor_id = ?", resource.ID).Delete(&dbapi.ProcessorAnnotation{}).Error; err != nil {
			return services.HandleUpdateError("Processor", err)
		}

		update := dbConn.Model(resource).Session(&gorm.Session{FullSaveAssociations: true}).
			Where("id = ? AND version = ?", resource.ID, resource.Version).Updates(resource)
		if err := update.Error; err != nil {
			return services.HandleUpdateError(`Processor`, err)
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
		k.bus.Notify("reconcile:processor")
	})

	// read it back.... to get the updated version...
	dbConn := k.connectionFactory.New().Preload(clause.Associations).Where("id = ?", resource.ID)
	if err := dbConn.First(&resource).Error; err != nil {
		return services.HandleGetError("Processor", "id", resource.ID, err)
	}

	return nil
}

// Delete deletes a processor from the database.
func (k *processorsService) Delete(ctx context.Context, id string) *errors.ServiceError {
	return errors.BadRequest("Not yet implemented.")
}

func (k *processorsService) SaveStatus(ctx context.Context, resource dbapi.ProcessorStatus) *errors.ServiceError {
	dbConn := k.connectionFactory.New()
	if err := dbConn.Model(resource).Save(resource).Error; err != nil {
		return errors.GeneralError("failed to update: %s", err.Error())
	}
	return nil
}

func (k *processorsService) ForEach(f func(*dbapi.Processor) *errors.ServiceError, query string, args ...interface{}) []error {
	dbConn := k.connectionFactory.New()
	rows, err := dbConn.
		Model(&dbapi.Processor{}).
		Where(query, args...).
		Joins("left join processor_statuses on processor_statuses.id = processors.id").
		Order("version").Rows()

	if err != nil {
		if goerrors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return []error{errors.GeneralError("unable to list processors: %s", err)}
	}
	defer shared.CloseQuietly(rows)

	var errs []error
	for rows.Next() {
		resource := dbapi.Processor{}

		// ScanRows is a method of `gorm.DB`, it can be used to scan a row into a struct
		err := dbConn.ScanRows(rows, &resource)
		if err != nil {
			errs = append(errs, errors.GeneralError("unable to scan processor: %s", err))
		}

		resource.Status.ID = resource.ID
		err = dbConn.Model(&dbapi.ProcessorStatus{}).First(&resource.Status).Error
		if err != nil {
			errs = append(errs, errors.GeneralError("unable to load processor status: %s", err))
		}

		if serr := f(&resource); serr != nil {
			errs = append(errs, serr)
		}

	}
	return errs
}

func (k *processorsService) ForceDelete(ctx context.Context, id string) *errors.ServiceError {
	if err := k.connectionFactory.New().Transaction(func(tx *gorm.DB) error {
		// delete deployment status, deployment, processor status and processor
		var deploymentId string
		if err := tx.Model(&dbapi.ProcessorDeployment{}).Where("processor_id = ?", id).
			Select("id").First(&deploymentId).Error; err != nil {
			if !services.IsRecordNotFoundError(err) {
				return services.HandleGetError("Processor deployment", "processor_id", id, err)
			}
		}
		if deploymentId != "" {
			if err := deleteProcessorDeployment(tx, deploymentId); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return services.HandleDeleteError("Processor", "id", id, err)
	}

	// delete processor in a separate transaction to allow deleting dangling deployments
	if err := k.Delete(ctx, id); err != nil {
		return err
	}
	return nil
}

func selectProcessorWithConditions(dbConn *gorm.DB, joinedStatus bool) *gorm.DB {
	if !joinedStatus {
		dbConn = dbConn.Joins("Status")
	}
	return dbConn.Model(&dbapi.ProcessorWithConditions{}).Table("processors").
		Select("processors.*, processor_deployment_statuses.conditions").
		Preload(clause.Associations).
		Joins("left join processor_deployments on processor_deployments.processor_id = processors.id " +
			"and processor_deployments.deleted_at IS NULL").
		Joins("left join processor_deployment_statuses on processor_deployment_statuses.id = processor_deployments.id " +
			"and processor_deployment_statuses.deleted_at IS NULL")
}

func filterProcessorsToOwnerOrOrg(ctx context.Context, dbConn *gorm.DB, factory *db.ConnectionFactory) (*gorm.DB, *errors.ServiceError) {

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
		// unassigned processors with no namespace_id use owner and org
		// assigned processors use tenant user or organisation
		dbConn = dbConn.Where("(processors.namespace_id is null AND (owner = ? OR organisation_id = ?)) OR (processors.namespace_id IS NOT NULL AND processors.namespace_id IN (?))",
			owner,
			orgId,
			factory.New().Table("connector_namespaces").Select("id").
				Where("deleted_at is null AND (tenant_user_id = ? OR tenant_organisation_id = ?)", owner, orgId))
	} else {
		dbConn = dbConn.Where("owner = ?", owner)
	}
	return dbConn, nil
}
