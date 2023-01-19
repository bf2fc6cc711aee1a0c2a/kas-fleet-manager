package actions

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/workers/kafka_mgrs/promotion/chain"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/golang/glog"
)

var _ chain.ReconcileAction[PromotionContext] = &ReserveDesiredQuotaAction{}

type ReserveDesiredQuotaAction struct {
	quotaServiceFactory services.QuotaServiceFactory
	kafkaConfig         config.KafkaConfig
}

func NewReserveDesiredQuotaAction(kafkaConfig config.KafkaConfig, quotaServiceFactory services.QuotaServiceFactory) chain.ReconcileAction[PromotionContext] {
	return &ReserveDesiredQuotaAction{
		quotaServiceFactory: quotaServiceFactory,
		kafkaConfig:         kafkaConfig,
	}
}

// PerformJob reserves quota for the received kafkaRequest object. The quota is reserved only if it has not been already reserved
// (ie: running this action multiple times on the same kafkaRequest doesn't allocate multiple quotas)
// The type of quota to be allocated is inferred from the DesiredBillingModel attribute of the kafkaRequest object
func (d *ReserveDesiredQuotaAction) PerformJob(kafkaRequest *dbapi.KafkaRequest, currentResult chain.ActionResult[PromotionContext]) (chain.ActionResult[PromotionContext], bool, error) {
	glog.Infof("reserving quota for '%s' to cluster with ID '%s'", kafkaRequest.DesiredKafkaBillingModel, kafkaRequest.ClusterID)
	res := chain.ActionResult[PromotionContext]{}
	quotaService, factoryErr := d.quotaServiceFactory.GetQuotaService(api.QuotaType(d.kafkaConfig.Quota.Type))
	if factoryErr != nil {
		return res, true, errors.NewWithCause(errors.ErrorGeneral, factoryErr, "unable to check quota")
	}

	// quota reservation methods has side effects that we don't need in this flow
	// I will create a copy of `kafkaRequest` to avoid such side effects
	kafkaCopy := *kafkaRequest
	subscriptionID, err := quotaService.ReserveQuotaIfNotAlreadyReserved(&kafkaCopy)
	if err != nil {
		return res, true, errors.NewWithCause(errors.ErrorGeneral, err, "unable to reserve quota")
	}
	glog.Infof("reserved quota for '%s' to cluster with ID '%s': %s", kafkaRequest.DesiredKafkaBillingModel, kafkaRequest.ClusterID, subscriptionID)
	res.SetValue(PromotionContext{SubscriptionID: subscriptionID})
	return res, false, nil
}
