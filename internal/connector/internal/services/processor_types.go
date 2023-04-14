package services

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/queryparser"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
	"github.com/golang/glog"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
)

//go:generate moq -out processor_types_moq.go . ProcessorTypesService
type ProcessorTypesService interface {
	//CR[UD]
	Create(resource *dbapi.ProcessorType) *errors.ServiceError
	Get(processorTypeId string) (*dbapi.ProcessorType, *errors.ServiceError)
	List(listArgs *services.ListArguments) (dbapi.ProcessorTypeList, *api.PagingMeta, *errors.ServiceError)
	//Workers
	CatalogEntriesReconciled() (bool, *errors.ServiceError)
	DeleteOrDeprecateRemovedTypes() *errors.ServiceError
	ForEachProcessorCatalogEntry(f func(processorTypeId string, channel string, ccc *config.ProcessorChannelConfig) *errors.ServiceError) *errors.ServiceError
	CleanupDeployments() *errors.ServiceError
	PutProcessorShardMetadata(*dbapi.ProcessorShardMetadata) (int64, *errors.ServiceError)
	GetLatestProcessorShardMetadata() (*dbapi.ProcessorShardMetadata, *errors.ServiceError)
}

var _ ProcessorTypesService = &processorTypesService{}

type processorTypesService struct {
	processorsConfig  *config.ProcessorsConfig
	connectionFactory *db.ConnectionFactory
}

func NewProcessorTypesService(processorsConfig *config.ProcessorsConfig, connectionFactory *db.ConnectionFactory) *processorTypesService {
	return &processorTypesService{
		processorsConfig:  processorsConfig,
		connectionFactory: connectionFactory,
	}
}

// Create updates/inserts a processor type in the database
func (p *processorTypesService) Create(resource *dbapi.ProcessorType) *errors.ServiceError {
	tid := resource.ID
	if shared.StringEmpty(tid) {
		return errors.Validation("processor type id is undefined")
	}

	// perform all type related DB updates in a single TX
	if err := p.connectionFactory.New().Transaction(func(dbConn *gorm.DB) error {

		var oldResource dbapi.ProcessorType
		if err := dbConn.Select("id").Preload("Annotations").
			Where("id = ?", tid).First(&oldResource).Error; err != nil {

			if services.IsRecordNotFoundError(err) {
				// We need to create the resource....
				if err := dbConn.Create(resource).Error; err != nil {
					return errors.GeneralError("failed to create processor type %q: %v", tid, err)
				}
			} else {
				return errors.NewWithCause(errors.ErrorGeneral, err, "unable to find processor type")
			}

		} else {
			// remove old associations first
			if err := dbConn.Where("processor_type_id = ?", tid).Delete(&dbapi.ProcessorTypeAnnotation{}).Error; err != nil {
				return errors.GeneralError("failed to remove processor type annotations %q: %v", tid, err)
			}
			if err := dbConn.Where("processor_type_id = ?", tid).Delete(&dbapi.ProcessorTypeLabel{}).Error; err != nil {
				return errors.GeneralError("failed to remove processor type labels %q: %v", tid, err)
			}
			if err := dbConn.Exec("DELETE FROM processor_type_channels WHERE processor_type_id = ?", tid).Error; err != nil {
				return errors.GeneralError("failed to remove processor type channels %q: %v", tid, err)
			}
			if err := dbConn.Where("processor_type_id = ?", tid).Delete(&dbapi.ProcessorTypeCapability{}).Error; err != nil {
				return errors.GeneralError("failed to remove processor type capabilities %q: %v", tid, err)
			}

			// update the existing processor type
			if err := dbConn.Session(&gorm.Session{FullSaveAssociations: true}).Updates(resource).Error; err != nil {
				return errors.GeneralError("failed to update processor type %q: %v", tid, err)
			}

			// update processor annotations
			if err := updateProcessorAnnotations(dbConn, oldResource); err != nil {
				return err
			}
		}

		// read it back.... to get the updated version...
		if err := dbConn.Where("id = ?", tid).
			Preload(clause.Associations).
			First(&resource).Error; err != nil {
			return services.HandleGetError("Processor type", "id", tid, err)
		}

		return nil
	}); err != nil {
		switch se := err.(type) {
		case *errors.ServiceError:
			return se
		default:
			return errors.GeneralError("failed to create/update processor type %q: %v", tid, se.Error())
		}
	}

	return nil
}

func updateProcessorAnnotations(dbConn *gorm.DB, oldResource dbapi.ProcessorType) error {
	// delete old type annotations copied to processors
	oldKeys := arrays.Map(oldResource.Annotations, func(ann dbapi.ProcessorTypeAnnotation) string {
		return ann.Key
	})
	tid := oldResource.ID
	if err := dbConn.Exec("DELETE FROM processor_annotations pa "+
		"USING processors p WHERE p.deleted_at IS NULL AND pa.processor_id = p.id AND "+
		"p.processor_type_id = ? AND pa.key IN ?", tid, oldKeys).Error; err != nil {
		return errors.GeneralError("failed to delete old processor annotations related to type %q: %v", tid, err)
	}
	// copy new type annotations to processors
	if err := dbConn.Exec("INSERT INTO processor_annotations (processor_id, key, value) "+
		"SELECT p.id, pa.key, pa.value FROM processor_type_annotations pa "+
		"JOIN processors p ON p.deleted_at IS NULL AND p.processor_type_id = pa.processor_type_id AND "+
		"pa.processor_type_id = ?", tid).Error; err != nil {
		return errors.GeneralError("failed to create new processor annotations related to type %q: %v", tid, err)
	}
	return nil
}

func (p *processorTypesService) Get(processorTypeId string) (*dbapi.ProcessorType, *errors.ServiceError) {
	if shared.StringEmpty(processorTypeId) {
		return nil, errors.Validation("processorTypeId is empty")
	}

	var resource dbapi.ProcessorType
	dbConn := p.connectionFactory.New()

	if err := dbConn.Unscoped().
		Preload(clause.Associations).
		Where("processor_types.id = ?", processorTypeId).
		First(&resource).Error; err != nil {
		return nil, services.HandleGetError(`ProcessorType`, `id`, processorTypeId, err)
	}
	if resource.DeletedAt.Valid {
		return nil, services.HandleGoneError("ProcessorType", "id", processorTypeId)
	}
	return &resource, nil
}

func GetValidProcessorTypeColumns() []string {
	return []string{"id", "created_at", "updated_at", "version", "name", "description", "label", "channel", "featured_rank", "pricing_tier", "deprecated"}
}

// List returns all processor types
func (p *processorTypesService) List(listArgs *services.ListArguments) (dbapi.ProcessorTypeList, *api.PagingMeta, *errors.ServiceError) {
	if err := listArgs.Validate(GetValidProcessorTypeColumns()); err != nil {
		return nil, nil, errors.NewWithCause(errors.ErrorMalformedRequest, err, "Unable to list ProcessorType requests: %s", err.Error())
	}

	var resourceList dbapi.ProcessorTypeList
	dbConn := p.connectionFactory.New()
	pagingMeta := &api.PagingMeta{
		Page: listArgs.Page,
		Size: listArgs.Size,
	}

	if len(p.processorsConfig.ProcessorsSupportedChannels) > 0 {
		dbConn = dbConn.Joins("LEFT JOIN processor_type_channels channels on channels.processor_type_id = processor_types.id")
		dbConn = dbConn.Where("channels.processor_channel_channel IN ?", p.processorsConfig.ProcessorsSupportedChannels)
		dbConn.Group("processor_types.id")
	}

	// Apply search query
	if len(listArgs.Search) > 0 {
		queryParser := queryparser.NewQueryParser(GetValidConnectorTypeColumns()...)
		searchDbQuery, err := queryParser.Parse(listArgs.Search)
		if err != nil {
			return resourceList, pagingMeta, errors.NewWithCause(errors.ErrorFailedToParseSearch, err, "Unable to list ProcessorType requests: %s", err.Error())
		}
		if strings.Contains(searchDbQuery.Query, "channel") {
			if len(p.processorsConfig.ProcessorsSupportedChannels) == 0 {
				dbConn = dbConn.Joins("LEFT JOIN processor_type_channels channels on channels.processor_type_id = processor_types.id")
			}
			searchDbQuery.Query = strings.ReplaceAll(searchDbQuery.Query, "channel", "channels.processor_channel_channel")
		}
		if strings.Contains(searchDbQuery.Query, "label") {
			if labelSetSearchClause.MatchString(searchDbQuery.Query) {
				dbConn = dbConn.Joins("LEFT JOIN (select processor_type_id, string_agg(label, ',' order by label) as label from processor_type_labels group by processor_type_id) as labels on labels.processor_type_id = processor_types.id")
			} else {
				dbConn = dbConn.Joins("LEFT JOIN processor_type_labels labels on labels.processor_type_id = processor_types.id")
			}
			searchDbQuery.Query = strings.ReplaceAll(searchDbQuery.Query, "label", "labels.label")
		}
		if strings.Contains(searchDbQuery.Query, "pricing_tier") {
			dbConn = dbConn.Joins("LEFT JOIN processor_type_annotations annotations on annotations.processor_type_id = processor_types.id")
			searchDbQuery.Query = strings.ReplaceAll(searchDbQuery.Query, "pricing_tier", "annotations.key = 'cos.bf2.org/pricing-tier' and annotations.value")
		}
		dbConn = dbConn.Where(searchDbQuery.Query, searchDbQuery.Values...)
	}

	if len(listArgs.OrderBy) == 0 {
		// default orderBy name
		dbConn = dbConn.Order("name ASC")
	}

	// Set the order by arguments if any
	for _, orderByArg := range listArgs.OrderBy {
		// ignore unsupported columns in orderBy
		if skipOrderByColumnsRegExp.MatchString(orderByArg) {
			continue
		}
		dbConn = dbConn.Order(orderByArg)
	}

	// set total, limit and paging (based on https://gitlab.cee.redhat.com/service/api-guidelines#user-content-paging)
	total := int64(pagingMeta.Total)
	p.connectionFactory.New().Table("(?) as types", dbConn.Model(&resourceList)).Count(&total)
	pagingMeta.Total = int(total)
	if pagingMeta.Size > pagingMeta.Total {
		pagingMeta.Size = pagingMeta.Total
	}
	dbConn = dbConn.Offset((pagingMeta.Page - 1) * pagingMeta.Size).Limit(pagingMeta.Size)

	// execute query
	result := dbConn.
		Preload(clause.Associations).
		Find(&resourceList)
	if result.Error != nil {
		return nil, nil, errors.ToServiceError(result.Error)
	}

	return resourceList, pagingMeta, nil
}

func (p *processorTypesService) CatalogEntriesReconciled() (bool, *errors.ServiceError) {
	var typeIds []string
	catalogChecksums := p.processorsConfig.CatalogChecksums
	for id := range catalogChecksums {
		typeIds = append(typeIds, id)
	}

	var processorTypes dbapi.ProcessorTypeList
	dbConn := p.connectionFactory.New()
	if err := dbConn.Select("id, checksum").Where("id in ?", typeIds).
		Find(&processorTypes).Error; err != nil {
		return false, services.HandleGetError("Processor type", "id", typeIds, err)
	}

	if len(catalogChecksums) != len(processorTypes) {
		return false, nil
	}
	for _, pt := range processorTypes {
		if pt.Checksum == nil || *pt.Checksum != catalogChecksums[pt.ID] {
			return false, nil
		}
	}
	return true, nil
}

func (p *processorTypesService) DeleteOrDeprecateRemovedTypes() *errors.ServiceError {
	notToBeDeletedIDs := make([]string, len(p.processorsConfig.CatalogEntries))
	for _, entry := range p.processorsConfig.CatalogEntries {
		notToBeDeletedIDs = append(notToBeDeletedIDs, entry.ProcessorType.Id)
	}
	glog.V(5).Infof("Processor Type IDs in catalog not to be deleted: %v", notToBeDeletedIDs)

	var usedProcessorTypeIDs []string
	dbConn := p.connectionFactory.New()
	if err := dbConn.Model(&dbapi.Processor{}).Distinct("processor_type_id").Find(&usedProcessorTypeIDs).Error; err != nil {
		return errors.GeneralError("failed to find active processors: %v", err.Error())
	}
	glog.V(5).Infof("Processor Type IDs used by at least one active processor not to be deleted: %v", usedProcessorTypeIDs)

	// flag deprecated types
	deprecatedIds := getDeprecatedTypes(usedProcessorTypeIDs, notToBeDeletedIDs)
	if len(deprecatedIds) > 0 {
		if err := dbConn.Model(&dbapi.ProcessorType{}).
			Where("id IN ?", deprecatedIds).
			Update("deprecated", true).Error; err != nil {
			return errors.GeneralError("failed to deprecate connector types with ids %v : %v", deprecatedIds, err.Error())
		}
		glog.V(5).Infof("Deprecated Connector Types with id IN: %v", deprecatedIds)
	}

	notToBeDeletedIDs = append(notToBeDeletedIDs, usedProcessorTypeIDs...)

	if err := dbConn.Delete(&dbapi.ProcessorType{}, "id NOT IN ?", notToBeDeletedIDs).Error; err != nil {
		return errors.GeneralError("failed to delete processor type with ids %v : %v", notToBeDeletedIDs, err.Error())
	}
	glog.V(5).Infof("Deleted Processor Type with id NOT IN: %v", notToBeDeletedIDs)
	return nil
}

func (p *processorTypesService) ForEachProcessorCatalogEntry(f func(processorTypeId string, channel string, ccc *config.ProcessorChannelConfig) *errors.ServiceError) *errors.ServiceError {
	for _, entry := range p.processorsConfig.CatalogEntries {
		// create/update processor type
		processorType, err := presenters.ConvertProcessorType(entry.ProcessorType)
		if err != nil {
			return errors.GeneralError("failed to convert processor type %s: %v", entry.ProcessorType.Id, err.Error())
		}
		if err := p.Create(processorType); err != nil {
			return err
		}

		// reconcile channels
		for channel, ccc := range entry.Channels {
			ccc := ccc
			err := f(entry.ProcessorType.Id, channel, &ccc)
			if err != nil {
				return err
			}
		}

		// update type checksum for latest catalog shard metadata
		dbConn := p.connectionFactory.New()
		if err = dbConn.Model(processorType).Where("id = ?", processorType.ID).
			UpdateColumn("checksum", p.processorsConfig.CatalogChecksums[processorType.ID]).Error; err != nil {
			return errors.GeneralError("failed to update processor type %s checksum: %v", entry.ProcessorType.Id, err.Error())
		}
	}
	return nil
}

func (p *processorTypesService) CleanupDeployments() *errors.ServiceError {
	type Result struct {
		DeploymentID string
	}

	// Find ProcessorDeployments that have not been deleted but whose Processor has been deleted...
	results := []Result{}
	dbConn := p.connectionFactory.New()
	err := dbConn.Table("processors").
		Select("processor_deployments.id AS deployment_id").
		Joins("JOIN processor_deployments ON processors.id = processor_deployments.processor_id").
		Where("processors.deleted_at IS NOT NULL").
		Where("processor_deployments.deleted_at IS NULL").
		Scan(&results).Error

	if err != nil {
		return errors.GeneralError("Unable to list Processor Deployments whose Processors have been deleted: %s", err)
	}

	for _, result := range results {
		// delete those deployments and associated status
		if err := dbConn.Delete(&dbapi.ProcessorDeploymentStatus{
			Model: db.Model{
				ID: result.DeploymentID,
			},
		}).Error; err != nil {
			return errors.GeneralError("Unable to delete Processor Deployment status whose Processors have been deleted: %s", err)
		}
		if err := dbConn.Delete(&dbapi.ProcessorDeployment{
			Model: db.Model{
				ID: result.DeploymentID,
			},
		}).Error; err != nil {
			return errors.GeneralError("Unable to delete Processor Deployment whose Processors have been deleted: %s", err)
		}
	}

	return nil
}

func (p *processorTypesService) PutProcessorShardMetadata(processorShardMetadata *dbapi.ProcessorShardMetadata) (int64, *errors.ServiceError) {
	// This is an over simplified implementation to ensure we store a ProcessorShardMetadata record
	dbConn := p.connectionFactory.New()
	if err := dbConn.Save(processorShardMetadata).Error; err != nil {
		return 0, errors.GeneralError("failed to create processor shard metadata %v: %v", processorShardMetadata, err)
	}
	return processorShardMetadata.ID, nil
}

func (p *processorTypesService) GetLatestProcessorShardMetadata() (*dbapi.ProcessorShardMetadata, *errors.ServiceError) {
	resource := &dbapi.ProcessorShardMetadata{}
	dbConn := p.connectionFactory.New()

	err := dbConn.
		Where(dbapi.ProcessorShardMetadata{}).
		Order("revision desc").
		First(&resource).Error

	if err != nil {
		if services.IsRecordNotFoundError(err) {
			return nil, errors.NotFound("processor type shard metadata not found")
		}
		return nil, errors.GeneralError("unable to get processor shard metadata: %s", err)
	}
	return resource, nil
}
