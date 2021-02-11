package handlers

import (
	"github.com/gorilla/mux"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/presenters"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
	"net/http"
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
