package handlers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/connector/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/gorilla/mux"
	"net/http"
)

type ConnectorTypesHandler struct {
	service services.ConnectorTypesService
}

var (
	maxConnectorTypeIdLength = 50
)

func NewConnectorTypesHandler(service services.ConnectorTypesService) *ConnectorTypesHandler {
	return &ConnectorTypesHandler{
		service: service,
	}
}
func (h ConnectorTypesHandler) Get(w http.ResponseWriter, r *http.Request) {
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
	}
	handleGet(w, r, cfg)
}

func (h ConnectorTypesHandler) List(w http.ResponseWriter, r *http.Request) {
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
	}

	handleList(w, r, cfg)
}
