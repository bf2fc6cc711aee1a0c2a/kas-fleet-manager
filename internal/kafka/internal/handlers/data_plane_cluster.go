package handlers

import (
	"fmt"
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/gorilla/mux"
)

type dataPlaneClusterHandler struct {
	dataplaneClusterService services.DataPlaneClusterService
}

func NewDataPlaneClusterHandler(dataplaneClusterService services.DataPlaneClusterService) *dataPlaneClusterHandler {
	return &dataPlaneClusterHandler{
		dataplaneClusterService: dataplaneClusterService,
	}
}

func (h *dataPlaneClusterHandler) UpdateDataPlaneClusterStatus(w http.ResponseWriter, r *http.Request) {
	dataPlaneClusterID := mux.Vars(r)["id"]

	var dataPlaneClusterUpdateRequest private.DataPlaneClusterUpdateStatusRequest

	cfg := &handlers.HandlerConfig{
		MarshalInto: &dataPlaneClusterUpdateRequest,
		Validate: []handlers.Validate{
			handlers.ValidateLength(&dataPlaneClusterID, "id", handlers.MinRequiredFieldLength, nil),
			h.validateBody(&dataPlaneClusterUpdateRequest),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			dataPlaneClusterStatus, err := presenters.ConvertDataPlaneClusterStatus(dataPlaneClusterUpdateRequest)
			if err != nil {
				return nil, errors.Validation(err.Error())
			}
			svcErr := h.dataplaneClusterService.UpdateDataPlaneClusterStatus(ctx, dataPlaneClusterID, dataPlaneClusterStatus)
			return nil, svcErr
		},
	}

	// TODO do we always to return HTTP 204 No Content?
	handlers.Handle(w, r, cfg, http.StatusNoContent)
}

func (h *dataPlaneClusterHandler) GetDataPlaneClusterConfig(w http.ResponseWriter, r *http.Request) {
	dataPlaneClusterID := mux.Vars(r)["id"]

	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.ValidateLength(&dataPlaneClusterID, "id", handlers.MinRequiredFieldLength, nil),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			dataClusterConfig, err := h.dataplaneClusterService.GetDataPlaneClusterConfig(ctx, dataPlaneClusterID)
			if err != nil {
				return nil, err
			}
			return presenters.PresentDataPlaneClusterConfig(dataClusterConfig), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (h *dataPlaneClusterHandler) validateBody(request *private.DataPlaneClusterUpdateStatusRequest) handlers.Validate {
	return func() *errors.ServiceError {
		return h.validateStrimzi(request)
	}
}

func (h *dataPlaneClusterHandler) validateStrimzi(request *private.DataPlaneClusterUpdateStatusRequest) *errors.ServiceError {
	for idx, strimziElem := range request.Strimzi {
		if strimziElem.Version == "" {
			return errors.FieldValidationError(fmt.Sprintf(".status.strimzi[%d].version cannot be empty", idx))
		}

		if len(strimziElem.KafkaVersions) == 0 {
			return errors.FieldValidationError(fmt.Sprintf(".status.strimzi[%d].kafkaVersions cannot be empty", idx))
		}

		if len(strimziElem.KafkaIbpVersions) == 0 {
			return errors.FieldValidationError(fmt.Sprintf(".status.strimzi[%d].kafkaIbpVersions cannot be empty", idx))
		}
	}

	return nil
}
