package handlers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
)

type kafkaHandler struct {
	service services.KafkaService
	config  services.ConfigService
}

func NewKafkaHandler(service services.KafkaService, configService services.ConfigService) *kafkaHandler {
	return &kafkaHandler{
		service: service,
		config:  configService,
	}
}

func (h kafkaHandler) Create(w http.ResponseWriter, r *http.Request) {
	var kafkaRequest openapi.KafkaRequestPayload
	cfg := &handlers.HandlerConfig{
		MarshalInto: &kafkaRequest,
		Validate: []handlers.Validate{
			handlers.ValidateAsyncEnabled(r, "creating kafka requests"),
			handlers.ValidateLength(&kafkaRequest.Name, "name", &handlers.MinRequiredFieldLength, &handlers.MaxKafkaNameLength),
			handlers.ValidKafkaClusterName(&kafkaRequest.Name, "name"),
			handlers.ValidateKafkaClusterNameIsUnique(&kafkaRequest.Name, h.service, r.Context()),
			handlers.ValidateCloudProvider(&kafkaRequest, h.config, "creating kafka requests"),
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

			listArgs := services.NewListArguments(r.URL.Query())

			if err := listArgs.Validate(); err != nil {
				return nil, errors.NewWithCause(errors.ErrorMalformedRequest, err, "Unable to list kafka requests: %s", err.Error())
			}

			kafkaRequests, paging, err := h.service.List(ctx, listArgs)
			if err != nil {
				return nil, err
			}

			kafkaRequestList := openapi.KafkaRequestList{
				Kind:  "KafkaRequestList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.KafkaRequest{},
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
