package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/account"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services/kafka_tls_certificate_management"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/gorilla/mux"
)

type adminKafkaHandler struct {
	kafkaService   services.KafkaService
	accountService account.AccountService
	clusterService services.ClusterService

	providerConfig *config.ProviderConfig
	kafkaConfig    *config.KafkaConfig

	kafkaTLSCertificateManagementService kafka_tls_certificate_management.KafkaTLSCertificateManagementService
}

func NewAdminKafkaHandler(kafkaService services.KafkaService, accountService account.AccountService, providerConfig *config.ProviderConfig, clusterService services.ClusterService, kafkaConfig *config.KafkaConfig,
	kafkaTLSCertificateManagementService kafka_tls_certificate_management.KafkaTLSCertificateManagementService) *adminKafkaHandler {
	return &adminKafkaHandler{
		kafkaService:   kafkaService,
		accountService: accountService,
		clusterService: clusterService,

		providerConfig:                       providerConfig,
		kafkaConfig:                          kafkaConfig,
		kafkaTLSCertificateManagementService: kafkaTLSCertificateManagementService,
	}
}

func (h adminKafkaHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			kafkaRequest, err := h.kafkaService.Get(ctx, id)
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

			if err := listArgs.Validate(GetAcceptedOrderByParams()); err != nil {
				return nil, errors.NewWithCause(errors.ErrorMalformedRequest, err, "unable to list kafka requests: %s", err.Error())
			}

			kafkaRequests, paging, err := h.kafkaService.List(ctx, listArgs)
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

			err := h.kafkaService.RegisterKafkaDeprovisionJob(ctx, id)
			return nil, err
		},
	}

	handlers.HandleDelete(w, r, cfg, http.StatusAccepted)
}

func (h *adminKafkaHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	ctx := r.Context()
	kafkaRequest, err := h.kafkaService.Get(ctx, id)

	var kafkaUpdateReq private.KafkaUpdateRequest
	cfg := &handlers.HandlerConfig{
		MarshalInto: &kafkaUpdateReq,
		Validate: []handlers.Validate{
			validateGettingKafkaFromDatabase(id, kafkaRequest, err),
			ValidateKafkaUpdateFields(
				&kafkaUpdateReq,
			),
			ValidateMaxDataRetentionSize(kafkaRequest, &kafkaUpdateReq),
			func() *errors.ServiceError { // Validate status
				kafkaStatus := kafkaRequest.Status
				if !arrays.Contains(constants.GetUpdateableStatuses(), kafkaStatus) {
					return errors.New(errors.ErrorValidation, fmt.Sprintf("unable to update kafka in %s status. Supported statuses for update are: %v", kafkaStatus, constants.GetUpdateableStatuses()))
				}
				return nil
			},
			func() *errors.ServiceError { // Validate DesiredKafkaVersion
				if kafkaRequest.DesiredKafkaVersion != kafkaUpdateReq.KafkaVersion && kafkaUpdateReq.KafkaVersion != "" && kafkaRequest.KafkaUpgrading {
					return errors.New(errors.ErrorValidation, "unable to update kafka version. Another upgrade is already in progress")
				}
				return nil
			},
			func() *errors.ServiceError { // Validate DesiredStrimziVersion
				if kafkaRequest.DesiredStrimziVersion != kafkaUpdateReq.StrimziVersion && kafkaUpdateReq.StrimziVersion != "" && kafkaRequest.StrimziUpgrading {
					return errors.New(errors.ErrorValidation, "unable to update strimzi version. Another upgrade is already in progress")
				}
				return nil
			},
			func() *errors.ServiceError { // Validate DesiredKafkaIBPVersion
				if kafkaRequest.DesiredKafkaIBPVersion != kafkaUpdateReq.KafkaIbpVersion && kafkaUpdateReq.KafkaIbpVersion != "" && kafkaRequest.KafkaIBPUpgrading {
					return errors.New(errors.ErrorValidation, "unable to update ibp version. Another upgrade is already in progress")
				}
				return nil
			},
			validateVersionsCompatibility(h, kafkaRequest, &kafkaUpdateReq),
			h.validateUpdateKafkaSuspended(kafkaRequest, &kafkaUpdateReq),
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

			getStatusBasedOnSuspendedParam := func(susp *bool, kafka *dbapi.KafkaRequest) string {
				if shared.IsNil(susp) {
					return kafka.Status
				} else {
					if *susp {
						if kafka.Status == constants.KafkaRequestStatusReady.String() {
							return constants.KafkaRequestStatusSuspending.String()
						}
					} else {
						if kafka.Status == constants.KafkaRequestStatusSuspended.String() || kafka.Status == constants.KafkaRequestStatusSuspending.String() {
							return constants.KafkaRequestStatusResuming.String()
						}
					}
				}
				return kafka.Status
			}

			updateRequired := update(&kafkaRequest.DesiredKafkaVersion, kafkaUpdateReq.KafkaVersion)
			updateRequired = update(&kafkaRequest.DesiredStrimziVersion, kafkaUpdateReq.StrimziVersion) || updateRequired
			updateRequired = update(&kafkaRequest.DesiredKafkaIBPVersion, kafkaUpdateReq.KafkaIbpVersion) || updateRequired
			updateRequired = update(&kafkaRequest.MaxDataRetentionSize, kafkaUpdateReq.MaxDataRetentionSize) || updateRequired

			newStatus := getStatusBasedOnSuspendedParam(kafkaUpdateReq.Suspended, kafkaRequest)
			updateRequired = update(&kafkaRequest.Status, newStatus) || updateRequired

			if updateRequired {
				err := h.kafkaService.VerifyAndUpdateKafkaAdmin(ctx, kafkaRequest)
				if err != nil {
					return nil, err
				}
			}
			return presenters.PresentKafkaRequestAdminEndpoint(kafkaRequest, h.accountService)
		},
	}
	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h *adminKafkaHandler) RevokeCertificateOfAKafka(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	ctx := r.Context()
	kafkaRequest, err := h.kafkaService.Get(ctx, id)

	var kafkaCertificationRevocationRequest private.KafkacertificateRevocationRequest
	cfg := &handlers.HandlerConfig{
		MarshalInto: &kafkaCertificationRevocationRequest,
		Validate: []handlers.Validate{
			validateGettingKafkaFromDatabase(id, kafkaRequest, err),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			reason, err := kafka_tls_certificate_management.ParseReason(int(kafkaCertificationRevocationRequest.RevocationReason))
			if err != nil {
				return nil, errors.NewWithCause(errors.ErrorBadRequest, err, err.Error())
			}

			certRevocationErr := h.kafkaTLSCertificateManagementService.RevokeCertificate(ctx, kafkaRequest.KafkasRoutesBaseDomainName, reason)

			if certRevocationErr != nil {
				return nil, errors.NewWithCause(errors.ErrorGeneral, certRevocationErr, certRevocationErr.Error())
			}

			return nil, nil
		},
	}
	handlers.Handle(w, r, cfg, http.StatusNoContent)
}

func (h *adminKafkaHandler) validateUpdateKafkaSuspended(kafkaRequest *dbapi.KafkaRequest, kafkaUpdateReq *private.KafkaUpdateRequest) handlers.Validate {
	return func() *errors.ServiceError {
		if kafkaUpdateReq.Suspended == nil {
			return nil
		}

		if *kafkaUpdateReq.Suspended {
			return h.validateUpdateKafkaCanBeSuspended(kafkaRequest, kafkaUpdateReq)()
		} else {
			return h.validateUpdateKafkaCanBeResumed(kafkaRequest, kafkaUpdateReq)()
		}
	}
}

func (h *adminKafkaHandler) validateUpdateKafkaCanBeSuspended(kafkaRequest *dbapi.KafkaRequest, kafkaUpdateReq *private.KafkaUpdateRequest) handlers.Validate {
	return func() *errors.ServiceError {
		suspendableStates := []string{constants.KafkaRequestStatusReady.String()}
		isSuspendableState := arrays.Contains(suspendableStates, kafkaRequest.Status)
		if !isSuspendableState {
			return errors.New(errors.ErrorValidation, "kafka instance with a status of %q cannot be suspended. Kafka instances can only be suspended in the following states: %s", kafkaRequest.Status, suspendableStates)
		}
		return nil
	}
}

func (h *adminKafkaHandler) validateUpdateKafkaCanBeResumed(kafkaRequest *dbapi.KafkaRequest, kafkaUpdateReq *private.KafkaUpdateRequest) handlers.Validate {
	return func() *errors.ServiceError {
		resumableStates := []string{constants.KafkaRequestStatusSuspended.String()}
		isResumableState := arrays.Contains(resumableStates, kafkaRequest.Status)
		if !isResumableState {
			return errors.New(errors.ErrorValidation, "kafka instance with a status of %q cannot be resumed. Kafka instances can only be resumed in the following states: %s", kafkaRequest.Status, resumableStates)
		}

		kafkaRequestHasExpirationSet := kafkaRequest.ExpiresAt.Valid
		if !kafkaRequestHasExpirationSet {
			return nil
		}

		timeNow := time.Now()
		kafkaBillingModelConfig, err := h.kafkaConfig.GetBillingModelByID(kafkaRequest.InstanceType, kafkaRequest.ActualKafkaBillingModel)
		if err != nil {
			return errors.ToServiceError(err)
		}
		gracePeriodDays := kafkaBillingModelConfig.GracePeriodDays
		durationGracePeriodDays := time.Duration(gracePeriodDays*86400) * time.Second
		startOfGracePeriod := kafkaRequest.ExpiresAt.Time.Add(-durationGracePeriodDays)
		isWithinOrAfterGracePeriod := timeNow.After(startOfGracePeriod)
		if isWithinOrAfterGracePeriod {
			return errors.New(errors.ErrorValidation, "kafka instance with a status of %q cannot be resumed due to the instance is suspended and it is within its grace period: start of grace period: %s ", kafkaRequest.Status, startOfGracePeriod)
		}

		return nil
	}
}

func validateGettingKafkaFromDatabase(kafkaId string, kafkaFromDatabase *dbapi.KafkaRequest, err *errors.ServiceError) func() *errors.ServiceError {
	return func() *errors.ServiceError {
		if err != nil {
			return err
		}
		if kafkaFromDatabase == nil {
			return errors.NotFound("unable to find kafka with id %q", kafkaId)
		}
		return nil
	}
}
