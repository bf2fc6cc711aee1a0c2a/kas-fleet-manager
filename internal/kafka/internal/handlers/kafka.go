package handlers

import (
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"

	"github.com/gorilla/mux"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
)

type kafkaHandler struct {
	service        services.KafkaService
	providerConfig *config.ProviderConfig
}

func NewKafkaHandler(service services.KafkaService, providerConfig *config.ProviderConfig) *kafkaHandler {
	return &kafkaHandler{
		service:        service,
		providerConfig: providerConfig,
	}
}

func (h kafkaHandler) Create(w http.ResponseWriter, r *http.Request) {
	var kafkaRequest public.KafkaRequestPayload
	cfg := &handlers.HandlerConfig{
		MarshalInto: &kafkaRequest,
		Validate: []handlers.Validate{
			handlers.ValidateAsyncEnabled(r, "creating kafka requests"),
			handlers.ValidateLength(&kafkaRequest.Name, "name", &handlers.MinRequiredFieldLength, &MaxKafkaNameLength),
			ValidKafkaClusterName(&kafkaRequest.Name, "name"),
			ValidateKafkaClusterNameIsUnique(&kafkaRequest.Name, h.service, r.Context()),
			ValidateCloudProvider(&kafkaRequest, h.providerConfig, "creating kafka requests"),
			handlers.ValidateMultiAZEnabled(&kafkaRequest.MultiAz, "creating kafka requests"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			convKafka := presenters.ConvertKafkaRequest(kafkaRequest)

			claims, err := auth.GetClaimsFromContext(ctx)
			if err != nil {
				return nil, errors.Unauthenticated("user not authenticated")
			}
			convKafka.Owner = auth.GetUsernameFromClaims(claims)
			convKafka.OrganisationId = auth.GetOrgIdFromClaims(claims)
			convKafka.OwnerAccountId = auth.GetAccountIdFromClaims(claims)

			svcErr := h.service.RegisterKafkaJob(convKafka)
			if svcErr != nil {
				return nil, svcErr
			}
			return presenters.PresentKafkaRequest(convKafka), nil
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
			return presenters.PresentKafkaRequest(kafkaRequest), nil
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
				converted := presenters.PresentKafkaRequest(kafkaRequest)
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
	cfg := &handlers.HandlerConfig{
		MarshalInto: &kafkaUpdateReq,
		Validate: []handlers.Validate{
			handlers.ValidateMinLength(&kafkaUpdateReq.Owner, "owner", 1),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			kafkaRequest, err2 := h.service.Get(ctx, id)
			if err2 != nil {
				return nil, err2
			}
			if kafkaRequest.Owner != kafkaUpdateReq.Owner {
				kafkaRequest.Owner = kafkaUpdateReq.Owner
				err3 := h.service.VerifyAndUpdateKafka(ctx, kafkaRequest)
				if err3 != nil {
					return nil, err3
				}
			}
			return presenters.PresentKafkaRequest(kafkaRequest), nil
		},
	}
	handlers.Handle(w, r, cfg, http.StatusOK)
}
