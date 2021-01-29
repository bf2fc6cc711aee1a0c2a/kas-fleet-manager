package handlers

import (
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/gorilla/mux"
)

type dataPlaneKafkaHandler struct {
	service services.DataPlaneKafkaService
	config  services.ConfigService
}

func NewDataPlaneKafkaHandler(service services.DataPlaneKafkaService, configService services.ConfigService) *dataPlaneKafkaHandler {
	return &dataPlaneKafkaHandler{
		service: service,
		config:  configService,
	}
}

func (h *dataPlaneKafkaHandler) UpdateKafkaStatuses(w http.ResponseWriter, r *http.Request) {
	clusterId := mux.Vars(r)["id"]
	var data = map[string]openapi.DataPlaneKafkaStatus{}

	cfg := &handlerConfig{
		MarshalInto: &data,
		Validate:    []validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			dataPlaneKafkaStatus := presenters.ConvertDataPlaneKafkaStatus(data)
			err := h.service.UpdateDataPlaneKafkaService(ctx, clusterId, dataPlaneKafkaStatus)
			return nil, err
		},
		ErrorHandler: handleError,
	}

	handle(w, r, cfg, http.StatusOK)
}
