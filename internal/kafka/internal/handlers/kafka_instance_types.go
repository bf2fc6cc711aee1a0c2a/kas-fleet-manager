package handlers

import (
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/gorilla/mux"
)

type supportedKafkaInstanceTypesHandler struct {
	service              services.SupportedKafkaInstanceTypesService
	cloudProviderService services.CloudProvidersService
	kafkaService         services.KafkaService
	kafkaConfig          *config.KafkaConfig
}

func NewSupportedKafkaInstanceTypesHandler(service services.SupportedKafkaInstanceTypesService, cloudProviderService services.CloudProvidersService, kafkaService services.KafkaService, kafkaConfig *config.KafkaConfig) *supportedKafkaInstanceTypesHandler {
	return &supportedKafkaInstanceTypesHandler{
		service:              service,
		cloudProviderService: cloudProviderService,
		kafkaService:         kafkaService,
		kafkaConfig:          kafkaConfig,
	}
}

func (h supportedKafkaInstanceTypesHandler) ListSupportedKafkaInstanceTypes(w http.ResponseWriter, r *http.Request) {
	cloudProvider := mux.Vars(r)["cloud_provider"]
	cloudRegion := mux.Vars(r)["cloud_region"]

	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.ValidateLength(&cloudProvider, "cloud_provider", &handlers.MinRequiredFieldLength, nil),
			handlers.ValidateLength(&cloudRegion, "cloud_region", &handlers.MinRequiredFieldLength, nil),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			supportedKafkaInstanceTypeList := public.SupportedKafkaInstanceTypesList{
				Kind:  "SupportedKafkaInstanceTypesList",
				Items: []public.SupportedKafkaInstanceType{},
			}

			regionInstanceTypeList, err := h.service.GetSupportedKafkaInstanceTypesByRegion(cloudProvider, cloudRegion)
			if err != nil {
				return nil, err
			}

			for _, regionInstanceType := range regionInstanceTypeList {
				converted := presenters.PresentSupportedKafkaInstanceType(&regionInstanceType)
				supportedKafkaInstanceTypeList.Items = append(supportedKafkaInstanceTypeList.Items, converted)
			}

			return supportedKafkaInstanceTypeList, nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}
