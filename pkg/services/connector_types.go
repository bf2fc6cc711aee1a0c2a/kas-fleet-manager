package services

//
// Unlike most other service types that are backed by a DB, this service
// reports back information about the supported connector types and they
// become available when as supporting services are deployed to the control
// pane cluster.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/golang/glog"
)

//go:generate moq -out connector_types_moq.go . ConnectorTypesService
type ConnectorTypesService interface {
	Get(id string) (*api.ConnectorType, *errors.ServiceError)
	GetServiceAddress(id string) (string, *errors.ServiceError)
	List(ctx context.Context, listArgs *ListArguments) (api.ConnectorTypeList, *api.PagingMeta, *errors.ServiceError)
	DiscoverExtensions() error
}

var _ ConnectorTypesService = &connectorTypesService{}

type connectorTypesService struct {
	connectorTypeIndex  api.ConnectorTypeIndex
	serviceAddressIndex map[string]string
	mu                  sync.RWMutex
	connectorsConfig    *config.ConnectorsConfig
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

func (k *connectorTypesService) GetServiceAddress(id string) (string, *errors.ServiceError) {
	if id == "" {
		return "", errors.Validation("id is undefined")
	}

	k.mu.RLock()
	resource, found := k.serviceAddressIndex[id]
	k.mu.RUnlock()

	if !found {
		return "", errors.NotFound("ConnectorType with id='%s' not found", id)
	}
	return resource, nil
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

type ConnectorTypeCatalog struct {
	ConnectorTypeIds []string `json:"connector_type_ids"`
}

func getConnectorTypeCatalog(ctx context.Context, serviceUrl string) (*ConnectorTypeCatalog, error) {
	response := ConnectorTypeCatalog{}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serviceUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	c := &http.Client{}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			return nil, err
		}
		return &response, nil
	} else {
		return nil, fmt.Errorf("expected Content-Type: application/json, but got %s", contentType)
	}
}

func getConnectorType(ctx context.Context, serviceUrl string) (*openapi.ConnectorType, error) {
	response := openapi.ConnectorType{}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serviceUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	c := &http.Client{}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			return nil, err
		}
		return &response, nil
	} else {
		return nil, fmt.Errorf("expected Content-Type: application/json, but got %s", contentType)
	}
}

func (k *connectorTypesService) DiscoverExtensions() error {
	k.discoverExtensions(k.connectorsConfig.ConnectorTypeSvcUrls)
	return nil
}

func (k *connectorTypesService) discoverExtensions(urls []string) {
	ctx := context.Background()

	connectorTypeIndex := map[string]*api.ConnectorType{}
	serviceAddressIndex := map[string]string{}

	for _, u := range urls {
		xu, err := url.Parse(u)
		if err != nil {
			glog.Warningf("url parse failed: %v", err)
			continue
		}

		response, err := getConnectorTypeCatalog(ctx, u)
		if err != nil {
			glog.Warningf("failed to get %s, %v", u, err)
			continue
		}

		if len(response.ConnectorTypeIds) == 0 {
			glog.Warningf("no connector_type_ids defined in %v", u)
			continue
		}

		bxu := *xu
		bxu.Path = ""
		base := bxu.String()

		for _, id := range response.ConnectorTypeIds {
			x := fmt.Sprintf("%s/api/managed-services-api/v1/kafka-connector-types/%s", base, id)
			response, err := getConnectorType(ctx, x)
			if err != nil {
				glog.Warningf("failed to get %s, %v", x, err)
				continue
			}

			converted := presenters.ConvertConnectorType(*response)
			converted.ExtensionURL = xu

			converted.CreatedAt = time.Time{}
			converted.UpdatedAt = time.Time{}
			connectorTypeIndex[id] = converted
			serviceAddressIndex[id] = base
		}
	}

	k.mu.Lock()
	k.connectorTypeIndex = connectorTypeIndex
	k.serviceAddressIndex = serviceAddressIndex
	k.mu.Unlock()
}
