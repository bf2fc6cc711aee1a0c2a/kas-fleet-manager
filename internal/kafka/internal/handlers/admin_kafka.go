package handlers

import (
	"fmt"
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/account"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/gorilla/mux"
)

type adminKafkaHandler struct {
	service        services.KafkaService
	accountService account.AccountService
	providerConfig *config.ProviderConfig
}

func NewAdminKafkaHandler(service services.KafkaService, accountService account.AccountService, providerConfig *config.ProviderConfig) *adminKafkaHandler {
	return &adminKafkaHandler{
		service:        service,
		accountService: accountService,
		providerConfig: providerConfig,
	}
}

func (h adminKafkaHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			kafkaRequest, err := h.service.Get(ctx, id)
			if err != nil {
				return nil, err
			}
			return presenters.PresentKafkaRequestAdminEndpoint(kafkaRequest, h.accountService)
		},
	}
	handlers.HandleGet(w, r, cfg)
}

func (h adminKafkaHandler) List(w http.ResponseWriter, r *http.Request) {
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

			kafkaRequestList := private.KafkaList{
				Kind:  "KafkaList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []private.Kafka{},
			}

			for _, kafkaRequest := range kafkaRequests {
				converted, err := presenters.PresentKafkaRequestAdminEndpoint(kafkaRequest, h.accountService)
				if err != nil {
					return nil, err
				}
				kafkaRequestList.Items = append(kafkaRequestList.Items, *converted)
			}

			return kafkaRequestList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h adminKafkaHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

func (h adminKafkaHandler) Update(w http.ResponseWriter, r *http.Request) {

	id := mux.Vars(r)["id"]
	ctx := r.Context()
	kafkaRequest, err := h.service.Get(ctx, id)

	var kafkaUpdateReq private.KafkaUpdateRequest
	cfg := &handlers.HandlerConfig{
		MarshalInto: &kafkaUpdateReq,
		Validate: []handlers.Validate{
			ValidateKafkaUpdateFields(
				&kafkaUpdateReq,
			),
			ValidateKafkaStorageSize(kafkaRequest, &kafkaUpdateReq),
			func() *errors.ServiceError { // Validate status
				kafkaStatus := kafkaRequest.Status
				if !shared.Contains(constants.GetUpdateableStatuses(), kafkaStatus) {
					return errors.New(errors.ErrorValidation, fmt.Sprintf("Unable to update kafka in %s status. Supported statuses for update are: %v", kafkaStatus, constants.GetUpdateableStatuses()))
				}
				return nil
			},
			func() *errors.ServiceError { // Validate DesiredKafkaVersion
				if kafkaRequest.DesiredKafkaVersion != kafkaUpdateReq.KafkaVersion && kafkaUpdateReq.KafkaVersion != "" && kafkaRequest.KafkaUpgrading {
					return errors.New(errors.ErrorValidation, "Unable to update kafka version. Another upgrade is already in progress.")
				}
				return nil
			},
			func() *errors.ServiceError { // Validate DesiredStrimziVersion
				if kafkaRequest.DesiredStrimziVersion != kafkaUpdateReq.StrimziVersion && kafkaUpdateReq.StrimziVersion != "" && kafkaRequest.StrimziUpgrading {
					return errors.New(errors.ErrorValidation, "Unable to update strimzi version. Another upgrade is already in progress.")
				}
				return nil
			},
			func() *errors.ServiceError { // Validate DesiredKafkaIBPVersion
				if kafkaRequest.DesiredKafkaIBPVersion != kafkaUpdateReq.KafkaIbpVersion && kafkaUpdateReq.KafkaIbpVersion != "" && kafkaRequest.KafkaIBPUpgrading {
					return errors.New(errors.ErrorValidation, "Unable to update ibp version. Another upgrade is already in progress.")
				}
				return nil
			},
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {

			if err != nil {
				return nil, err
			}

			update := func(val1 *string, val2 string) bool {
				if val2 != "" && *val1 != val2 {
					*val1 = val2
					return true
				}
				return false
			}

			updateRequired := update(&kafkaRequest.DesiredKafkaVersion, kafkaUpdateReq.KafkaVersion)
			updateRequired = update(&kafkaRequest.DesiredStrimziVersion, kafkaUpdateReq.StrimziVersion) || updateRequired
			updateRequired = update(&kafkaRequest.DesiredKafkaIBPVersion, kafkaUpdateReq.KafkaIbpVersion) || updateRequired
			updateRequired = update(&kafkaRequest.KafkaStorageSize, kafkaUpdateReq.KafkaStorageSize) || updateRequired

			if updateRequired {
				err3 := h.service.VerifyAndUpdateKafkaAdmin(ctx, kafkaRequest)
				if err3 != nil {
					return nil, err3
				}
			}
			return presenters.PresentKafkaRequestAdminEndpoint(kafkaRequest, h.accountService)
		},
	}
	handlers.Handle(w, r, cfg, http.StatusOK)
}
