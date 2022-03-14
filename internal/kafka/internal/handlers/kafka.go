package handlers

import (
	"fmt"
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"

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
	var kafkaRequest public.KafkaRequestPayload
	ctx := r.Context()
	convKafka := &dbapi.KafkaRequest{}

	cfg := &handlers.HandlerConfig{
		MarshalInto: &kafkaRequest,
		Validate: []handlers.Validate{
			handlers.ValidateAsyncEnabled(r, "creating kafka requests"),
			handlers.ValidateLength(&kafkaRequest.Name, "name", &handlers.MinRequiredFieldLength, &MaxKafkaNameLength),
			ValidKafkaClusterName(&kafkaRequest.Name, "name"),
			ValidateKafkaClusterNameIsUnique(&kafkaRequest.Name, h.service, r.Context()),
			ValidateKafkaClaims(ctx, &kafkaRequest, convKafka),
			ValidateCloudProvider(&h.service, convKafka, h.providerConfig, "creating kafka requests"),
			handlers.ValidateMultiAZEnabled(&kafkaRequest.MultiAz, "creating kafka requests"),
			func() *errors.ServiceError { // Validate plan
				instanceType, err := h.service.AssignInstanceType(convKafka)
				if err != nil {
					return err
				}
				if stringSet(&kafkaRequest.Plan) {
					plan := config.Plan(kafkaRequest.Plan)
					instTypeFromPlan, e := plan.GetInstanceType()
					if e != nil || instTypeFromPlan != string(instanceType) {
						return errors.New(errors.ErrorBadRequest, fmt.Sprintf("Unable to detect instance type in plan provided: '%s'", kafkaRequest.Plan))
					}
					size, e1 := plan.GetSizeID()
					if e1 != nil {
						return errors.New(errors.ErrorBadRequest, fmt.Sprintf("Unable to detect instance size in plan provided: '%s'", kafkaRequest.Plan))
					}
					_, e2 := h.kafkaConfig.GetKafkaInstanceSize(instTypeFromPlan, size)

					if e2 != nil {
						return errors.InstancePlanNotSupported("Unsupported plan provided: '%s'", kafkaRequest.Plan)
					}
					convKafka.SizeId = size
				} else {
					rSize, err := h.kafkaConfig.GetFirstAvailableSize(instanceType.String())
					if err != nil {
						return errors.InstanceTypeNotSupported("Unsupported kafka instance type: '%s' provided", instanceType.String())
					}
					convKafka.SizeId = rSize.Id
				}
				convKafka.InstanceType = instanceType.String()
				return nil
			},
		},
		Action: func() (interface{}, *errors.ServiceError) {
			svcErr := h.service.RegisterKafkaJob(convKafka)
			if svcErr != nil {
				return nil, svcErr
			}
			return presenters.PresentKafkaRequest(convKafka, h.kafkaConfig), nil
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
			return presenters.PresentKafkaRequest(kafkaRequest, h.kafkaConfig), nil
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
				converted := presenters.PresentKafkaRequest(kafkaRequest, h.kafkaConfig)
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

			return presenters.PresentKafkaRequest(kafkaRequest, h.kafkaConfig), nil
		},
	}
	handlers.Handle(w, r, cfg, http.StatusOK)
}
