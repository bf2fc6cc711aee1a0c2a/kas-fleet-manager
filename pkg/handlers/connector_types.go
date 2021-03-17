package handlers

import (
	"net/http"
	"net/http/httputil"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

type connectorTypesHandler struct {
	service services.ConnectorTypesService
}

var (
	maxConnectorTypeIdLength = 50
)

func NewConnectorTypesHandler(service services.ConnectorTypesService) *connectorTypesHandler {
	return &connectorTypesHandler{
		service: service,
	}
}
func (h connectorTypesHandler) Get(w http.ResponseWriter, r *http.Request) {
	connectorTypeId := mux.Vars(r)["connector_type_id"]
	cfg := &handlerConfig{
		Validate: []validate{
			validation("connector_type_id", &connectorTypeId, minLen(1), maxLen(maxConnectorTypeIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			resource, err := h.service.Get(connectorTypeId)
			if err != nil {
				return nil, err
			}
			return presenters.PresentConnectorType(resource), nil
		},
		ErrorHandler: handleError,
	}
	handleGet(w, r, cfg)
}

func (h connectorTypesHandler) ProxyToExtensionService(w http.ResponseWriter, r *http.Request) {
	glog.Info("proxying to extension")
	connectorTypeId := mux.Vars(r)["connector_type_id"]

	err := validation("connector_type_id", &connectorTypeId, minLen(1), maxLen(maxConnectorTypeIdLength))()
	if err != nil {
		handleError(r.Context(), w, err)
		return
	}

	resource, err := h.service.Get(connectorTypeId)
	if err != nil {
		handleError(r.Context(), w, err)
		return
	}

	url := resource.ExtensionURL
	proxy := httputil.NewSingleHostReverseProxy(url)

	r.URL.Host = url.Host
	r.URL.Scheme = url.Scheme
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	r.Host = url.Host

	proxy.ServeHTTP(w, r)
}

func (h connectorTypesHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := services.NewListArguments(r.URL.Query())
			resources, paging, err := h.service.List(ctx, listArgs)
			if err != nil {
				return nil, err
			}

			resourceList := openapi.ConnectorTypeList{
				Kind:  "ConnectorTypeList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
			}

			for _, resource := range resources {
				converted := presenters.PresentConnectorType(resource)
				resourceList.Items = append(resourceList.Items, converted)
			}

			return resourceList, nil
		},
		ErrorHandler: handleError,
	}

	handleList(w, r, cfg)
}
