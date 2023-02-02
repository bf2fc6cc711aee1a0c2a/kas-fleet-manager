package services

import (
	"context"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/vault"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/queryparser"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProcessorDeploymentsService interface {
	Create(ctx context.Context, resource *dbapi.ProcessorDeployment) *errors.ServiceError
	Get(ctx context.Context, id string) (*dbapi.ProcessorDeployment, *errors.ServiceError)
	GetByProcessorId(ctx context.Context, processorId string) (*dbapi.ProcessorDeployment, *errors.ServiceError)
	Update(ctx context.Context, resource *dbapi.ProcessorDeployment) *errors.ServiceError
	List(ctx context.Context, clusterId string, filterChannelUpdates bool, filterOperatorUpdates bool, includeDanglingDeploymentsOnly bool, listArgs *services.ListArguments, gtVersion int64) (dbapi.ProcessorDeploymentList, *api.PagingMeta, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	SaveStatus(ctx context.Context, status dbapi.ProcessorDeploymentStatus) *errors.ServiceError
}

var _ ProcessorDeploymentsService = &processorDeploymentsService{}

type processorDeploymentsService struct {
	connectionFactory *db.ConnectionFactory
	bus               signalbus.SignalBus
	vaultService      vault.VaultService
}

func NewProcessorDeploymentsService(connectionFactory *db.ConnectionFactory, bus signalbus.SignalBus,
	vaultService vault.VaultService) *processorDeploymentsService {
	return &processorDeploymentsService{
		connectionFactory: connectionFactory,
		bus:               bus,
		vaultService:      vaultService,
	}
}

// Create creates a processor in the database
func (k *processorDeploymentsService) Create(ctx context.Context, resource *dbapi.ProcessorDeployment) *errors.ServiceError {
	dbConn := k.connectionFactory.New()
	if err := dbConn.Create(resource).Error; err != nil {
		return errors.GeneralError("failed to create ProcessorDeployment: %v", err)
	}

	// read it back.... to get the updated version after triggers have run.
	if err := dbConn.Where("id = ?", resource.ID).First(&resource).Error; err != nil {
		return services.HandleGetError("ProcessorDeployment", "id", resource.ID, err)
	}

	if resource.ClusterID != "" {
		_ = db.AddPostCommitAction(ctx, func() {
			k.bus.Notify(fmt.Sprintf("/kafka_connector_clusters/%s/processor_deployments", resource.ClusterID))
		})
	}

	return nil
}

// Get gets a ProcessorDeployment by id from the database
func (k *processorDeploymentsService) Get(ctx context.Context, id string) (resource *dbapi.ProcessorDeployment, serr *errors.ServiceError) {
	dbConn := k.connectionFactory.New()
	if err := dbConn.Unscoped().Preload(clause.Associations).
		Joins("Status").Joins("ProcessorShardMetadata").Joins("Processor").
		Where("processor_deployments.id = ?", id).First(&resource).Error; err != nil {
		return resource, services.HandleGetError("Processor deployment", "id", id, err)
	}

	if resource.DeletedAt.Valid {
		return resource, services.HandleGoneError("Processor deployment", "id", id)
	}

	return
}

func (k *processorDeploymentsService) GetByProcessorId(ctx context.Context, processorID string) (resource *dbapi.ProcessorDeployment, serr *errors.ServiceError) {
	dbConn := k.connectionFactory.New()
	if err := dbConn.Unscoped().Preload(clause.Associations).
		Joins("Status").Joins("ProcessorShardMetadata").Joins("Processor").
		Where("processor_id = ?", processorID).First(&resource).Error; err != nil {
		return resource, services.HandleGetError("Processor deployment", "processor_id", processorID, err)
	}

	if resource.DeletedAt.Valid {
		return resource, services.HandleGoneError("Processor deployment", "processor_id", processorID)
	}

	return
}

// Update updates a processor deployment in the database.
func (k *processorDeploymentsService) Update(ctx context.Context, resource *dbapi.ProcessorDeployment) *errors.ServiceError {
	dbConn := k.connectionFactory.New()

	if err := dbConn.Save(resource).Error; err != nil {
		return services.HandleCreateError(`Processor deployment`, err)
	}

	if err := dbConn.Where("id = ?", resource.ID).Select("version").First(&resource).Error; err != nil {
		return services.HandleGetError(`Processor deployment`, "id", resource.ID, err)
	}

	if resource.ClusterID != "" {
		_ = db.AddPostCommitAction(ctx, func() {
			k.bus.Notify(fmt.Sprintf("/kafka_connector_clusters/%s/processor_deployments", resource.ClusterID))
		})
	}

	return nil
}

func getValidProcessorDeploymentColumns() []string {
	return []string{"processor_id", "processor_version", "cluster_id", "operator_id", "namespace_id"}
}

// List returns all processor deployments assigned to the cluster
func (k *processorDeploymentsService) List(ctx context.Context, clusterId string, filterChannelUpdates bool, filterOperatorUpdates bool, includeDanglingDeploymentsOnly bool, listArgs *services.ListArguments, gtVersion int64) (dbapi.ProcessorDeploymentList, *api.PagingMeta, *errors.ServiceError) {
	var resourceList dbapi.ProcessorDeploymentList
	dbConn := k.connectionFactory.New()
	// specify preload for annotations only, to avoid skipping deleted processors
	dbConn = dbConn.Preload("Annotations").Joins("Status").Joins("ProcessorShardMetadata").Joins("Processor")

	pagingMeta := &api.PagingMeta{
		Page: listArgs.Page,
		Size: listArgs.Size,
	}

	if clusterId != "" {
		dbConn = dbConn.Where("processor_deployments.cluster_id = ?", clusterId)
	}
	if gtVersion != 0 {
		dbConn = dbConn.Where("processor_deployments.version > ?", gtVersion)
	}
	if filterChannelUpdates {
		dbConn = dbConn.Where("\"ProcessorShardMetadata\".\"latest_revision\" IS NOT NULL")
	}
	if filterOperatorUpdates {
		dbConn = dbConn.Where("\"Status\".\"upgrade_available\"")
	}
	if includeDanglingDeploymentsOnly {
		dbConn = dbConn.Where("\"Processor\".\"deleted_at\" IS NOT NULL")
	} else {
		dbConn = dbConn.Where("\"Processor\".\"deleted_at\" IS NULL")
	}

	// Apply search query
	if len(listArgs.Search) > 0 {
		queryParser := coreServices.NewQueryParserWithColumnPrefix("processor_deployments", getValidProcessorDeploymentColumns()...)
		searchDbQuery, err := queryParser.Parse(listArgs.Search)
		if err != nil {
			return resourceList, pagingMeta, errors.NewWithCause(errors.ErrorFailedToParseSearch, err, "unable to list processor deployments requests: %s", err.Error())
		}
		dbConn = dbConn.Where(searchDbQuery.Query, searchDbQuery.Values...)
	}

	// set total, limit and paging (based on https://gitlab.cee.redhat.com/service/api-guidelines#user-content-paging)
	total := int64(pagingMeta.Total)
	dbConn.Session(&gorm.Session{}).Model(&resourceList).Count(&total)
	pagingMeta.Total = int(total)

	if pagingMeta.Size > pagingMeta.Total {
		pagingMeta.Size = pagingMeta.Total
	}
	dbConn = dbConn.Offset((pagingMeta.Page - 1) * pagingMeta.Size).Limit(pagingMeta.Size)

	// default the order by version
	dbConn = dbConn.Order("processor_deployments.version")

	// execute query
	if err := dbConn.Find(&resourceList).Error; err != nil {
		return resourceList, pagingMeta, services.HandleGetError("Processor deployment",
			fmt.Sprintf("filterChannelUpdates='%v' includeDanglingDeploymentsOnly=%v listArgs='%+v' cluster_id",
				filterChannelUpdates, includeDanglingDeploymentsOnly, listArgs), clusterId, err)
	}

	return resourceList, pagingMeta, nil
}

// Delete deletes a processor deployment from the database.
func (k *processorDeploymentsService) Delete(ctx context.Context, id string) *errors.ServiceError {
	dbConn := k.connectionFactory.New()
	// no err, deployment existed..
	if err := dbConn.Where("id = ?", id).Delete(&dbapi.ProcessorDeployment{}).Error; err != nil {
		err := services.HandleDeleteError("ProcessorDeployment", "id", id, err)
		if err != nil {
			return err
		}
	}
	if err := dbConn.Where("id = ?", id).Delete(&dbapi.ProcessorDeploymentStatus{}).Error; err != nil {
		err := services.HandleDeleteError("ProcessorDeploymentStatus", "id", id, err)
		if err != nil {
			return err
		}
	}
	return nil
}
func (k *processorDeploymentsService) SaveStatus(ctx context.Context, status dbapi.ProcessorDeploymentStatus) *errors.ServiceError {
	dbConn := k.connectionFactory.New()

	// let's get the processor id of the deployment...
	deployment := dbapi.ProcessorDeployment{}
	if err := dbConn.Unscoped().Select("processor_id", "deleted_at").
		Where("id = ?", status.ID).
		First(&deployment).Error; err != nil {
		return services.HandleGetError("Processor deployment", "id", status.ID, err)
	}
	if deployment.DeletedAt.Valid {
		return services.HandleGoneError("Processor deployment", "id", status.ID)
	}

	if err := dbConn.Model(&status).Where("id = ? and version <= ?", status.ID, status.Version).Save(&status).Error; err != nil {
		return errors.Conflict("failed to update processor deployment status: %s, probably a stale processor deployment status version was used: %d", err.Error(), status.Version)
	}

	processor := dbapi.Processor{}
	if err := dbConn.Select("desired_state").
		Where("id = ?", deployment.ProcessorID).
		First(&processor).Error; err != nil {
		return services.HandleGetError("Processor", "id", deployment.ProcessorID, err)
	}

	processorStatus := dbapi.ProcessorStatus{}
	if err := dbConn.Select("phase").
		Where("id = ?", deployment.ProcessorID).
		First(&processorStatus).Error; err != nil {
		return services.HandleGetError("Processor", "id", deployment.ProcessorID, err)
	}

	processorStatus.Phase = status.Phase
	if status.Phase == dbapi.ProcessorStatusPhaseDeleted {
		// we don't need the processor deployment anymore...
		if err := k.Delete(ctx, status.ID); err != nil {
			return err
		}
	}

	// update the processor status
	if err := dbConn.Where("id = ?", deployment.ProcessorID).Updates(&processorStatus).Error; err != nil {
		return services.HandleUpdateError("Processor status", err)
	}

	return nil
}
