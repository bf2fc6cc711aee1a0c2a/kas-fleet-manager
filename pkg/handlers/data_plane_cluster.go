package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/presenters"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/private/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
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
