package handlers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/gorilla/mux"
)

type dataPlaneKafkaHandler struct {
	service      services.DataPlaneKafkaService
	kafkaService services.KafkaService
}

func NewDataPlaneKafkaHandler(service services.DataPlaneKafkaService, kafkaService services.KafkaService) *dataPlaneKafkaHandler {
	return &dataPlaneKafkaHandler{
		service:      service,
		kafkaService: kafkaService,
	}
}

func (h *dataPlaneKafkaHandler) UpdateKafkaStatuses(w http.ResponseWriter, r *http.Request) {
	clusterId := mux.Vars(r)["id"]
	var data = map[string]private.DataPlaneKafkaStatus{}

	cfg := &handlers.HandlerConfig{
		MarshalInto: &data,
		Validate:    []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			dataPlaneKafkaStatus := presenters.ConvertDataPlaneKafkaStatus(data)
			err := h.service.UpdateDataPlaneKafkaService(ctx, clusterId, dataPlaneKafkaStatus)
			return nil, err
		},
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h *dataPlaneKafkaHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	clusterID := mux.Vars(r)["id"]
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.ValidateLength(&clusterID, "id", &handlers.MinRequiredFieldLength, nil),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			managedKafkas, err := h.kafkaService.GetManagedKafkaByClusterID(clusterID)
			if err != nil {
				return nil, err
			}

			managedKafkaList := private.ManagedKafkaList{
				Kind:  "ManagedKafkaList",
				Items: []private.ManagedKafka{},
			}

			for _, mk := range managedKafkas {
				converted := presenters.PresentManagedKafka(&mk)
				managedKafkaList.Items = append(managedKafkaList.Items, converted)
			}
			return managedKafkaList, nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}
