package handlers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/gorilla/mux"
	"net/http"
)

type managedKafkaHandler struct {
	service services.KafkaService
	config  services.ConfigService
}

func NewManagedKafkaHandler(service services.KafkaService, configService services.ConfigService) *managedKafkaHandler {
	return &managedKafkaHandler{
		service: service,
		config:  configService,
	}
}

func (h managedKafkaHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	clusterID := mux.Vars(r)["id"]
	cfg := &handlerConfig{
		Validate: []validate{
			validateLength(&clusterID, "id", &minRequiredFieldLength, nil),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			managedKafkas, err := h.service.GetManagedKafkaByClusterID(clusterID)
			if err != nil {
				return nil, err
			}

			managedKafkaList := openapi.ManagedKafkaList{
				Kind:  "ManagedKafkaList",
				Page:  1,
				Size:  1,
				Total: 1,
				Items: []openapi.ManagedKafka{},
			}

			for _, mk := range managedKafkas {
				converted := presenters.PresentManagedKafka(&mk)
				managedKafkaList.Items = append(managedKafkaList.Items, converted)
			}
			managedKafkaList.Size = int32(len(managedKafkaList.Items))
			managedKafkaList.Total = managedKafkaList.Size
			return managedKafkaList, nil
		},
		ErrorHandler: handleError,
	}

	handleGet(w, r, cfg)
}
