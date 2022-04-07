package handlers

import (
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
	"github.com/gorilla/mux"
)

type supportedKafkaInstanceTypesHandler struct {
	service services.SupportedKafkaInstanceTypesService
}

func NewSupportedKafkaInstanceTypesHandler(service services.SupportedKafkaInstanceTypesService) *supportedKafkaInstanceTypesHandler {
	return &supportedKafkaInstanceTypesHandler{
		service: service,
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
				InstanceTypes: []public.SupportedKafkaInstanceType{},
			}

			regionInstanceTypeList, err := h.service.GetSupportedKafkaInstanceTypesByRegion(cloudProvider, cloudRegion)
			if err != nil {
				if err.IsInstanceTypeNotSupported() {
					logger.Logger.Error(err)
					return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to get supported Kafka instance types")
				}
				return nil, err
			}
			for _, instanceType := range regionInstanceTypeList {
				converted := presenters.PresentSupportedKafkaInstanceTypes(&instanceType)
				supportedKafkaInstanceTypeList.InstanceTypes = append(supportedKafkaInstanceTypeList.InstanceTypes, converted)
			}

			return supportedKafkaInstanceTypeList, nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}
