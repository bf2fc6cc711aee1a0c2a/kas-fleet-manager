package quota

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

type amsQuotaService struct {
	amsClient   ocm.AMSClient
	kafkaConfig *config.KafkaConfig
}

func newBaseQuotaReservedResourceResourceBuilder() amsv1.ReservedResourceBuilder {
	rr := amsv1.ReservedResourceBuilder{}
	rr.ResourceType("cluster.aws") //cluster.aws
	rr.BYOC(false)                 //false
	rr.ResourceName("rhosak")      //"rhosak"
	rr.BillingModel("marketplace") // "marketplace" or "standard"
	rr.AvailabilityZoneType("single")
	rr.Count(1)
	return rr
}

var supportedAMSBillingModels map[string]struct{} = map[string]struct{}{
	string(amsv1.BillingModelMarketplace):    {},
	string(amsv1.BillingModelStandard):       {},
	string(amsv1.BillingModelMarketplaceAWS): {},
}

func (q amsQuotaService) GetMarketplaceFromBillingAccountInformation(externalId string, instanceType types.KafkaInstanceType, billingCloudAccountId string, marketplace *string) (string, *errors.ServiceError) {
	orgId, err := q.amsClient.GetOrganisationIdFromExternalId(externalId)
	if err != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, err, fmt.Sprintf("Error checking quota: failed to get organization with external id %v", externalId))
	}

	quotaType := instanceType.GetQuotaType()
	quotaCosts, err := q.amsClient.GetQuotaCostsForProduct(orgId, quotaType.GetResourceName(), quotaType.GetProduct())
	if err != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, err, fmt.Sprintf("Error checking quota: failed to get assigned quota of type %v for organization with id %v", instanceType.GetQuotaType(), orgId))
	}

	var matchingBillingAccounts = 0
	var billingAccounts []amsv1.CloudAccount
	var matchedMarketplace string

	for _, quotaCost := range quotaCosts {
		for _, cloudAccount := range quotaCost.CloudAccounts() {
			billingAccounts = append(billingAccounts, *cloudAccount)
			if cloudAccount.CloudAccountID() == billingCloudAccountId {
				if marketplace != nil && *marketplace != cloudAccount.CloudProviderID() {
					continue
				}

				// matching billing account found
				matchingBillingAccounts++
				matchedMarketplace = cloudAccount.CloudProviderID()
			}
		}
	}

	// only one matching billing account is expected. If there are multiple then they are with different
	// cloud providers
	if matchingBillingAccounts == 1 {
		return matchedMarketplace, nil
	} else if matchingBillingAccounts > 1 {
		return "", errors.InvalidBillingAccount("Multiple matching billing accounts found, only one expected. Available billing accounts: %v", billingAccounts)
	}

	if len(billingAccounts) == 0 {
		return "", errors.InvalidBillingAccount("No billing accounts available in quota")
	}

	return "", errors.InvalidBillingAccount("No matching billing account found. Provided: %s, Available: %v", billingCloudAccountId, billingAccounts)
}

func (q amsQuotaService) CheckIfQuotaIsDefinedForInstanceType(username string, externalId string, instanceType types.KafkaInstanceType) (bool, *errors.ServiceError) {
	orgId, err := q.amsClient.GetOrganisationIdFromExternalId(externalId)
	if err != nil {
		return false, errors.NewWithCause(errors.ErrorGeneral, err, fmt.Sprintf("Error checking quota: failed to get organization with external id %v", externalId))
	}

	hasQuota, err := q.hasConfiguredQuotaCost(orgId, instanceType.GetQuotaType())
	if err != nil {
		return false, errors.NewWithCause(errors.ErrorGeneral, err, fmt.Sprintf("Error checking quota: failed to get assigned quota of type %v for organization with id %v", instanceType.GetQuotaType(), orgId))
	}

	return hasQuota, nil
}

// hasConfiguredQuotaCost returns true if the given organizationID has at least
// one AMS QuotaCost that complies with the following conditions:
// - Matches the given input quotaType
// - Contains at least one AMS RelatedResources whose billing model is one
//   of the supported Billing Models specified in supportedAMSBillingModels
// - Has an "Allowed" value greater than 0
// An error is returned if the given organizationID has a QuotaCost
// with an unsupported billing model and there are no supported billing models
func (q amsQuotaService) hasConfiguredQuotaCost(organizationID string, quotaType ocm.KafkaQuotaType) (bool, error) {
	quotaCosts, err := q.amsClient.GetQuotaCostsForProduct(organizationID, quotaType.GetResourceName(), quotaType.GetProduct())
	if err != nil {
		return false, err
	}

	var foundUnsupportedBillingModel string

	for _, qc := range quotaCosts {
		if qc.Allowed() > 0 {
			for _, rr := range qc.RelatedResources() {
				if _, isCompatibleBillingModel := supportedAMSBillingModels[rr.BillingModel()]; isCompatibleBillingModel {
					return true, nil
				} else {
					foundUnsupportedBillingModel = rr.BillingModel()
				}
			}
		}
	}

	if foundUnsupportedBillingModel != "" {
		return false, errors.GeneralError("Product %s only has unsupported allowed billing models. Last one found: %s", quotaType.GetProduct(), foundUnsupportedBillingModel)
	}

	return false, nil
}

// getAvailableBillingModelFromKafkaInstanceType gets the billing model of a
// kafka instance type by looking at the resource name and product of the
// instanceType. Only QuotaCosts that have available quota, or that contain a
// RelatedResource with "cost" 0 are considered. Only
// "standard" and "marketplace" billing models are considered. If both are
// detected "standard" is returned.
func (q amsQuotaService) getAvailableBillingModelFromKafkaInstanceType(externalID string, instanceType types.KafkaInstanceType, sizeRequired int, marketplace string, billingCloudAccountId string) (string, error) {
	orgId, err := q.amsClient.GetOrganisationIdFromExternalId(externalID)
	if err != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, err, fmt.Sprintf("Error checking quota: failed to get organization with external id %v", externalID))
	}

	quotaCosts, err := q.amsClient.GetQuotaCostsForProduct(orgId, instanceType.GetQuotaType().GetResourceName(), instanceType.GetQuotaType().GetProduct())
	if err != nil {
		return "", errors.InsufficientQuotaError("%v: error getting quotas for product %s", err, instanceType.GetQuotaType().GetProduct())
	}

	billingModel := ""
	matchedBillingModels := 0
	for _, qc := range quotaCosts {
		foundBillingCloudAccount := false
		if marketplace != "" {
			for _, ca := range qc.CloudAccounts() {
				if billingCloudAccountId == ca.CloudAccountID() {
					foundBillingCloudAccount = true
					break
				}
			}
		}
		if !foundBillingCloudAccount {
			continue
		}
		for _, rr := range qc.RelatedResources() {
			if qc.Consumed()+sizeRequired <= qc.Allowed() || rr.Cost() == 0 {
				if marketplace != "" {
					if marketplace == "aws" && rr.BillingModel() == string(amsv1.BillingModelMarketplaceAWS) {
						return rr.BillingModel(), nil
					}
				} else {
					if rr.BillingModel() == string(amsv1.BillingModelStandard) {
						return rr.BillingModel(), nil
					} else if rr.BillingModel() == string(amsv1.BillingModelMarketplace) || rr.BillingModel() == string(amsv1.BillingModelMarketplaceAWS) {
						billingModel = rr.BillingModel()
						matchedBillingModels++
					}
				}
			}
		}
	}

	if matchedBillingModels > 1 {
		return "", errors.New(errors.ErrorGeneral, "more than one billing model detected for the kafka instance")
	}

	return billingModel, nil
}

func (q amsQuotaService) ReserveQuota(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (string, *errors.ServiceError) {
	kafkaId := kafka.ID

	rr := newBaseQuotaReservedResourceResourceBuilder()

	kafkaInstanceSize, e := q.kafkaConfig.GetKafkaInstanceSize(kafka.InstanceType, kafka.SizeId)
	if e != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, e, "Error reserving quota")
	}

	bm, err := q.getAvailableBillingModelFromKafkaInstanceType(kafka.OrganisationId, instanceType, kafkaInstanceSize.CapacityConsumed, kafka.Marketplace, kafka.BillingCloudAccountId)
	if err != nil {
		svcErr := errors.ToServiceError(err)
		return "", errors.NewWithCause(svcErr.Code, svcErr, "Error getting billing model")
	}
	if bm == "" {
		return "", errors.InsufficientQuotaError("Error getting billing model: No available billing model found")
	}
	rr.BillingModel(amsv1.BillingModel(bm))
	rr.Count(kafkaInstanceSize.QuotaConsumed)

	cb, _ := amsv1.NewClusterAuthorizationRequest().
		AccountUsername(kafka.Owner).
		CloudProviderID(kafka.CloudProvider).
		ProductID(instanceType.GetQuotaType().GetProduct()).
		Managed(true).
		ClusterID(kafkaId).
		ExternalClusterID(kafkaId).
		Disconnected(false).
		BYOC(false).
		AvailabilityZone("single").
		Reserve(true).
		Resources(&rr).
		CloudAccountID(kafka.BillingCloudAccountId).
		Build()

	resp, err := q.amsClient.ClusterAuthorization(cb)
	if err != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, err, "Error reserving quota")
	}

	if resp.Allowed() {
		return resp.Subscription().ID(), nil
	} else {
		return "", errors.InsufficientQuotaError("Insufficient Quota")
	}
}

func (q amsQuotaService) DeleteQuota(subscriptionId string) *errors.ServiceError {
	if subscriptionId == "" {
		return nil
	}

	_, err := q.amsClient.DeleteSubscription(subscriptionId)
	if err != nil {
		return errors.GeneralError("failed to delete the quota: %v", err)
	}
	return nil
}
