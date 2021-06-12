package handlers

import (
	presenters2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/presenters"
	services2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/connector/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/gorilla/mux"
	"net/http"
)

type ConnectorTypesHandler struct {
	service services2.ConnectorTypesService
}

var (
	maxConnectorTypeIdLength = 50
)

func NewConnectorTypesHandler(service services2.ConnectorTypesService) *ConnectorTypesHandler {
	return &ConnectorTypesHandler{
		service: service,
	}
}
func (h ConnectorTypesHandler) Get(w http.ResponseWriter, r *http.Request) {
	connectorTypeId := mux.Vars(r)["connector_type_id"]
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_type_id", &connectorTypeId, handlers.MinLen(1), handlers.MaxLen(maxConnectorTypeIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			resource, err := h.service.Get(connectorTypeId)
			if err != nil {
				return nil, err
			}
			return presenters2.PresentConnectorType(resource), nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

func (h ConnectorTypesHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
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
				converted := presenters2.PresentConnectorType(resource)
				resourceList.Items = append(resourceList.Items, converted)
			}

			return resourceList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}
