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
	"github.com/golang/glog"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/presenters"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

//go:generate moq -out connector_types_moq.go . ConnectorTypesService
type ConnectorTypesService interface {
	Get(id string) (*api.ConnectorType, *errors.ServiceError)
	List(ctx context.Context, listArgs *ListArguments) (api.ConnectorTypeList, *api.PagingMeta, *errors.ServiceError)
	DiscoverExtensions() error
}

var _ ConnectorTypesService = &connectorTypesService{}

type connectorTypesService struct {
	connectorTypeIndex   api.ConnectorTypeIndex
	connectorTypeIndexMu sync.Mutex
	connectorsConfig     *config.ConnectorsConfig
	closeWatch           func()
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

	k.connectorTypeIndexMu.Lock()
	resource := k.connectorTypeIndex[id]
	k.connectorTypeIndexMu.Unlock()

	if resource == nil {
		return nil, errors.NotFound("ConnectorType with id='%s' not found", id)
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
	k.connectorTypeIndexMu.Lock()
	r := make(api.ConnectorTypeList, len(k.connectorTypeIndex))
	i := 0
	for _, v := range k.connectorTypeIndex {
		r[i] = v
		i++
	}
	k.connectorTypeIndexMu.Unlock()

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
	if k.closeWatch != nil {
		panic("DiscoverExtensions should only called once.")
	}
	closeWatch, err := k.connectorsConfig.WatchFiles(func(c config.ConnectorsConfig) {
		k.discoverExtensions(c.ConnectorTypeSvcUrls)
	})
	if err != nil {
		return err
	}
	// TODO: let callers close the open watch.
	k.closeWatch = closeWatch
	k.discoverExtensions(k.connectorsConfig.ConnectorTypeSvcUrls)
	return nil
}

func (k *connectorTypesService) discoverExtensions(urls []string) {
	ctx := context.Background()
	discoveredList := []*api.ConnectorType{}
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
			x := fmt.Sprintf("%s/api/managed-services-api/v1/connector-types/%s", base, id)
			response, err := getConnectorType(ctx, x)
			if err != nil {
				glog.Warningf("failed to get %s, %v", x, err)
				continue
			}

			converted := presenters.ConvertConnectorType(*response)
			converted.ExtensionURL = xu
			discoveredList = append(discoveredList, converted)
		}
	}

	k.connectorTypeIndexMu.Lock()
	defer k.connectorTypeIndexMu.Unlock()
	k.connectorTypeIndex = map[string]*api.ConnectorType{}
	for _, c := range discoveredList {
		c.CreatedAt = time.Time{}
		c.UpdatedAt = time.Time{}
		k.connectorTypeIndex[c.ID] = c
	}
}
