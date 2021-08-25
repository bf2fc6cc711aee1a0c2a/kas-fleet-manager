package handlers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/workers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/gorilla/mux"
	"net/http"
)

type ConnectorTypesHandler struct {
	service services.ConnectorTypesService
	manager *workers.ConnectorManager
}

var (
	maxConnectorTypeIdLength = 50
)

func NewConnectorTypesHandler(service services.ConnectorTypesService, manager *workers.ConnectorManager) *ConnectorTypesHandler {
	return &ConnectorTypesHandler{
		service: service,
		manager: manager,
	}
}

func (h ConnectorTypesHandler) Get(w http.ResponseWriter, r *http.Request) {
	// this API depends on the startup reconcile occurring so that all the connector types are
	// indexed in the DB
	h.manager.StartupReconcileWG.Wait()

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
			connectorType, err2 := presenters.PresentConnectorType(resource)
			if err2 != nil {
				return connectorType, errors.ToServiceError(err2)
			}
			return connectorType, nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

func (h ConnectorTypesHandler) List(w http.ResponseWriter, r *http.Request) {
	// this API depends on the startup reconcile occurring so that all the connector types are
	// indexed in the DB
	h.manager.StartupReconcileWG.Wait()

	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := coreServices.NewListArguments(r.URL.Query())
			resources, paging, err := h.service.List(ctx, listArgs)
			if err != nil {
				return nil, err
			}

			resourceList := public.ConnectorTypeList{
				Kind:  "ConnectorTypeList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
			}

			for _, resource := range resources {
				converted, err1 := presenters.PresentConnectorType(resource)
				if err1 != nil {
					return nil, errors.ToServiceError(err1)
				}
				resourceList.Items = append(resourceList.Items, *converted)
			}

			return resourceList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}
