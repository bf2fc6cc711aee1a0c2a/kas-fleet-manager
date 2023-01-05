package promotion

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/workers/kafka_mgrs/promotion/chain"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/workers/kafka_mgrs/promotion/internal/actions"
	apiErrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type PromotionKafkaManager struct {
	workers.BaseWorker
	kafkaService        services.KafkaService
	quotaServiceFactory services.QuotaServiceFactory
	kafkaConfig         *config.KafkaConfig
}

func NewPromotionKafkaManager(reconciler workers.Reconciler,
	kafkaService services.KafkaService,
	kafkaConfig *config.KafkaConfig,
	quotaServiceFactory services.QuotaServiceFactory) *PromotionKafkaManager {
	return &PromotionKafkaManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "promoting_kafka",
			Reconciler: reconciler,
		},
		kafkaService:        kafkaService,
		quotaServiceFactory: quotaServiceFactory,
		kafkaConfig:         kafkaConfig,
	}
}

// Start initializes the kafka manager to reconcile kafka requests to be promoted.
func (k *PromotionKafkaManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling kafka requests to be promoted to stop.
func (k *PromotionKafkaManager) Stop() {
	k.StopWorker(k)
}

func (k *PromotionKafkaManager) updateFailedPromotionDetails(kafkaRequest *dbapi.KafkaRequest, promotionError error) error {
	kafkaRequest.PromotionDetails = promotionError.Error()

	if serviceError, ok := promotionError.(*apiErrors.ServiceError); ok && serviceError.Recoverable() {
		kafkaRequest.PromotionStatus = dbapi.KafkaPromotionStatusPromoting
	} else {
		kafkaRequest.PromotionStatus = dbapi.KafkaPromotionStatusFailed
	}

	err := k.kafkaService.Update(kafkaRequest)
	if err != nil {
		return err
	}
	return nil
}

func (k *PromotionKafkaManager) Reconcile() []error {
	glog.Infoln("reconciling kafkas to be promoted")
	kafkasToPromote, err := k.kafkaService.ListKafkasToBePromoted()
	if err != nil {
		return []error{errors.Wrap(err, "failed to list kafkas to promote")}
	}
	glog.Infof("found %d kafkas to promote", len(kafkasToPromote))

	var promotionErrors apiErrors.ErrorList

	for _, kafka := range kafkasToPromote {
		subscriptionID, promotionError := k.promote(kafka)

		if promotionError != nil {
			promotionErrors.AddErrors(errors.Wrapf(promotionError, "failed to promote kafka with id '%s'", kafka.ID))
			glog.Errorf("failed promoting kafka with ID '%s' : %s", kafka.ID, promotionError.Error())
			if err := k.updateFailedPromotionDetails(kafka, promotionError); err != nil {
				// log the error
				glog.Errorf("failed saving promotion error details for kafka '%s' into the database: %s", kafka.ID, err.Error())
			}
		} else {
			glog.Infof("kafka with ID '%s' promoted to '%s'. New subscription ID: %s", kafka.ID, kafka.ActualKafkaBillingModel, subscriptionID)
		}
	}

	return promotionErrors.ToErrorSlice()
}

func (k *PromotionKafkaManager) promote(kafka *dbapi.KafkaRequest) (string, error) {
	// setup pipeline
	promotionChain := chain.NewReconcileActionRunner(
		actions.NewReserveDesiredQuotaAction(*k.kafkaConfig, k.quotaServiceFactory),
		actions.NewDeleteActualQuotaAction(*k.kafkaConfig, k.quotaServiceFactory),
		actions.NewUpdateKafkaRequestAction(k.kafkaService),
	)
	res, err := promotionChain.Run(kafka)
	if err != nil {
		return "", err
	}
	return res.SubscriptionID, nil
}
