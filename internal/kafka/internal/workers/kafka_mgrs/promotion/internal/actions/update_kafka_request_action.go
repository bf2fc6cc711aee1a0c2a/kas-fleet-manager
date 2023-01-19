package actions

import (
	"database/sql"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/workers/kafka_mgrs/promotion/chain"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/golang/glog"
	"time"
)

var _ chain.ReconcileAction[PromotionContext] = &UpdateKafkaRequestAction{}

/**************
 * This action updates the kafka request after a successful promotion
 * WARNING: It is expected that one of the previous chain ring should populate the `SubscriptionID` field of the result
 * When updating the kafka request, it will:
 * * Empty the `PromotionStatus` field
 * * Empty the `PromotionDetails` field
 * * Assign the `DesiredBillingModel` to `ActualBillingModel`
 * * Update the `SubscriptionId` field
 */

type UpdateKafkaRequestAction struct {
	kafkaService services.KafkaService
}

func NewUpdateKafkaRequestAction(kafkaService services.KafkaService) chain.ReconcileAction[PromotionContext] {
	return &UpdateKafkaRequestAction{
		kafkaService: kafkaService,
	}
}

func (u UpdateKafkaRequestAction) PerformJob(kafkaRequest *dbapi.KafkaRequest, currentResult chain.ActionResult[PromotionContext]) (chain.ActionResult[PromotionContext], bool, error) {
	glog.Infof("cluster with ID '%s' promoted from '%s' to '%s'. Updating the database info", kafkaRequest.ClusterID, kafkaRequest.ActualKafkaBillingModel, kafkaRequest.DesiredKafkaBillingModel)
	// gorm ignores zero values, so to zero `PromotionStatus` and `PromotionDetails` we need to use a map
	updates := map[string]any{}
	updates["promotion_status"] = ""
	updates["promotion_details"] = ""
	updates["actual_kafka_billing_model"] = kafkaRequest.DesiredKafkaBillingModel
	// current result must have the subscription id at this point
	updates["subscription_id"] = currentResult.Value().SubscriptionID

	updates["expires_at"] = sql.NullTime{
		Time:  time.Time{},
		Valid: false,
	}

	err := u.kafkaService.Updates(kafkaRequest, updates)
	if err != nil {
		// we need to mark the error as recoverable so that the reconciler will keep on retrying
		return currentResult, true, errors.NewServiceErrorBuilder().Wrap(*err).Recoverable().Build()
	}
	return currentResult, false, nil
}
