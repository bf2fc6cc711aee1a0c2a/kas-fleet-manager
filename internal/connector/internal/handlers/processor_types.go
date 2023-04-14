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

type ProcessorTypesHandler struct {
	service services.ProcessorTypesService
	manager *workers.ConnectorManager
}

var (
	maxProcessorTypeIdLength = 63
)

func NewProcessorTypesHandler(service services.ProcessorTypesService, manager *workers.ConnectorManager) *ProcessorTypesHandler {
	return &ProcessorTypesHandler{
		service: service,
		manager: manager,
	}
}

func (h ProcessorTypesHandler) Get(w http.ResponseWriter, r *http.Request) {
	processorTypeId := mux.Vars(r)["processor_type_id"]
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("processor_type_id", &processorTypeId, handlers.MinLen(1), handlers.MaxLen(maxProcessorTypeIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			resource, err := h.service.Get(processorTypeId)
			if err != nil {
				return nil, err
			}
			processorType, err2 := presenters.PresentProcessorType(resource)
			if err2 != nil {
				return processorType, errors.ToServiceError(err2)
			}
			return processorType, nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

func (h ProcessorTypesHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			listArgs := coreServices.NewListArguments(r.URL.Query())
			resources, paging, err := h.service.List(listArgs)
			if err != nil {
				return nil, err
			}

			resourceList := public.ProcessorTypeList{
				Kind:  "ProcessorTypeList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
			}

			for _, resource := range resources {
				converted, err1 := presenters.PresentProcessorType(resource)
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
