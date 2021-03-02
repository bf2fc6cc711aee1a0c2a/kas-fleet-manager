package handlers

import (
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/gorilla/mux"
)

type dataPlaneClusterHandler struct {
	service services.DataPlaneClusterService
	config  services.ConfigService
}

func NewDataPlaneClusterHandler(service services.DataPlaneClusterService, configService services.ConfigService) *dataPlaneClusterHandler {
	return &dataPlaneClusterHandler{
		service: service,
		config:  configService,
	}
}

func (h *dataPlaneClusterHandler) UpdateDataPlaneClusterStatus(w http.ResponseWriter, r *http.Request) {
	dataPlaneClusterID := mux.Vars(r)["id"]

	var dataPlaneClusterUpdateRequest openapi.DataPlaneClusterUpdateStatusRequest

	cfg := &handlerConfig{
		MarshalInto: &dataPlaneClusterUpdateRequest,
		Validate: []validate{
			validateLength(&dataPlaneClusterID, "id", &minRequiredFieldLength, nil),
			h.validateBody(&dataPlaneClusterUpdateRequest),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			dataPlaneClusterStatus := presenters.ConvertDataPlaneClusterStatus(dataPlaneClusterUpdateRequest)
			err := h.service.UpdateDataPlaneClusterStatus(ctx, dataPlaneClusterID, dataPlaneClusterStatus)
			return nil, err
		},
		ErrorHandler: handleError,
	}

	// TODO do we always to return HTTP 204 No Content?
	handle(w, r, cfg, http.StatusNoContent)
}

func (h *dataPlaneClusterHandler) validateBody(request *openapi.DataPlaneClusterUpdateStatusRequest) validate {
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

		// TODO should some kind of validation of expected specific conditions be
		// made? if so, should be made at this level or only at the level directly
		// needed?

		return nil
	}
}

func (h *dataPlaneClusterHandler) validateNodeInfo(request *openapi.DataPlaneClusterUpdateStatusRequest) *errors.ServiceError {
	nodeInfo := &request.NodeInfo
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

	return nil
}

func (h *dataPlaneClusterHandler) validateResizeInfo(request *openapi.DataPlaneClusterUpdateStatusRequest) *errors.ServiceError {
	resizeInfo := &request.ResizeInfo

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
	if resizeInfo.Delta.IngressEgressThroughputPerSec == nil {
		return errors.FieldValidationError("resizeInfo delta ingressegress throughput per second must be set")
	}
	if resizeInfo.Delta.MaxPartitions == nil {
		return errors.FieldValidationError("resieInfo delta partitions must be set")
	}

	return nil
}

func (h *dataPlaneClusterHandler) validateTotal(request *openapi.DataPlaneClusterUpdateStatusRequest) *errors.ServiceError {
	total := &request.Total
	if total.Connections == nil {
		return errors.FieldValidationError("total connections must be set")
	}
	if total.DataRetentionSize == nil {
		return errors.FieldValidationError("total data retention size must be set")
	}
	if total.IngressEgressThroughputPerSec == nil {
		return errors.FieldValidationError("total ingressegress throughput per second must be set")
	}
	if total.Partitions == nil {
		return errors.FieldValidationError("total partitions must be set")
	}

	return nil
}

func (h *dataPlaneClusterHandler) validateRemaining(request *openapi.DataPlaneClusterUpdateStatusRequest) *errors.ServiceError {
	remaining := &request.Remaining

	if remaining.Connections == nil {
		return errors.FieldValidationError("remaining connections must be set")
	}
	if remaining.DataRetentionSize == nil {
		return errors.FieldValidationError("remaining data retention size must be set")
	}
	if remaining.IngressEgressThroughputPerSec == nil {
		return errors.FieldValidationError("remaining ingressegress throughput per second must be set")
	}
	if remaining.Partitions == nil {
		return errors.FieldValidationError("remaining partitions must be set")
	}

	return nil
}
