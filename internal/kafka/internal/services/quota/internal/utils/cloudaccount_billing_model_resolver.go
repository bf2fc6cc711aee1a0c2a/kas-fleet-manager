package utils

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

var _ BillingModelResolver = &cloudAccountBillingModelResolver{}

type cloudAccountBillingModelResolver struct {
	quotaConfigProvider AMSQuotaConfigProvider
	kafkaConfig         *config.KafkaConfig
}

func (r *cloudAccountBillingModelResolver) SupportRequest(kafka *dbapi.KafkaRequest) bool {
	//_, err := r.kafkaConfig.GetBillingModelByID(kafka.InstanceType, "marketplace")
	//if err != nil {
	//	return false
	//}
	return kafka.BillingCloudAccountId != ""
}

func (r *cloudAccountBillingModelResolver) Resolve(orgId string, kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (BillingModelDetails, error) {
	kafkaBillingModel, err := r.kafkaConfig.GetBillingModelByID(instanceType.String(), "marketplace")
	if err != nil {
		return BillingModelDetails{}, err
	}

	kafkaInstanceSize, err := r.kafkaConfig.GetKafkaInstanceSize(kafka.InstanceType, kafka.SizeId)
	if err != nil {
		return BillingModelDetails{}, errors.NewWithCause(errors.ErrorGeneral, err, "Error reserving quota")
	}

	quotaCosts, err := r.quotaConfigProvider.GetQuotaCostsForProduct(orgId, kafkaBillingModel.AMSResource, kafkaBillingModel.AMSProduct)
	for _, cloudAccount := range getCloudAccounts(quotaCosts) {
		if cloudAccount.CloudAccountID() == kafka.BillingCloudAccountId && (cloudAccount.CloudProviderID() == kafka.Marketplace || kafka.Marketplace == "") {
			// check if we have quota
			if !hasSufficientMarketplaceQuota(quotaCosts, kafkaInstanceSize.QuotaConsumed) {
				return BillingModelDetails{}, errors.InsufficientQuotaError("quota does not support marketplace billing or has insufficient quota")
			}

			// assign marketplace in case it was not provided in the request
			kafka.Marketplace = cloudAccount.CloudProviderID()
			billingModel, err := getMarketplaceBillingModelForCloudProvider(cloudAccount.CloudProviderID())
			if err != nil {
				return BillingModelDetails{}, err
			}

			return BillingModelDetails{
				KafkaBillingModel: kafkaBillingModel,
				AMSBillingModel:   string(billingModel),
			}, nil
		}
	}

	return BillingModelDetails{}, errors.InsufficientQuotaError("no matching marketplace quota found for product %s", kafkaBillingModel.AMSProduct)
}
