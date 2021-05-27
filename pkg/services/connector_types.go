package services

//
// Unlike most other service types that are backed by a DB, this service
// reports back information about the supported connector types and they
// become available when as supporting services are deployed to the control
// pane cluster.

import (
	"context"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

type ConnectorCatalogKey struct {
	Id      string `json:"id"`
	Channel string `json:"channel"`
}

type ConnectorTypesService interface {
	Get(id string) (*api.ConnectorType, *errors.ServiceError)
	GetConnectorCatalogEntry(id string, channel string) (*config.ConnectorChannelConfig, *errors.ServiceError)
	List(ctx context.Context, listArgs *ListArguments) (api.ConnectorTypeList, *api.PagingMeta, *errors.ServiceError)
}

var _ ConnectorTypesService = &connectorTypesService{}

type connectorTypesService struct {
	connectorTypeIndex map[string]*config.ConnectorCatalogEntry
	connectorsConfig   *config.ConnectorsConfig
}

func NewConnectorTypesService(connectorsConfig *config.ConnectorsConfig) *connectorTypesService {

	connectorTypeIndex := map[string]*config.ConnectorCatalogEntry{}
	for _, entry := range connectorsConfig.CatalogEntries {
		connectorTypeIndex[entry.ConnectorType.Id] = &entry
	}
	return &connectorTypesService{
		connectorTypeIndex: connectorTypeIndex,
		connectorsConfig:   connectorsConfig,
	}
}

func (k *connectorTypesService) Get(id string) (*api.ConnectorType, *errors.ServiceError) {
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
func (k *connectorTypesService) List(ctx context.Context, listArgs *ListArguments) (api.ConnectorTypeList, *api.PagingMeta, *errors.ServiceError) {
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

func (k *connectorTypesService) getConnectorTypeList() api.ConnectorTypeList {
	r := make(api.ConnectorTypeList, len(k.connectorsConfig.CatalogEntries))
	for i, v := range k.connectorsConfig.CatalogEntries {
		r[i] = presenters.ConvertConnectorType(v.ConnectorType)
	}
	return r
}
