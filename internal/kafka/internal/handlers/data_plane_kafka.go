package handlers

import (
	v1 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/managedkafkas.managedkafka.bf2.org/v1"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/gorilla/mux"
)

type dataPlaneKafkaHandler struct {
	dataPlaneKafkaService services.DataPlaneKafkaService
	kafkaService          services.KafkaService
}

func NewDataPlaneKafkaHandler(dataPlaneKafkaService services.DataPlaneKafkaService, kafkaService services.KafkaService) *dataPlaneKafkaHandler {
	return &dataPlaneKafkaHandler{
		dataPlaneKafkaService: dataPlaneKafkaService,
		kafkaService:          kafkaService,
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
			err := h.dataPlaneKafkaService.UpdateDataPlaneKafkaService(ctx, clusterId, dataPlaneKafkaStatus)
			return nil, err
		},
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h *dataPlaneKafkaHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	clusterID := mux.Vars(r)["id"]
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.ValidateLength(&clusterID, "id", handlers.MinRequiredFieldLength, nil),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			managedKafkas, err := h.kafkaService.GetManagedKafkaByClusterID(clusterID)
			if err != nil {
				return nil, err
			}

			reservedManagedKafkas, err := h.kafkaService.GenerateReservedManagedKafkasByClusterID(clusterID)
			if err != nil {
				return nil, err
			}
			managedKafkas = append(managedKafkas, reservedManagedKafkas...)

			managedKafkaList := private.ManagedKafkaList{
				Kind:  "ManagedKafkaList",
				Items: []private.ManagedKafka{},
			}

			managedKafkaList.Items = append(
				managedKafkaList.Items,
				arrays.Map(managedKafkas, func(mk v1.ManagedKafka) private.ManagedKafka { return presenters.PresentManagedKafka(&mk) })...,
			)

			return managedKafkaList, nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}
