package handlers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/golang-jwt/jwt/v4"
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	config "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/authorization"

	"github.com/gorilla/mux"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
)

type kafkaHandler struct {
	service        services.KafkaService
	providerConfig *config.ProviderConfig
	authService    authorization.Authorization
	kafkaConfig    *config.KafkaConfig
}

func NewKafkaHandler(service services.KafkaService, providerConfig *config.ProviderConfig, authService authorization.Authorization, kafkaConfig *config.KafkaConfig) *kafkaHandler {
	return &kafkaHandler{
		service:        service,
		providerConfig: providerConfig,
		authService:    authService,
		kafkaConfig:    kafkaConfig,
	}
}

func (h kafkaHandler) Create(w http.ResponseWriter, r *http.Request) {
	var kafkaRequestPayload public.KafkaRequestPayload
	ctx := r.Context()

	handlerContext := NewValidationContext()

	cfg := &handlers.HandlerConfig{
		MarshalInto: &kafkaRequestPayload,
		Validate: []handlers.Validate{
			handlers.ValidateAsyncEnabled(r, "creating kafka requests"),
			handlers.ValidateLength(&kafkaRequestPayload.Name, "name", handlers.MinRequiredFieldLength, &MaxKafkaNameLength),
			ValidKafkaClusterName(&kafkaRequestPayload.Name, "name"),
			ValidateKafkaClusterNameIsUnique(&kafkaRequestPayload.Name, h.service, r.Context()),
			ValidateKafkaClaims(handlerContext, ctx),
			ValidateCloudProvider(handlerContext, ctx, &h.service, &kafkaRequestPayload, h.providerConfig, "creating kafka requests"),
			ValidateKafkaPlan(handlerContext, ctx, &h.service, h.kafkaConfig, &kafkaRequestPayload),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			convKafka := presenters.ConvertKafkaRequest(kafkaRequestPayload)

			claims := handlerContext[CLAIMS].(jwt.MapClaims)
			convKafka.Owner = auth.GetUsernameFromClaims(claims)
			convKafka.OrganisationId = auth.GetOrgIdFromClaims(claims)
			convKafka.OwnerAccountId = auth.GetAccountIdFromClaims(claims)
			convKafka.SizeId = handlerContext.GetString(KAFKA_SIZE_ID)
			convKafka.InstanceType = handlerContext.GetString(KAFKA_INSTANCE_TYPE)
			convKafka.CloudProvider = handlerContext.GetString(CLOUD_PROVIDER)
			convKafka.Region = handlerContext.GetString(REGION)

			svcErr := h.service.RegisterKafkaJob(convKafka)
			if svcErr != nil {
				return nil, svcErr
			}
			return presenters.PresentKafkaRequest(convKafka, h.kafkaConfig)
		},
	}

	// return 202 status accepted
	handlers.Handle(w, r, cfg, http.StatusAccepted)
}

func (h kafkaHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			kafkaRequest, err := h.service.Get(ctx, id)
			if err != nil {
				return nil, err
			}
			return presenters.PresentKafkaRequest(kafkaRequest, h.kafkaConfig)
		},
	}
	handlers.HandleGet(w, r, cfg)
}

// Delete is the handler for deleting a kafka request
func (h kafkaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.ValidateAsyncEnabled(r, "deleting kafka requests"),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()

			err := h.service.RegisterKafkaDeprovisionJob(ctx, id)
			return nil, err
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusAccepted)
}

func (h kafkaHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := coreServices.NewListArguments(r.URL.Query())

			if err := listArgs.Validate(); err != nil {
				return nil, errors.NewWithCause(errors.ErrorMalformedRequest, err, "Unable to list kafka requests: %s", err.Error())
			}

			kafkaRequests, paging, err := h.service.List(ctx, listArgs)
			if err != nil {
				return nil, err
			}

			kafkaRequestList := public.KafkaRequestList{
				Kind:  "KafkaRequestList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []public.KafkaRequest{},
			}

			for _, kafkaRequest := range kafkaRequests {
				converted, err := presenters.PresentKafkaRequest(kafkaRequest, h.kafkaConfig)
				if err != nil {
					return public.KafkaRequestList{}, err
				}
				kafkaRequestList.Items = append(kafkaRequestList.Items, converted)
			}

			return kafkaRequestList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

// Update is the handler for updating a kafka request
func (h kafkaHandler) Update(w http.ResponseWriter, r *http.Request) {
	var kafkaUpdateReq public.KafkaUpdateRequest
	id := mux.Vars(r)["id"]
	ctx := r.Context()
	kafkaRequest, kafkaGetError := h.service.Get(ctx, id)
	validateKafkaFound := func() handlers.Validate {
		return func() *errors.ServiceError {
			return kafkaGetError
		}
	}
	cfg := &handlers.HandlerConfig{
		MarshalInto: &kafkaUpdateReq,
		Validate: []handlers.Validate{
			validateKafkaFound(),
			ValidateKafkaUserFacingUpdateFields(ctx, h.authService, kafkaRequest, &kafkaUpdateReq),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			updatedNeeded := false
			if kafkaUpdateReq.ReauthenticationEnabled != nil && kafkaRequest.ReauthenticationEnabled != *kafkaUpdateReq.ReauthenticationEnabled {
				kafkaRequest.ReauthenticationEnabled = *kafkaUpdateReq.ReauthenticationEnabled
				updatedNeeded = true
			}

			if kafkaUpdateReq.Owner != nil && kafkaRequest.Owner != *kafkaUpdateReq.Owner {
				kafkaRequest.Owner = *kafkaUpdateReq.Owner
				updatedNeeded = true
			}

			if updatedNeeded {
				updateErr := h.service.Updates(kafkaRequest, map[string]interface{}{
					"reauthentication_enabled": kafkaRequest.ReauthenticationEnabled,
					"owner":                    kafkaRequest.Owner,
				})

				if updateErr != nil {
					return nil, updateErr
				}
			}

			return presenters.PresentKafkaRequest(kafkaRequest, h.kafkaConfig)
		},
	}
	handlers.Handle(w, r, cfg, http.StatusOK)
}
