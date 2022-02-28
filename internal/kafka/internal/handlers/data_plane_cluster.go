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
	service services.DataPlaneClusterService
}

func NewDataPlaneClusterHandler(service services.DataPlaneClusterService) *dataPlaneClusterHandler {
	return &dataPlaneClusterHandler{
		service: service,
	}
}

func (h *dataPlaneClusterHandler) UpdateDataPlaneClusterStatus(w http.ResponseWriter, r *http.Request) {
	dataPlaneClusterID := mux.Vars(r)["id"]

	var dataPlaneClusterUpdateRequest private.DataPlaneClusterUpdateStatusRequest

	cfg := &handlers.HandlerConfig{
		MarshalInto: &dataPlaneClusterUpdateRequest,
		Validate: []handlers.Validate{
			handlers.ValidateLength(&dataPlaneClusterID, "id", &handlers.MinRequiredFieldLength, nil),
			h.validateBody(&dataPlaneClusterUpdateRequest),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			dataPlaneClusterStatus, err := presenters.ConvertDataPlaneClusterStatus(dataPlaneClusterUpdateRequest)
			if err != nil {
				return nil, errors.Validation(err.Error())
			}
			svcErr := h.service.UpdateDataPlaneClusterStatus(ctx, dataPlaneClusterID, dataPlaneClusterStatus)
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
			handlers.ValidateLength(&dataPlaneClusterID, "id", &handlers.MinRequiredFieldLength, nil),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			dataClusterConfig, err := h.service.GetDataPlaneClusterConfig(ctx, dataPlaneClusterID)
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
		err := h.validateNodeInfo(request)
		if err != nil {
			return err
		}

		err = h.validateTotal(request)
		if err != nil {
			return err
		}

		err = h.validateRemaining(request)
		if err != nil {
			return err
		}

		err = h.validateResizeInfo(request)
		if err != nil {
			return err
		}

		err = h.validateStrimzi(request)
		if err != nil {
			return err
		}

		// TODO should some kind of validation of expected specific conditions be
		// made? if so, should be made at this level or only at the level directly
		// needed?

		return nil
	}
}

func (h *dataPlaneClusterHandler) validateNodeInfo(request *private.DataPlaneClusterUpdateStatusRequest) *errors.ServiceError {
	if request.NodeInfo != nil {
		nodeInfo := *request.NodeInfo
		if nodeInfo.Ceiling == nil {
			return errors.FieldValidationError("nodeinfo ceiling attribute must be set")
		}
		if nodeInfo.Current == nil {
			return errors.FieldValidationError("nodeinfo current attribute must be set")
		}
		if nodeInfo.CurrentWorkLoadMinimum == nil {
			return errors.FieldValidationError("nodeinfo currentWorkLoadMinimum attribute must be set")
		}
		if nodeInfo.Floor == nil {
			return errors.FieldValidationError("nodeinfo floor attribute must be set")
		}
	}
	return nil
}

func (h *dataPlaneClusterHandler) validateResizeInfo(request *private.DataPlaneClusterUpdateStatusRequest) *errors.ServiceError {
	if request.ResizeInfo != nil {
		resizeInfo := *request.ResizeInfo

		if resizeInfo.NodeDelta == nil {
			return errors.FieldValidationError("resizeInfo nodeDelta attribute must be set")
		}
		if resizeInfo.Delta == nil {
			return errors.FieldValidationError("resizeInfo delta attribute must be set")
		}
		if resizeInfo.Delta.Connections == nil {
			return errors.FieldValidationError("resizeInfo delta connections must be set")
		}
		if resizeInfo.Delta.DataRetentionSize == nil {
			return errors.FieldValidationError("resizeInfo delta data retention size must be set")
		}
		if resizeInfo.Delta.IngressThroughputPerSec == nil {
			return errors.FieldValidationError("resizeInfo delta ingress throughput per second must be set")
		}
		if resizeInfo.Delta.EgressThroughputPerSec == nil {
			return errors.FieldValidationError("resizeInfo delta egress throughput per second must be set")
		}

		if resizeInfo.Delta.Partitions == nil {
			return errors.FieldValidationError("resieInfo delta partitions must be set")
		}
	}
	return nil
}

func (h *dataPlaneClusterHandler) validateTotal(request *private.DataPlaneClusterUpdateStatusRequest) *errors.ServiceError {
	total := request.Total
	if total.Connections == nil {
		return errors.FieldValidationError("total connections must be set")
	}
	if total.DataRetentionSize == nil {
		return errors.FieldValidationError("total data retention size must be set")
	}
	if total.IngressThroughputPerSec == nil {
		return errors.FieldValidationError("total ingress throughput per second must be set")
	}
	if total.EgressThroughputPerSec == nil {
		return errors.FieldValidationError("total egress throughput per second must be set")
	}
	if total.Partitions == nil {
		return errors.FieldValidationError("total partitions must be set")
	}

	return nil
}

func (h *dataPlaneClusterHandler) validateRemaining(request *private.DataPlaneClusterUpdateStatusRequest) *errors.ServiceError {
	remaining := request.Remaining

	if remaining.Connections == nil {
		return errors.FieldValidationError("remaining connections must be set")
	}
	if remaining.DataRetentionSize == nil {
		return errors.FieldValidationError("remaining data retention size must be set")
	}
	if remaining.IngressThroughputPerSec == nil {
		return errors.FieldValidationError("remaining ingress throughput per second must be set")
	}
	if remaining.EgressThroughputPerSec == nil {
		return errors.FieldValidationError("remaining egress throughput per second must be set")
	}
	if remaining.Partitions == nil {
		return errors.FieldValidationError("remaining partitions must be set")
	}

	return nil
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
