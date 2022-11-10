package utils

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

var _ BillingModelResolver = &simpleBillingModelResolver{}

type simpleBillingModelResolver struct {
	quotaConfigProvider AMSQuotaConfigProvider
	kafkaConfig         *config.KafkaConfig
}

func (s *simpleBillingModelResolver) SupportRequest(kafka *dbapi.KafkaRequest) bool {
	return kafka.BillingCloudAccountId == "" && kafka.Marketplace == ""
}

func (r *simpleBillingModelResolver) Resolve(orgId string, kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (BillingModelDetails, error) {
	kafkaInstanceSize, err := r.kafkaConfig.GetKafkaInstanceSize(kafka.InstanceType, kafka.SizeId)
	if err != nil {
		return BillingModelDetails{}, errors.NewWithCause(errors.ErrorGeneral, err, "Error reserving quota")
	}

	// try standard and marketplace
	var kafkaBillingModels []config.KafkaBillingModel
	standardBillingModel, err := r.kafkaConfig.GetBillingModelByID(instanceType.String(), "standard")
	if err == nil {
		kafkaBillingModels = append(kafkaBillingModels, standardBillingModel)
	}
	marketplaceBillingModel, err := r.kafkaConfig.GetBillingModelByID(instanceType.String(), "marketplace")
	if err == nil {
		kafkaBillingModels = append(kafkaBillingModels, marketplaceBillingModel)
	}

	for _, kbm := range kafkaBillingModels {
		available, amsBillingModel, err := r.isBillingModelAvailable(orgId, kbm, kafkaInstanceSize.CapacityConsumed)
		if err != nil {
			return BillingModelDetails{}, err
		}
		if available {
			// we found the details. now we can refine the `AMSBillingModel` if it is a marketplace
			if kbm.HasSupportForMarketplace() { // we have only standard and marketplace in this resolver
				amsBillingModel, err = r.resolveMarketplaceType(orgId, kafka, kbm, amsBillingModel)
				if err != nil {
					return BillingModelDetails{}, err
				}
			}

			return BillingModelDetails{
				KafkaBillingModel: kbm,
				AMSBillingModel:   amsBillingModel,
			}, nil
		}
	}

	return BillingModelDetails{}, errors.InsufficientQuotaError("unable to resolve billing model")
}

func (r *simpleBillingModelResolver) resolveMarketplaceType(orgId string, kafka *dbapi.KafkaRequest, kafkaBillingModel config.KafkaBillingModel, amsBillingModel string) (string, error) {
	quotaCosts, err := r.quotaConfigProvider.GetQuotaCostsForProduct(orgId, kafkaBillingModel.AMSResource, kafkaBillingModel.AMSProduct)
	if err != nil {
		return "", errors.InsufficientQuotaError("%v: error getting quotas for product %s", err, kafkaBillingModel.AMSProduct)
	}

	cloudAccounts := getCloudAccounts(quotaCosts)
	if amsBillingModel == string(amsv1.BillingModelMarketplace) && len(cloudAccounts) == 1 {
		// at this point we know that there is only one cloud account
		cloudAccount := cloudAccounts[0]

		if arrays.Contains(supportedCloudProviders, cloudAccount.CloudProviderID()) {
			kafka.BillingCloudAccountId = cloudAccount.CloudAccountID()
			kafka.Marketplace = cloudAccount.CloudProviderID()

			billingModel, err := getMarketplaceBillingModelForCloudProvider(cloudAccount.CloudProviderID())
			if err != nil {
				return "", err
			}

			return string(billingModel), nil
		}

	}

	// billing model was standard or can't choose a cloud account
	return amsBillingModel, nil
}

func (r *simpleBillingModelResolver) isBillingModelAvailable(orgId string, kafkaBillingModel config.KafkaBillingModel, requiredSize int) (bool, string, error) {
	quotaCosts, err := r.quotaConfigProvider.GetQuotaCostsForProduct(orgId, kafkaBillingModel.AMSResource, kafkaBillingModel.AMSProduct)
	if err != nil {
		return false, "", errors.InsufficientQuotaError("%v: error getting quotas for product %s", err, kafkaBillingModel.AMSProduct)
	}

	for _, qc := range quotaCosts {
		for _, rr := range qc.RelatedResources() {
			if qc.Consumed()+requiredSize <= qc.Allowed() || rr.Cost() == 0 {
				if kafkaBillingModel.HasSupportForAMSBillingModel(rr.BillingModel()) {
					return true, rr.BillingModel(), nil
				}
			}
		}
	}

	return false, "", nil
}
