package utils

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

/********************************************************************************************************************
 * This resolver is used when the user specifies a `Cloud Account ID` in the request.
 * The AMS Billing Model is then detected by checking what is the marketplace supported by that account.
 ********************************************************************************************************************/

var _ BillingModelResolver = &cloudAccountBillingModelResolver{}

type cloudAccountBillingModelResolver struct {
	quotaConfigProvider AMSQuotaConfigProvider
	kafkaConfig         *config.KafkaConfig
}

func (r *cloudAccountBillingModelResolver) SupportRequest(kafka *dbapi.KafkaRequest) bool {
	return kafka.BillingCloudAccountId != ""
}

func (r *cloudAccountBillingModelResolver) Resolve(orgID string, kafka *dbapi.KafkaRequest) (BillingModelDetails, error) {
	kafkaBillingModel, err := r.kafkaConfig.GetBillingModelByID(kafka.InstanceType, "marketplace")
	if err != nil {
		return BillingModelDetails{}, err
	}

	return r.resolve(orgID, kafka, kafkaBillingModel)
}

// resolve - tries to resolve the AMS Billing Model by performing the following checks:
// 1) Tries to retrieve the cloud account. A failure here will result in an error.
// 2) Checks if the user has sufficient marketplace quota. If not, an error is returned.
// 3) Tries to get the marketplace associated with the cloud provider (if not already provided in the request).
// 4) If the kafka billing model supports the marketplace detected in [3], we have found id, otherwise an error is returned.
func (r *cloudAccountBillingModelResolver) resolve(orgID string, kafka *dbapi.KafkaRequest, kafkaBillingModel config.KafkaBillingModel) (BillingModelDetails, error) {
	kafkaInstanceSize, err := r.kafkaConfig.GetKafkaInstanceSize(kafka.InstanceType, kafka.SizeId)
	if err != nil {
		return BillingModelDetails{}, errors.NewWithCause(errors.ErrorGeneral, err, "error reserving quota")
	}

	quotaCosts, err := r.quotaConfigProvider.GetQuotaCostsForProduct(orgID, kafkaBillingModel.AMSResource, kafkaBillingModel.AMSProduct)
	if err != nil {
		return BillingModelDetails{}, errors.InsufficientQuotaError("%v: error getting quotas for product %s", err, kafkaBillingModel.AMSProduct)
	}

	idx, cloudAccount := arrays.FindFirst(getCloudAccounts(quotaCosts), func(cloudAccount *v1.CloudAccount) bool {
		return cloudAccount.CloudAccountID() == kafka.BillingCloudAccountId &&
			(kafka.Marketplace == "" || cloudAccount.CloudProviderID() == kafka.Marketplace)
	})

	if idx != -1 {
		if !hasSufficientMarketplaceQuota(quotaCosts, kafkaInstanceSize.QuotaConsumed) {
			return BillingModelDetails{}, errors.InsufficientQuotaError("quota does not support marketplace billing or has insufficient quota")
		}

		// assign marketplace in case it was not provided in the request
		kafka.Marketplace = cloudAccount.CloudProviderID()
		billingModel, err := getMarketplaceBillingModelForCloudProvider(cloudAccount.CloudProviderID())
		if err != nil {
			return BillingModelDetails{}, err
		}

		if !kafkaBillingModel.HasSupportForAMSBillingModel(string(billingModel)) {
			return BillingModelDetails{}, errors.InsufficientQuotaError("specified cloud account supports %s, while requested kafka billing model (%s) only supports %v",
				kafkaBillingModel.AMSProduct, kafkaBillingModel.ID, kafkaBillingModel.AMSBillingModels)
		}
		return BillingModelDetails{
			KafkaBillingModel: kafkaBillingModel,
			AMSBillingModel:   string(billingModel),
		}, nil
	}

	return BillingModelDetails{}, errors.InsufficientQuotaError("no matching marketplace quota found for product %s", kafkaBillingModel.AMSProduct)
}
