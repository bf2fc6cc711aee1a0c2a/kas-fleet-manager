package services

//
// Unlike most other service types that are backed by a DB, this service
// reports back information about the supported connector types and they
// become available when as supporting services are deployed to the control
// pane cluster.

import (
	"context"
	"net/url"
	"sort"
	"sync"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	catalog_api "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/connector_catalog/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
)

type ConnectorCatalogKey struct {
	Id      string `json:"id"`
	Channel string `json:"channel"`
}

type ConnectorCatalogEntry = catalog_api.ConnectorCatalogEntry

type ConnectorTypesService interface {
	Get(id string) (*api.ConnectorType, *errors.ServiceError)
	GetConnectorCatalogEntry(id string, channel string) (*ConnectorCatalogEntry, *errors.ServiceError)
	List(ctx context.Context, listArgs *ListArguments) (api.ConnectorTypeList, *api.PagingMeta, *errors.ServiceError)
	DiscoverExtensions() error
}

var _ ConnectorTypesService = &connectorTypesService{}

type connectorTypesService struct {
	connectorTypeIndex api.ConnectorTypeIndex
	connectorCatalog   map[ConnectorCatalogKey]ConnectorCatalogEntry
	mu                 sync.RWMutex
	connectorsConfig   *config.ConnectorsConfig
}

func NewConnectorTypesService(config *config.ConnectorsConfig) *connectorTypesService {
	return &connectorTypesService{
		connectorsConfig: config,
	}
}

func (k *connectorTypesService) Get(id string) (*api.ConnectorType, *errors.ServiceError) {
	if id == "" {
		return nil, errors.Validation("id is undefined")
	}

	k.mu.RLock()
	resource := k.connectorTypeIndex[id]
	k.mu.RUnlock()

	if resource == nil {
		return nil, errors.NotFound("ConnectorType with id='%s' not found", id)
	}
	return resource, nil
}

func (k *connectorTypesService) GetConnectorCatalogEntry(id string, channel string) (*ConnectorCatalogEntry, *errors.ServiceError) {
	if id == "" {
		return nil, errors.Validation("id is undefined")
	}

	k.mu.RLock()
	resource, found := k.connectorCatalog[ConnectorCatalogKey{
		Id:      id,
		Channel: channel,
	}]
	k.mu.RUnlock()

	if !found {
		return nil, errors.NotFound("ConnectorType with id='%s' not found", id)
	}
	return &resource, nil
}

// List returns all connector types
func (k *connectorTypesService) List(ctx context.Context, listArgs *ListArguments) (api.ConnectorTypeList, *api.PagingMeta, *errors.ServiceError) {
	resourceList := k.connectorTypesSortedByName()
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

func (k *connectorTypesService) connectorTypesSortedByName() api.ConnectorTypeList {
	k.mu.RLock()
	r := make(api.ConnectorTypeList, len(k.connectorTypeIndex))
	i := 0
	for _, v := range k.connectorTypeIndex {
		r[i] = v
		i++
	}
	k.mu.RUnlock()

	sort.Slice(r, func(i, j int) bool {
		return r[i].Name < r[j].Name
	})
	return r
}

func (k *connectorTypesService) DiscoverExtensions() error {
	k.discoverExtensions(k.connectorsConfig.ConnectorTypeSvcUrls)
	return nil
}

func (k *connectorTypesService) discoverExtensions(urls []string) {
	ctx := context.Background()

	connectorTypeIndex := map[string]*api.ConnectorType{}
	serviceAddressIndex := map[string]string{}
	connectorCatalog := map[ConnectorCatalogKey]ConnectorCatalogEntry{}

	for _, u := range urls {
		xu, err := url.Parse(u)
		if err != nil {
			logger.Logger.Warningf("url parse failed: %v", err)
			continue
		}

		config := catalog_api.NewConfiguration()
		config.BasePath = u
		client := catalog_api.NewAPIClient(config)

		catalog, _, err := client.DefaultApi.ListConnectorCatalog(ctx)
		if err != nil {
			logger.Logger.Warningf("failed to list catalog from: %s, %v", u, err)
			continue
		}

		for _, ct := range catalog {
			id := ct.Id

			response, _, err := client.DefaultApi.GetConnectorTypeByID(ctx, id)
			if err != nil {
				logger.Logger.Warningf("failed to get connector type %s, %v", id, err)
				continue
			}

			converted := presenters.ConvertConnectorType(response)
			converted.ExtensionURL = xu

			converted.CreatedAt = time.Time{}
			converted.UpdatedAt = time.Time{}
			if connectorTypeIndex[id] == nil {
				converted.Channels = []string{ct.Channel}
				connectorTypeIndex[id] = converted
				serviceAddressIndex[id] = u
				connectorCatalog[ConnectorCatalogKey{
					Id:      ct.Id,
					Channel: ct.Channel,
				}] = ct
			} else {
				if Contains(connectorTypeIndex[id].Channels, ct.Channel) {
					logger.Logger.Warningf("skipping duplicate type id '%s' and channel '%s' for connector type service found in: %s, keeping the one found in: %s", id, ct.Channel, u, serviceAddressIndex[id])
					continue
				}
				connectorTypeIndex[id].Channels = append(connectorTypeIndex[id].Channels, ct.Channel)
				connectorCatalog[ConnectorCatalogKey{
					Id:      ct.Id,
					Channel: ct.Channel,
				}] = ct
			}

		}
	}

	k.mu.Lock()
	k.connectorTypeIndex = connectorTypeIndex
	k.connectorCatalog = connectorCatalog
	k.mu.Unlock()
}
