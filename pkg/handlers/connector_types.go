package handlers

import (
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httputil"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/presenters"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/private/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
)

type connectorTypesHandler struct {
	service services.ConnectorTypesService
}

func NewConnectorTypesHandler(service services.ConnectorTypesService) *connectorTypesHandler {
	return &connectorTypesHandler{
		service: service,
	}
}
func (h connectorTypesHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			resource, err := h.service.Get(id)
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
	id := mux.Vars(r)["id"]

	resource, err := h.service.Get(id)
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
