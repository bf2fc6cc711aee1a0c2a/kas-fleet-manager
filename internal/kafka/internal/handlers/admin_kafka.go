package handlers

import (
	"fmt"
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/account"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/gorilla/mux"
)

type adminKafkaHandler struct {
	kafkaService   services.KafkaService
	accountService account.AccountService
	providerConfig *config.ProviderConfig
	clusterService services.ClusterService
}

func NewAdminKafkaHandler(kafkaService services.KafkaService, accountService account.AccountService, providerConfig *config.ProviderConfig, clusterService services.ClusterService) *adminKafkaHandler {
	return &adminKafkaHandler{
		kafkaService:   kafkaService,
		accountService: accountService,
		providerConfig: providerConfig,
		clusterService: clusterService,
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
				return nil, errors.NewWithCause(errors.ErrorMalformedRequest, err, "Unable to list kafka requests: %s", err.Error())
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

func (h adminKafkaHandler) Update(w http.ResponseWriter, r *http.Request) {

	id := mux.Vars(r)["id"]
	ctx := r.Context()
	kafkaRequest, err := h.kafkaService.Get(ctx, id)

	var kafkaUpdateReq private.KafkaUpdateRequest
	cfg := &handlers.HandlerConfig{
		MarshalInto: &kafkaUpdateReq,
		Validate: []handlers.Validate{
			func() *errors.ServiceError { // Validate kafka found
				if err != nil {
					return err
				}
				if kafkaRequest == nil {
					return errors.NotFound("Unable to find kafka with id '%s'", id)
				}
				return nil
			},
			ValidateKafkaUpdateFields(
				&kafkaUpdateReq,
			),
			ValidateKafkaStorageSize(kafkaRequest, &kafkaUpdateReq),
			func() *errors.ServiceError { // Validate status
				kafkaStatus := kafkaRequest.Status
				if !arrays.Contains(constants.GetUpdateableStatuses(), kafkaStatus) {
					return errors.New(errors.ErrorValidation, fmt.Sprintf("Unable to update kafka in %s status. Supported statuses for update are: %v", kafkaStatus, constants.GetUpdateableStatuses()))
				}
				return nil
			},
			func() *errors.ServiceError { // Validate DesiredKafkaVersion
				if kafkaRequest.DesiredKafkaVersion != kafkaUpdateReq.KafkaVersion && kafkaUpdateReq.KafkaVersion != "" && kafkaRequest.KafkaUpgrading {
					return errors.New(errors.ErrorValidation, "Unable to update kafka version. Another upgrade is already in progress.")
				}

				currentKafkaVersion, _ := arrays.FirstNonEmpty(kafkaRequest.ActualKafkaVersion, kafkaRequest.DesiredKafkaVersion)

				vCompKafka, ek := api.CompareSemanticVersionsMajorAndMinor(currentKafkaVersion, kafkaRequest.DesiredKafkaVersion)

				if ek != nil {
					return errors.New(errors.ErrorValidation, fmt.Sprintf("Unable to compare desired kafka version: %s with actual kafka version: %s", kafkaRequest.DesiredKafkaVersion, currentKafkaVersion))
				}

				// no minor/ major version downgrades allowed for kafka version
				if vCompKafka > 0 {
					return errors.New(errors.ErrorValidation, fmt.Sprintf("Unable to downgrade kafka: %s version: %s to the following kafka version: %s", kafkaRequest.ID, currentKafkaVersion, kafkaRequest.DesiredKafkaVersion))
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

				currentIBPVersion, _ := arrays.FirstNonEmpty(kafkaRequest.ActualKafkaIBPVersion, kafkaRequest.DesiredKafkaIBPVersion)
				vCompOldNewIbp, eIbp := api.CompareBuildAwareSemanticVersions(currentIBPVersion, kafkaRequest.DesiredKafkaIBPVersion)

				if eIbp != nil {
					return errors.New(errors.ErrorValidation, fmt.Sprintf("Unable to compare actual ibp version: %s with desired ibp version: %s", currentIBPVersion, kafkaRequest.DesiredKafkaVersion))
				}

				// actual ibp version cannot be greater than desired ibp version (no downgrade allowed)
				if vCompOldNewIbp > 0 {
					return errors.New(errors.ErrorValidation, fmt.Sprintf("Unable to downgrade kafka: %s ibp version: %s to a lower version: %s", kafkaRequest.ID, kafkaRequest.DesiredKafkaIBPVersion, currentIBPVersion))
				}

				vCompIbpKafka, eIbpK := api.CompareBuildAwareSemanticVersions(kafkaRequest.DesiredKafkaIBPVersion, kafkaRequest.DesiredKafkaVersion)

				if eIbpK != nil {
					return errors.New(errors.ErrorValidation, fmt.Sprintf("Unable to compare kafka ibp version: %s with kafka version: %s", kafkaRequest.DesiredKafkaIBPVersion, kafkaRequest.DesiredKafkaVersion))
				}

				// ibp version cannot be greater than kafka version
				if vCompIbpKafka > 0 {
					return errors.New(errors.ErrorValidation, fmt.Sprintf("Unable to update kafka: %s ibp version: %s with kafka version: %s", kafkaRequest.ID, kafkaRequest.DesiredKafkaIBPVersion, kafkaRequest.DesiredKafkaVersion))
				}

				return nil
			},
			func() *errors.ServiceError { // Validate Strimzi Version
				cluster, err := h.clusterService.FindClusterByID(kafkaRequest.ClusterID)
				if err != nil {
					return errors.NewWithCause(errors.ErrorGeneral, err, "Unable to find cluster associated with kafka request: %s", kafkaRequest.ID)
				}

				if cluster == nil {
					return errors.New(errors.ErrorValidation, fmt.Sprintf("Unable to get cluster for kafka %s", kafkaRequest.ID))
				}

				kafkaVersionAvailable, err2 := h.clusterService.IsStrimziKafkaVersionAvailableInCluster(cluster, kafkaRequest.DesiredStrimziVersion, kafkaRequest.DesiredKafkaVersion, kafkaRequest.DesiredKafkaIBPVersion)

				if err2 != nil {
					return errors.Validation(err2.Error())
				}

				if !kafkaVersionAvailable {
					return errors.New(errors.ErrorValidation, fmt.Sprintf("Unable to update kafka: %s with kafka version: %s", kafkaRequest.ID, kafkaRequest.DesiredKafkaVersion))
				}

				strimziVersionReady, err2 := h.clusterService.CheckStrimziVersionReady(cluster, kafkaRequest.DesiredStrimziVersion)

				if err2 != nil {
					return errors.Validation(err2.Error())
				}

				if !strimziVersionReady {
					return errors.New(errors.ErrorValidation, fmt.Sprintf("Unable to update kafka: %s with strimzi version: %s", kafkaRequest.ID, kafkaRequest.DesiredStrimziVersion))
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
				err3 := h.kafkaService.VerifyAndUpdateKafkaAdmin(ctx, kafkaRequest)
				if err3 != nil {
					return nil, err3
				}
			}
			return presenters.PresentKafkaRequestAdminEndpoint(kafkaRequest, h.accountService)
		},
	}
	handlers.Handle(w, r, cfg, http.StatusOK)
}
