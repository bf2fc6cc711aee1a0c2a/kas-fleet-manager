package services

import (
	"context"
	"encoding/base64"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/vault"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
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

//go:generate moq -out processors_moq.go . ProcessorsService
type ProcessorsService interface {
	//CRUD
	Create(ctx context.Context, resource *dbapi.Processor) *errors.ServiceError
	Get(ctx context.Context, id string) (*dbapi.ProcessorWithConditions, *errors.ServiceError)
	List(ctx context.Context, listArgs *services.ListArguments, clusterId string) (dbapi.ProcessorWithConditionsList, *api.PagingMeta, *errors.ServiceError)
	Update(ctx context.Context, resource *dbapi.Processor) *errors.ServiceError
	Delete(ctx context.Context, id string) *errors.ServiceError
	//Workers
	ForEach(f func(*dbapi.Processor) *errors.ServiceError, query string, args ...interface{}) []error
	SaveStatus(ctx context.Context, resource dbapi.ProcessorStatus) *errors.ServiceError
	//Admin
	ForceDelete(ctx context.Context, id string) *errors.ServiceError
	//Agent
	ResolveProcessorRefsWithBase64Secrets(resource *dbapi.Processor) (bool, *errors.ServiceError)
}

var _ ProcessorsService = &processorsService{}

type processorsService struct {
	connectionFactory           *db.ConnectionFactory
	bus                         signalbus.SignalBus
	vaultService                vault.VaultService
	processorDeploymentsService ProcessorDeploymentsService
}

func NewProcessorsService(connectionFactory *db.ConnectionFactory, bus signalbus.SignalBus,
	vaultService vault.VaultService, processorDeploymentsService ProcessorDeploymentsService) *processorsService {
	return &processorsService{
		connectionFactory:           connectionFactory,
		bus:                         bus,
		vaultService:                vaultService,
		processorDeploymentsService: processorDeploymentsService,
	}
}

// Create creates a processor in the database
func (p *processorsService) Create(ctx context.Context, resource *dbapi.Processor) *errors.ServiceError {
	dbConn := p.connectionFactory.New()
	if err := dbConn.Create(resource).Error; err != nil {
		return errors.GeneralError("failed to create processor: %v", err)
	}

	// read it back.... to get the updated version after triggers have run.
	if err := dbConn.Where("id = ?", resource.ID).First(&resource).Error; err != nil {
		return services.HandleGetError("Processor", "id", resource.ID, err)
	}

	resource.Status.ID = resource.ID
	resource.Status.Phase = dbapi.ProcessorStatusPhasePreparing
	if err := dbConn.Save(&resource.Status).Error; err != nil {
		return errors.GeneralError("failed to save status: %v", err)
	}

	_ = db.AddPostCommitAction(ctx, func() {
		// Wake up the reconcile loop...
		p.bus.Notify("reconcile:processor")
	})

	return nil
}

// Get gets a processor by id from the database
func (p *processorsService) Get(ctx context.Context, id string) (*dbapi.ProcessorWithConditions, *errors.ServiceError) {
	dbConn := p.connectionFactory.New()
	var resource dbapi.ProcessorWithConditions

	dbConn = selectProcessorWithConditions(dbConn, false)
	dbConn = dbConn.Where("processors.id = ?", id)

	var err *errors.ServiceError
	admin, err := isAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if !admin {
		dbConn, err = filterProcessorsToOwnerOrOrg(ctx, dbConn, p.connectionFactory)
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
	return []string{"id", "created_at", "updated_at", "name", "owner", "organisation_id", "processor_type_id", "desired_state", "state", "channel", "namespace_id", "kafka_id", "kafka_bootstrap_server", "service_account_client_id"}
}

// List returns all processors visible to the user within the requested paging window.
func (p *processorsService) List(ctx context.Context, listArgs *services.ListArguments, clusterId string) (dbapi.ProcessorWithConditionsList, *api.PagingMeta, *errors.ServiceError) {
	if err := listArgs.Validate(GetValidProcessorColumns()); err != nil {
		return nil, nil, errors.NewWithCause(errors.ErrorMalformedRequest, err, "Unable to list processor requests: %s", err.Error())
	}

	dbConn := p.connectionFactory.New()
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
		dbConn, err = filterProcessorsToOwnerOrOrg(ctx, dbConn, p.connectionFactory)
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
func (p *processorsService) Update(ctx context.Context, resource *dbapi.Processor) *errors.ServiceError {

	if resource.Version == 0 {
		return errors.BadRequest("resource version is required")
	}

	if err := p.connectionFactory.New().Transaction(func(dbConn *gorm.DB) error {

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
		p.bus.Notify("reconcile:processor")
	})

	// read it back.... to get the updated version...
	dbConn := p.connectionFactory.New().Preload(clause.Associations).Where("id = ?", resource.ID)
	if err := dbConn.First(&resource).Error; err != nil {
		return services.HandleGetError("Processor", "id", resource.ID, err)
	}

	return nil
}

// Delete deletes a processor from the database.
func (p *processorsService) Delete(ctx context.Context, id string) *errors.ServiceError {
	if id == "" {
		return errors.Validation("id is undefined")
	}
	dbConn := p.connectionFactory.New()

	var resource dbapi.Processor
	if err := dbConn.Where("id = ?", id).First(&resource).Error; err != nil {
		return services.HandleGetError("Processor", "id", id, err)
	}
	if err := dbConn.Delete(&resource).Error; err != nil {
		return errors.GeneralError("unable to delete processor with id %s: %s", resource.ID, err)
	}

	// delete the associated relations
	if err := dbConn.Where("id = ?", id).Delete(&dbapi.ProcessorStatus{}).Error; err != nil {
		return services.HandleGetError("ProcessorStatus", "id", id, err)
	}

	_ = db.AddPostCommitAction(ctx, func() {
		// delete related distributed resources...
		if resource.ServiceAccount.ClientSecretRef != "" {
			err := p.vaultService.DeleteSecretString(resource.ServiceAccount.ClientSecretRef)
			if err != nil {
				logger.Logger.Errorf("failed to delete vault secret key '%s': %v", resource.ServiceAccount.ClientSecretRef, err)
			}
		}
	})

	return nil
}

func (p *processorsService) ForEach(f func(*dbapi.Processor) *errors.ServiceError, query string, args ...interface{}) []error {
	dbConn := p.connectionFactory.New()
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

func (p *processorsService) SaveStatus(ctx context.Context, resource dbapi.ProcessorStatus) *errors.ServiceError {
	dbConn := p.connectionFactory.New()
	if err := dbConn.Model(resource).Save(resource).Error; err != nil {
		return errors.GeneralError("failed to update: %s", err.Error())
	}
	return nil
}

func (p *processorsService) ForceDelete(ctx context.Context, id string) *errors.ServiceError {
	if err := p.connectionFactory.New().Transaction(func(tx *gorm.DB) error {
		// delete deployment status, deployment, processor status and processor
		var deploymentId string
		if err := tx.Model(&dbapi.ProcessorDeployment{}).Where("processor_id = ?", id).
			Select("id").First(&deploymentId).Error; err != nil {
			if !services.IsRecordNotFoundError(err) {
				return services.HandleGetError("Processor deployment", "processor_id", id, err)
			}
		}
		if deploymentId != "" {
			if err := p.processorDeploymentsService.Delete(ctx, id); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return services.HandleDeleteError("Processor", "id", id, err)
	}

	// delete processor in a separate transaction to allow deleting dangling deployments
	if err := p.Delete(ctx, id); err != nil {
		return err
	}
	return nil
}

func (p *processorsService) ResolveProcessorRefsWithBase64Secrets(resource *dbapi.Processor) (bool, *errors.ServiceError) {
	err := getProcessorSecretsFromVaultAsBase64(resource, p.vaultService)
	if err != nil {
		return err.Code == errors.ErrorGeneral, err
	}

	return false, nil
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

func getProcessorSecretsFromVaultAsBase64(resource *dbapi.Processor, vault vault.VaultService) *errors.ServiceError {
	if resource.ServiceAccount.ClientSecretRef != "" {
		v, err := vault.GetSecretString(resource.ServiceAccount.ClientSecretRef)
		if err != nil {
			return errors.GeneralError("could not get kafka client secrets from the vault: %v", err.Error())
		}
		encoded := base64.StdEncoding.EncodeToString([]byte(v))
		resource.ServiceAccount.ClientSecret = encoded
	}
	return nil
}
