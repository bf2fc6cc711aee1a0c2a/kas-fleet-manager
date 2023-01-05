package actions

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/workers/kafka_mgrs/promotion/chain"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

var _ chain.ReconcileAction[PromotionContext] = &DeleteActualQuotaAction{}

type DeleteActualQuotaAction struct {
	quotaServiceFactory services.QuotaServiceFactory
	kafkaConfig         config.KafkaConfig
}

func NewDeleteActualQuotaAction(kafkaConfig config.KafkaConfig, quotaServiceFactory services.QuotaServiceFactory) chain.ReconcileAction[PromotionContext] {
	return &DeleteActualQuotaAction{
		quotaServiceFactory: quotaServiceFactory,
		kafkaConfig:         kafkaConfig,
	}
}

func (d *DeleteActualQuotaAction) PerformJob(kafkaRequest *dbapi.KafkaRequest, currentResult chain.ActionResult[PromotionContext]) (chain.ActionResult[PromotionContext], bool, error) {
	res := chain.ActionResult[PromotionContext]{}
	quotaService, factoryErr := d.quotaServiceFactory.GetQuotaService(api.QuotaType(d.kafkaConfig.Quota.Type))
	if factoryErr != nil {
		return res, true, errors.NewWithCause(errors.ErrorGeneral, factoryErr, "unable to delete quota")
	}
	instanceType, err := d.kafkaConfig.SupportedInstanceTypes.Configuration.GetKafkaInstanceTypeByID(kafkaRequest.InstanceType)
	if err != nil {
		// instance type was validated at creation stage. This should never happen.
		return res, true, err
	}
	bm, err := instanceType.GetKafkaSupportedBillingModelByID(kafkaRequest.ActualKafkaBillingModel)
	if err != nil {
		// actual billing model was validated at creation stage. This should never happen.
		return res, true, err
	}

	// WARNING: all the other methods called before are returning an `error`. Quota service methods
	// are returning a *ServiceError object. If we reuse the `err` variable here, err != nil will be
	// always `false`.
	if err := quotaService.DeleteQuotaForBillingModel(kafkaRequest.SubscriptionId, *bm); err != nil {
		// at this stage we have already reserved the new quota, so we need the reconciler to keep retrying
		// mark the error as recoverable
		return res, true, errors.NewServiceErrorBuilder().Wrap(*err).Recoverable().Build()
	}
	// deleted
	return res, false, nil
}
