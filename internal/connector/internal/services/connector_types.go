package services

//
// Unlike most other service types that are backed by a DB, this service
// reports back information about the supported connector types and they
// become available when as supporting services are deployed to the control
// pane cluster.

import (
	"gorm.io/gorm"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	coreService "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/queryparser"
	"github.com/golang/glog"
)

type ConnectorTypesService interface {
	Get(id string) (*dbapi.ConnectorType, *errors.ServiceError)
	List(listArgs *services.ListArguments) (dbapi.ConnectorTypeList, *api.PagingMeta, *errors.ServiceError)
	ForEachConnectorCatalogEntry(f func(id string, channel string, ccc *config.ConnectorChannelConfig) *errors.ServiceError) *errors.ServiceError

	PutConnectorShardMetadata(ctc *dbapi.ConnectorShardMetadata) (int64, *errors.ServiceError)
	GetConnectorShardMetadata(id int64) (*dbapi.ConnectorShardMetadata, *errors.ServiceError)
	GetLatestConnectorShardMetadataID(tid, channel string) (int64, *errors.ServiceError)
	GetLatestConnectorShardMetadata(tid, channel string) (*dbapi.ConnectorShardMetadata, *errors.ServiceError)
	CatalogEntriesReconciled() (bool, *errors.ServiceError)
	DeleteUnusedAndNotInCatalog() *errors.ServiceError
	ListCatalogEntries(*coreService.ListArguments) ([]dbapi.ConnectorCatalogEntry, *api.PagingMeta, *errors.ServiceError)
	GetCatalogEntry(tyd string) (*dbapi.ConnectorCatalogEntry, *errors.ServiceError)
}

var _ ConnectorTypesService = &connectorTypesService{}

type connectorTypesService struct {
	connectorsConfig  *config.ConnectorsConfig
	connectionFactory *db.ConnectionFactory
}

func NewConnectorTypesService(connectorsConfig *config.ConnectorsConfig, connectionFactory *db.ConnectionFactory) *connectorTypesService {
	return &connectorTypesService{
		connectorsConfig:  connectorsConfig,
		connectionFactory: connectionFactory,
	}
}

// Create updates/inserts a connector type in the database
func (cts *connectorTypesService) Create(resource *dbapi.ConnectorType) *errors.ServiceError {
	tid := resource.ID
	if tid == "" {
		return errors.Validation("Connector type id is undefined")
	}

	dbConn := cts.connectionFactory.New()

	var oldResource dbapi.ConnectorType
	if err := dbConn.Select("id").Where("id = ?", tid).First(&oldResource).Error; err != nil {

		if services.IsRecordNotFoundError(err) {
			// We need to create the resource....
			if err := dbConn.Create(resource).Error; err != nil {
				return errors.GeneralError("failed to create connector type %q: %v", tid, err)
			}
		} else {
			return errors.NewWithCause(errors.ErrorGeneral, err, "Unable to find connector type")
		}

	} else {
		// remove old associations first
		if err := dbConn.Where("connector_type_id = ?", tid).Delete(&dbapi.ConnectorTypeLabel{}).Error; err != nil {
			return errors.GeneralError("failed to remove connector type labels %q: %v", tid, err)
		}
		if err := dbConn.Exec("DELETE FROM connector_type_channels WHERE connector_type_id = ?", tid).Error; err != nil {
			return errors.GeneralError("failed to remove connector type channels %q: %v", tid, err)
		}
		if err := dbConn.Where("connector_type_id = ?", tid).Delete(&dbapi.ConnectorTypeCapability{}).Error; err != nil {
			return errors.GeneralError("failed to remove connector type capabilities %q: %v", tid, err)
		}

		// update the existing connector type
		if err := dbConn.Session(&gorm.Session{FullSaveAssociations: true}).Updates(resource).Error; err != nil {
			return errors.GeneralError("failed to update connector type %q: %v", tid, err)
		}
	}

	// read it back.... to get the updated version...
	if err := dbConn.Where("id = ?", tid).
		Preload("Channels").Preload("Labels").Preload("Capabilities").
		First(&resource).Error; err != nil {
		return services.HandleGetError("Connector", "id", tid, err)
	}

	return nil
}

func (cts *connectorTypesService) Get(id string) (*dbapi.ConnectorType, *errors.ServiceError) {
	if id == "" {
		return nil, errors.Validation("TypeId is empty.")
	}

	var resource dbapi.ConnectorType
	dbConn := cts.connectionFactory.New()

	if err := dbConn.Unscoped().
		Preload("Channels").
		Preload("Labels").
		Preload("Capabilities").
		Where("connector_types.id = ?", id).
		First(&resource).Error; err != nil {
		return nil, services.HandleGetError(`Connector type`, `id`, id, err)
	}
	if resource.DeletedAt.Valid {
		return nil, services.HandleGoneError("Connector type", "id", id)
	}
	return &resource, nil
}

func GetValidConnectorTypeColumns() []string {
	return []string{"name", "description", "version", "label", "channel"}
}

// List returns all connector types
func (cts *connectorTypesService) List(listArgs *services.ListArguments) (dbapi.ConnectorTypeList, *api.PagingMeta, *errors.ServiceError) {
	if err := listArgs.Validate(GetValidConnectorTypeColumns()); err != nil {
		return nil, nil, errors.NewWithCause(errors.ErrorMalformedRequest, err, "Unable to list connector type requests: %s", err.Error())
	}

	//var resourceList dbapi.ConnectorTypeList
	var resourceList dbapi.ConnectorTypeList
	dbConn := cts.connectionFactory.New()
	pagingMeta := &api.PagingMeta{
		Page: listArgs.Page,
		Size: listArgs.Size,
	}

	// Apply search query
	if len(listArgs.Search) > 0 {
		queryParser := queryparser.NewQueryParser(GetValidConnectorTypeColumns()...)
		searchDbQuery, err := queryParser.Parse(listArgs.Search)
		if err != nil {
			return resourceList, pagingMeta, errors.NewWithCause(errors.ErrorFailedToParseSearch, err, "Unable to list connector type requests: %s", err.Error())
		}
		if strings.Contains(searchDbQuery.Query, "channel") {
			dbConn = dbConn.Joins("LEFT JOIN connector_type_channels channels on channels.connector_type_id = connector_types.id")
			searchDbQuery.Query = strings.ReplaceAll(searchDbQuery.Query, "channel", "channels.connector_channel_channel")
		}
		if strings.Contains(searchDbQuery.Query, "label") {
			dbConn = dbConn.Joins("LEFT JOIN connector_type_labels labels on labels.connector_type_id = connector_types.id")
			searchDbQuery.Query = strings.ReplaceAll(searchDbQuery.Query, "label", "labels.label")
		}
		dbConn = dbConn.Where(searchDbQuery.Query, searchDbQuery.Values...)
	}

	if len(listArgs.OrderBy) == 0 {
		// default orderBy name
		dbConn = dbConn.Order("name ASC")
	}

	// Set the order by arguments if any
	for _, orderByArg := range listArgs.OrderBy {
		dbConn = dbConn.Order(orderByArg)
	}

	// set total, limit and paging (based on https://gitlab.cee.redhat.com/service/api-guidelines#user-content-paging)
	total := int64(pagingMeta.Total)
	dbConn.Model(&resourceList).Count(&total)
	pagingMeta.Total = int(total)
	if pagingMeta.Size > pagingMeta.Total {
		pagingMeta.Size = pagingMeta.Total
	}
	dbConn = dbConn.Offset((pagingMeta.Page - 1) * pagingMeta.Size).Limit(pagingMeta.Size)

	// execute query
	result := dbConn.
		Preload("Channels").
		Preload("Labels").
		Preload("Capabilities").
		Find(&resourceList)
	if result.Error != nil {
		return nil, nil, errors.ToServiceError(result.Error)
	}

	return resourceList, pagingMeta, nil
}

func (cts *connectorTypesService) ForEachConnectorCatalogEntry(f func(id string, channel string, ccc *config.ConnectorChannelConfig) *errors.ServiceError) *errors.ServiceError {

	for _, entry := range cts.connectorsConfig.CatalogEntries {
		// create/update connector type
		connectorType, err := presenters.ConvertConnectorType(entry.ConnectorType)
		if err != nil {
			return errors.GeneralError("failed to convert connector type %s: %v", entry.ConnectorType.Id, err.Error())
		}
		if err := cts.Create(connectorType); err != nil {
			return err
		}

		// reconcile channels
		for channel, ccc := range entry.Channels {
			ccc := ccc
			err := f(entry.ConnectorType.Id, channel, &ccc)
			if err != nil {
				return err
			}
		}

		// update type checksum for latest catalog shard metadata
		dbConn := cts.connectionFactory.New()
		if err = dbConn.Model(connectorType).Where("id = ?", connectorType.ID).
			UpdateColumn("checksum", cts.connectorsConfig.CatalogChecksums[connectorType.ID]).Error; err != nil {
			return errors.GeneralError("failed to update connector type %s checksum: %v", entry.ConnectorType.Id, err.Error())
		}
	}
	return nil
}

func (cts *connectorTypesService) PutConnectorShardMetadata(ctc *dbapi.ConnectorShardMetadata) (int64, *errors.ServiceError) {

	var resource dbapi.ConnectorShardMetadata

	dbConn := cts.connectionFactory.New()
	dbConn = dbConn.Select("id")
	dbConn = dbConn.Where("connector_type_id = ?", ctc.ConnectorTypeId)
	dbConn = dbConn.Where("channel = ?", ctc.Channel)
	dbConn = dbConn.Where("shard_metadata = ?", ctc.ShardMetadata)

	if err := dbConn.First(&resource).Error; err != nil {
		if services.IsRecordNotFoundError(err) {

			// We need to create the resource....
			dbConn = cts.connectionFactory.New()
			if err := dbConn.Save(ctc).Error; err != nil {
				return 0, errors.GeneralError("failed to create connector type channel %q: %v", ctc.Channel, err)
			}

			// read it back again to get it's version.
			dbConn = cts.connectionFactory.New()
			dbConn = dbConn.Select("id")
			dbConn = dbConn.Where("connector_type_id = ?", ctc.ConnectorTypeId)
			dbConn = dbConn.Where("channel = ?", ctc.Channel)
			dbConn = dbConn.Where("shard_metadata = ?", ctc.ShardMetadata)
			if err := dbConn.First(&resource).Error; err != nil {
				return 0, errors.NewWithCause(errors.ErrorGeneral, err, "Unable to find connector type channel after insert")
			}

			// update the other records to know the latest_id
			dbConn = cts.connectionFactory.New()
			dbConn = dbConn.Table("connector_shard_metadata")
			dbConn = dbConn.Where("id <> ?", resource.ID)
			dbConn = dbConn.Where("connector_type_id = ?", ctc.ConnectorTypeId)
			dbConn = dbConn.Where("channel = ?", ctc.Channel)
			if err := dbConn.Update(`latest_id`, resource.ID).Error; err != nil {
				return 0, errors.GeneralError("failed to create connector type channel: %v", err)
			}

			return resource.ID, nil

		} else {
			return 0, errors.NewWithCause(errors.ErrorGeneral, err, "Unable to find connector type channel")
		}
	} else {
		// resource existed... update the ctc with the version it's been assigned in the DB...
		return resource.ID, nil
	}
}

func (cts *connectorTypesService) GetConnectorShardMetadata(id int64) (*dbapi.ConnectorShardMetadata, *errors.ServiceError) {
	resource := &dbapi.ConnectorShardMetadata{}
	dbConn := cts.connectionFactory.New()

	err := dbConn.
		Where("id", id).
		First(&resource).Error

	if err != nil {
		if services.IsRecordNotFoundError(err) {
			return nil, errors.NotFound("connector type channel not found")
		}
		return nil, errors.GeneralError("Unable to get connector type channel: %s", err)
	}
	return resource, nil
}

func (cts *connectorTypesService) GetLatestConnectorShardMetadataID(tid, channel string) (int64, *errors.ServiceError) {
	resource := &dbapi.ConnectorShardMetadata{}
	dbConn := cts.connectionFactory.New()

	err := dbConn.
		Select("id").
		Where("connector_type_id", tid).
		Where("channel", channel).
		Order("id desc").
		First(&resource).Error

	if err != nil {
		if services.IsRecordNotFoundError(err) {
			return 0, errors.NotFound("connector type channel not found")
		}
		return 0, errors.GeneralError("Unable to get connector type channel: %s", err)
	}
	return resource.ID, nil
}

func (cts *connectorTypesService) GetLatestConnectorShardMetadata(tid, channel string) (*dbapi.ConnectorShardMetadata, *errors.ServiceError) {
	resource := &dbapi.ConnectorShardMetadata{}
	dbConn := cts.connectionFactory.New()

	err := dbConn.Unscoped().
		Where("connector_type_id", tid).
		Where("channel", channel).
		First(&resource).Error

	if err != nil {
		if services.IsRecordNotFoundError(err) {
			return nil, errors.NotFound("connector type channel not found")
		}
		return nil, errors.GeneralError("Unable to get connector type channel: %s", err)
	}

	return resource, nil
}

func (cts *connectorTypesService) CatalogEntriesReconciled() (bool, *errors.ServiceError) {
	var typeIds []string
	catalogChecksums := cts.connectorsConfig.CatalogChecksums
	for id := range catalogChecksums {
		typeIds = append(typeIds, id)
	}

	var connectorTypes dbapi.ConnectorTypeList
	dbConn := cts.connectionFactory.New()
	if err := dbConn.Select("id, checksum").Where("id in ?", typeIds).
		Find(&connectorTypes).Error; err != nil {
		return false, services.HandleGetError("Connector type", "id", typeIds, err)
	}

	done := len(catalogChecksums) == len(connectorTypes)
	if done {
		for _, ct := range connectorTypes {
			if ct.Checksum == nil || *ct.Checksum != catalogChecksums[ct.ID] {
				done = false
			}
		}
	}
	return done, nil
}

func (cts *connectorTypesService) DeleteUnusedAndNotInCatalog() *errors.ServiceError {
	notToBeDeletedIDs := make([]string, len(cts.connectorsConfig.CatalogEntries))
	for _, entry := range cts.connectorsConfig.CatalogEntries {
		notToBeDeletedIDs = append(notToBeDeletedIDs, entry.ConnectorType.Id)
	}
	glog.V(5).Infof("Connector Type IDs in catalog not to be deleted: %v", notToBeDeletedIDs)

	var usedConnectorTypeIDs []string
	dbConn := cts.connectionFactory.New()
	if err := dbConn.Model(&dbapi.Connector{}).Distinct("connector_type_id").Find(&usedConnectorTypeIDs).Error; err != nil {
		return errors.GeneralError("failed to find active connectors: %v", err.Error())
	}
	glog.V(5).Infof("Connector Type IDs used by at least an active connector not to be deleted: %v", usedConnectorTypeIDs)

	notToBeDeletedIDs = append(notToBeDeletedIDs, usedConnectorTypeIDs...)

	if err := dbConn.Delete(&dbapi.ConnectorType{}, "id NOT IN ?", notToBeDeletedIDs).Error; err != nil {
		return errors.GeneralError("failed to delete connector type with ids %v : %v", notToBeDeletedIDs, err.Error())
	}
	glog.V(5).Infof("Deleted Connector Type with id NOT IN: %v", notToBeDeletedIDs)
	return nil
}

func (cts *connectorTypesService) ListCatalogEntries(listArgs *coreService.ListArguments) ([]dbapi.ConnectorCatalogEntry, *api.PagingMeta, *errors.ServiceError) {
	types, pagin, err := cts.List(listArgs)
	if err != nil {
		return nil, nil, err
	}

	entries := make([]dbapi.ConnectorCatalogEntry, len(types))

	for i := range types {
		ct := types[i]
		entry, err := cts.toConnectorCatalogEntry(ct)
		if err != nil {
			return nil, nil, err
		}

		entries[i] = *entry
	}

	return entries, pagin, nil
}

func (cts *connectorTypesService) GetCatalogEntry(id string) (*dbapi.ConnectorCatalogEntry, *errors.ServiceError) {
	ct, err := cts.Get(id)
	if err != nil {
		return nil, err
	}

	return cts.toConnectorCatalogEntry(ct)
}

func (cts *connectorTypesService) toConnectorCatalogEntry(ct *dbapi.ConnectorType) (*dbapi.ConnectorCatalogEntry, *errors.ServiceError) {
	entry := dbapi.ConnectorCatalogEntry{
		ConnectorType: ct,
		Channels:      make(map[string]*dbapi.ConnectorShardMetadata),
	}

	for _, c := range ct.Channels {
		meta, err := cts.GetLatestConnectorShardMetadata(ct.ID, c.Channel)
		if err != nil {
			return nil, err
		}

		entry.Channels[c.Channel] = meta
	}

	return &entry, nil
}
