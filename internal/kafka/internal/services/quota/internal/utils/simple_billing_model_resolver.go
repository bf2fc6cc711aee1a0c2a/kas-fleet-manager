package utils

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

/********************************************************************************************************************
 * This resolver is used when the user didn't specify the `billing model` and the `marketplace` in the kafka request.
 * The billing model is resolved this way: if the user has quota for `standard`, it returns `standard`. If he has
 * quota for `marketplace` it returns `marketplace`.
 *
 * Attempts are done to resolve the marketplace type too.
 ********************************************************************************************************************/

var _ BillingModelResolver = &simpleBillingModelResolver{}

type simpleBillingModelResolver struct {
	quotaConfigProvider AMSQuotaConfigProvider
	kafkaConfig         *config.KafkaConfig
}

func (*simpleBillingModelResolver) SupportRequest(kafka *dbapi.KafkaRequest) bool {
	return kafka.BillingCloudAccountId == "" && kafka.Marketplace == ""
}

// Resolve - tries to resolve the KafkaBillingModel and amsBillingModel by analysing the parameter received by the user.
// This function checks if for the requested instance type the `standard` and `marketplace` billing model are defined
// and produces a sorted array contained those billing models (if defined) in the priority order (standard first,
// marketplace second), then demands the resolving logic to the `resolve` method.
func (r *simpleBillingModelResolver) Resolve(orgID string, kafka *dbapi.KafkaRequest) (BillingModelDetails, error) {
	// try standard and marketplace
	var kafkaBillingModels []config.KafkaBillingModel
	standardBillingModel, err := r.kafkaConfig.GetBillingModelByID(kafka.InstanceType, "standard")
	if err == nil {
		kafkaBillingModels = append(kafkaBillingModels, standardBillingModel)
	}
	marketplaceBillingModel, err := r.kafkaConfig.GetBillingModelByID(kafka.InstanceType, "marketplace")
	if err == nil {
		kafkaBillingModels = append(kafkaBillingModels, marketplaceBillingModel)
	}

	return r.resolve(orgID, kafka, kafkaBillingModels)
}

// resolve - This method tries to resolve the billing model and, eventually, the marketplace type for the received
// kafka request among the received kafkaBillingModels. The billing models in the `kafkaBillingModel` array are evaluated
// in the received order, so that an order of preference can be honored.
// Parameters:
// orgID: The organisation ID
// kafka: The kafka request received by the user
// kafkaBillingModels: The list of kafka billing models to be considered, ordered by preference
func (r *simpleBillingModelResolver) resolve(orgID string, kafka *dbapi.KafkaRequest, kafkaBillingModels []config.KafkaBillingModel) (BillingModelDetails, error) {
	kafkaInstanceSize, err := r.kafkaConfig.GetKafkaInstanceSize(kafka.InstanceType, kafka.SizeId)
	if err != nil {
		return BillingModelDetails{}, errors.NewWithCause(errors.ErrorGeneral, err, "Error reserving quota")
	}

	return r.resolve(orgId, kafka, kafkaBillingModels)
}

// resolve - This method tries to resolve the billing model and, eventually, the marketplace type for the received
// kafka request among the received kafkaBillingModels. The billing models in the `kafkaBillingModel` array are evaluated
// in the received order, so that an order of preference can be honored.
// Parameters:
// orgId: The organisation ID
// kafka: The kafka request received by the user
// kafkaBillingModels: The list of kafka billing models to be considered, ordered by preference
func (r *simpleBillingModelResolver) resolve(orgId string, kafka *dbapi.KafkaRequest, kafkaBillingModels []config.KafkaBillingModel) (BillingModelDetails, error) {
	kafkaInstanceSize, err := r.kafkaConfig.GetKafkaInstanceSize(kafka.InstanceType, kafka.SizeId)
	if err != nil {
		return BillingModelDetails{}, errors.NewWithCause(errors.ErrorGeneral, err, "Error reserving quota")
	}

	for _, kbm := range kafkaBillingModels {
		available, amsBillingModel, err := r.isBillingModelAvailable(orgID, kbm, kafkaInstanceSize.CapacityConsumed)
		if err != nil {
			return BillingModelDetails{}, err
		}
		if available {
			// we found the details. now we can refine the `AMSBillingModel` if it is a marketplace
			if kbm.HasSupportForMarketplace() { // we have only standard and marketplace in this resolver
				amsBillingModel, err = r.resolveMarketplaceType(orgID, kafka, kbm, amsBillingModel)
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

	return BillingModelDetails{}, errors.InsufficientQuotaError("no quota available for any of the matched kafka billing models %v",
		arrays.Map(kafkaBillingModels, func(kbm config.KafkaBillingModel) string { return kbm.ID }),
	)
}

func (r *simpleBillingModelResolver) resolveMarketplaceType(orgID string, kafka *dbapi.KafkaRequest, kafkaBillingModel config.KafkaBillingModel, amsBillingModel string) (string, error) {
	quotaCosts, err := r.quotaConfigProvider.GetQuotaCostsForProduct(orgID, kafkaBillingModel.AMSResource, kafkaBillingModel.AMSProduct)
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

// isBillingModelAvailable - checks if the received billing model is available for the specified orgID.
// A billing model is considered available if all the following conditions are true:
// 1) Quota costs are defined for the triplet `orgID, resource, product`
// 2) At least one such quota cost has enough available quota to fit the requested instance size
// 3) At least one of the quota costs detected at point 2 has a billing model supported by the received `kafkaBillingModel` object.
// If more than one quota_cost satisfies all the previous points, the first one is returned.
func (r *simpleBillingModelResolver) isBillingModelAvailable(orgID string, kafkaBillingModel config.KafkaBillingModel, requiredSize int) (bool, string, error) {
	quotaCosts, err := r.quotaConfigProvider.GetQuotaCostsForProduct(orgID, kafkaBillingModel.AMSResource, kafkaBillingModel.AMSProduct)
	if err != nil {
		return false, "", errors.InsufficientQuotaError("%v: error getting quotas for product %s", err, kafkaBillingModel.AMSProduct)
	}

	isRelatedResourceMatch := func(qc *amsv1.QuotaCost, rr *amsv1.RelatedResource, kafkaBillingModel *config.KafkaBillingModel) bool {
		return (qc.Consumed()+requiredSize <= qc.Allowed() || rr.Cost() == 0) &&
			kafkaBillingModel.HasSupportForAMSBillingModel(rr.BillingModel())
	}

	for _, qc := range quotaCosts {
		if idx, rr := arrays.FindFirst(qc.RelatedResources(), func(rr *amsv1.RelatedResource) bool { return isRelatedResourceMatch(qc, rr, &kafkaBillingModel) }); idx != -1 {
			return true, rr.BillingModel(), nil
		}
	}

	return false, "", nil
}
