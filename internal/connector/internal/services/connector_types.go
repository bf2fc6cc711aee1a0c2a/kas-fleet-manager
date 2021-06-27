package services

//
// Unlike most other service types that are backed by a DB, this service
// reports back information about the supported connector types and they
// become available when as supporting services are deployed to the control
// pane cluster.

import (
	"context"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
)

type ConnectorCatalogKey struct {
	Id      string `json:"id"`
	Channel string `json:"channel"`
}

type ConnectorTypesService interface {
	Get(id string) (*dbapi.ConnectorType, *errors.ServiceError)
	GetConnectorCatalogEntry(id string, channel string) (*config.ConnectorChannelConfig, *errors.ServiceError)
	List(ctx context.Context, listArgs *services.ListArguments) (dbapi.ConnectorTypeList, *api.PagingMeta, *errors.ServiceError)
	ForEachConnectorCatalogEntry(f func(id string, channel string, ccc *config.ConnectorChannelConfig) *errors.ServiceError) *errors.ServiceError

	PutConnectorShardMetadata(ctc *dbapi.ConnectorShardMetadata) (int64, *errors.ServiceError)
	GetLatestConnectorShardMetadataID(tid, channel string) (int64, *errors.ServiceError)
	GetConnectorShardMetadata(id int64) (*dbapi.ConnectorShardMetadata, *errors.ServiceError)
}

var _ ConnectorTypesService = &connectorTypesService{}

type connectorTypesService struct {
	connectorTypeIndex map[string]*config.ConnectorCatalogEntry
	connectorsConfig   *config.ConnectorsConfig
	connectionFactory  *db.ConnectionFactory
}

func NewConnectorTypesService(connectorsConfig *config.ConnectorsConfig, connectionFactory *db.ConnectionFactory) *connectorTypesService {

	connectorTypeIndex := map[string]*config.ConnectorCatalogEntry{}
	for i := range connectorsConfig.CatalogEntries {
		entry := connectorsConfig.CatalogEntries[i]
		connectorTypeIndex[entry.ConnectorType.Id] = &entry
	}
	return &connectorTypesService{
		connectorTypeIndex: connectorTypeIndex,
		connectorsConfig:   connectorsConfig,
		connectionFactory:  connectionFactory,
	}
}

func (k *connectorTypesService) Get(id string) (*dbapi.ConnectorType, *errors.ServiceError) {
	if id == "" {
		return nil, errors.Validation("id is undefined")
	}

	resource := k.connectorTypeIndex[id]
	if resource == nil {
		return nil, errors.NotFound("ConnectorType with id='%s' not found", id)
	}

	return presenters.ConvertConnectorType(resource.ConnectorType), nil
}

func (k *connectorTypesService) GetConnectorCatalogEntry(id string, channel string) (*config.ConnectorChannelConfig, *errors.ServiceError) {
	if id == "" {
		return nil, errors.Validation("id is undefined")
	}

	resource := k.connectorTypeIndex[id]
	if resource == nil {
		return nil, errors.NotFound("ConnectorType with id='%s' not found", id)
	}

	channelConfig := resource.Channels[channel]
	if channelConfig == nil {
		return nil, errors.NotFound("ConnectorType with id='%s' does not have channel %s", id, channel)
	}

	return channelConfig, nil
}

// List returns all connector types
func (k *connectorTypesService) List(ctx context.Context, listArgs *services.ListArguments) (dbapi.ConnectorTypeList, *api.PagingMeta, *errors.ServiceError) {
	resourceList := k.getConnectorTypeList()
	pagingMeta := &api.PagingMeta{
		Page: listArgs.Page,
		Size: listArgs.Size,
	}

	// set total, limit and paging (based on https://gitlab.cee.redhat.com/service/api-guidelines#user-content-paging)
	pagingMeta.Total = len(resourceList)
	if pagingMeta.Size > pagingMeta.Total {
		pagingMeta.Size = pagingMeta.Total
	}
	offset := (pagingMeta.Page - 1) * pagingMeta.Size
	resourceList = resourceList[offset : offset+pagingMeta.Size]

	return resourceList, pagingMeta, nil
}

func (k *connectorTypesService) getConnectorTypeList() dbapi.ConnectorTypeList {
	r := make(dbapi.ConnectorTypeList, len(k.connectorsConfig.CatalogEntries))
	for i, v := range k.connectorsConfig.CatalogEntries {
		r[i] = presenters.ConvertConnectorType(v.ConnectorType)
	}
	return r
}

func (k *connectorTypesService) ForEachConnectorCatalogEntry(f func(id string, channel string, ccc *config.ConnectorChannelConfig) *errors.ServiceError) *errors.ServiceError {
	for id, entry := range k.connectorTypeIndex {
		for channel, ccc := range entry.Channels {
			err := f(id, channel, ccc)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (k *connectorTypesService) PutConnectorShardMetadata(ctc *dbapi.ConnectorShardMetadata) (int64, *errors.ServiceError) {

	var resource dbapi.ConnectorShardMetadata

	dbConn := k.connectionFactory.New()
	dbConn = dbConn.Select("id")
	dbConn = dbConn.Where("connector_type_id = ?", ctc.ConnectorTypeId)
	dbConn = dbConn.Where("channel = ?", ctc.Channel)
	dbConn = dbConn.Where("shard_metadata = ?", ctc.ShardMetadata)

	if err := dbConn.First(&resource).Error; err != nil {
		if services.IsRecordNotFoundError(err) {

			// We need to create the resource....
			dbConn = k.connectionFactory.New()
			if err := dbConn.Save(ctc).Error; err != nil {
				return 0, errors.GeneralError("failed to create connector type channel: %v", err)
			}

			// read it back again to get it's version.
			dbConn = k.connectionFactory.New()
			dbConn = dbConn.Select("id")
			dbConn = dbConn.Where("connector_type_id = ?", ctc.ConnectorTypeId)
			dbConn = dbConn.Where("channel = ?", ctc.Channel)
			dbConn = dbConn.Where("shard_metadata = ?", ctc.ShardMetadata)
			if err := dbConn.First(&resource).Error; err != nil {
				return 0, errors.NewWithCause(errors.ErrorGeneral, err, "Unable to find connector type channel after insert")
			}

			// update the other records to know the latest_id
			dbConn = k.connectionFactory.New()
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

func (k *connectorTypesService) GetLatestConnectorShardMetadataID(tid, channel string) (int64, *errors.ServiceError) {
	resource := &dbapi.ConnectorShardMetadata{}
	dbConn := k.connectionFactory.New()

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

func (k *connectorTypesService) GetConnectorShardMetadata(id int64) (*dbapi.ConnectorShardMetadata, *errors.ServiceError) {
	resource := &dbapi.ConnectorShardMetadata{}
	dbConn := k.connectionFactory.New()

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
