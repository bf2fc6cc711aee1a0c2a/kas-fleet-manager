package handlers

import (
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/gorilla/mux"
)

type kafkaPromoteHandler struct {
	service                      services.KafkaService
	kafkaConfig                  *config.KafkaConfig
	kafkaPromoteValidatorFactory KafkaPromoteValidatorFactory
}

func NewKafkaPromoteHandler(service services.KafkaService, kafkaConfig *config.KafkaConfig, kafkaPromoteValidatorFactory KafkaPromoteValidatorFactory) *kafkaPromoteHandler {
	return &kafkaPromoteHandler{
		service:                      service,
		kafkaConfig:                  kafkaConfig,
		kafkaPromoteValidatorFactory: kafkaPromoteValidatorFactory,
	}
}

const kafkaPromoteRouteVariableID = "id"

func (h kafkaPromoteHandler) Promote(w http.ResponseWriter, r *http.Request) {
	var kafkaPromoteRequest public.KafkaPromoteRequest
	id := mux.Vars(r)[kafkaPromoteRouteVariableID]
	ctx := r.Context()
	kafkaRequest, kafkaGetError := h.service.Get(ctx, id)
	validateKafkaFound := func() handlers.Validate {
		return func() *errors.ServiceError {
			return kafkaGetError
		}
	}
	cfg := &handlers.HandlerConfig{
		MarshalInto: &kafkaPromoteRequest,
		Validate: []handlers.Validate{
			handlers.ValidateAsyncEnabled(r, "promote a kafka request"),
			handlers.ValidateMinLength(&kafkaPromoteRequest.DesiredKafkaBillingModel, "desired_kafka_billing_model", 1),
			validateKafkaFound(),
			validateUserIsKafkaOwnerOrOrgAdmin(ctx, kafkaRequest),
			validateKafkaRequestToPromoteHasAPromotableActualKafkaBillingModel(kafkaRequest),
			validateKafkaRequestToPromoteHasAPromotableStatus(kafkaRequest),
			validateRequestedKafkaPromotionHasDifferentKafkaBillingModel(&kafkaPromoteRequest, kafkaRequest),
			validateNoKafkaPromotionInProgress(kafkaRequest),
			validateKafkaPromoteRequestFields(&kafkaPromoteRequest, kafkaRequest, h.service, h.kafkaConfig, h.kafkaPromoteValidatorFactory),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			kafkaFieldsToUpdate := map[string]interface{}{}
			kafkaFieldsToUpdate["desired_kafka_billing_model"] = kafkaPromoteRequest.DesiredKafkaBillingModel
			// We clear any potential promotion details being set due to a previous
			// Kafka promotion failure
			if kafkaRequest.PromotionDetails != "" {
				kafkaFieldsToUpdate["promotion_details"] = ""
			}
			kafkaFieldsToUpdate["promotion_status"] = dbapi.KafkaPromotionStatusPromoting

			kafkaFieldsToUpdate["marketplace"] = ""
			kafkaFieldsToUpdate["billing_cloud_account_id"] = ""
			// TODO improve not referencing a hardcoded string
			if kafkaPromoteRequest.DesiredKafkaBillingModel == "marketplace" {
				kafkaFieldsToUpdate["marketplace"] = kafkaPromoteRequest.DesiredMarketplace
				kafkaFieldsToUpdate["billing_cloud_account_id"] = kafkaPromoteRequest.DesiredBillingCloudAccountId
			}

			updateErr := h.service.Updates(kafkaRequest, kafkaFieldsToUpdate)
			return nil, updateErr
		},
	}

	handlers.Handle(w, r, cfg, http.StatusAccepted)
}
