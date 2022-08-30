package quota

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cloudproviders"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

type amsQuotaService struct {
	amsClient   ocm.AMSClient
	kafkaConfig *config.KafkaConfig
}

const (
	CloudProviderAWS   = "aws"
	CloudProviderRHM   = "rhm"
	CloudProviderAzure = "azure"
)

const (
	amsReservedResourceResourceTypeClusterAWS = "cluster.aws"
	amsReservedResourceResourceTypeClusterGCP = "cluster.gcp"

	amsClusterAuthorizationRequestAvailabilityZoneSingle = "single"
	amsClusterAuthorizationRequestAvailabilityZoneMulti  = "multi"
)

var supportedCloudProviders = []string{CloudProviderAWS, CloudProviderRHM, CloudProviderAzure}

var supportedMarketplaceBillingModels = []string{
	string(amsv1.BillingModelMarketplace),
	string(amsv1.BillingModelMarketplaceAWS),
	string(amsv1.BillingModelMarketplaceRHM),
	string(amsv1.BillingModelMarketplaceAzure),
}

func getMarketplaceBillingModelForCloudProvider(cloudProvider string) (amsv1.BillingModel, error) {
	switch cloudProvider {
	case CloudProviderAWS:
		return amsv1.BillingModelMarketplaceAWS, nil
	case CloudProviderRHM:
		return amsv1.BillingModelMarketplace, nil
	case CloudProviderAzure:
		return amsv1.BillingModelMarketplaceAzure, nil
	}

	return "", errors.InvalidBillingAccount("unsupported cloud provider")
}

func (q amsQuotaService) getAMSReservedResourceResourceType(cloudProviderID cloudproviders.CloudProviderID) string {
	switch cloudProviderID {
	case cloudproviders.AWS:
		return amsReservedResourceResourceTypeClusterAWS
	case cloudproviders.GCP:
		return amsReservedResourceResourceTypeClusterGCP
	default:
		return ""
	}
}

func (q amsQuotaService) getAMSClusterAuthorizationRequestAvailabilityZone(multiAZ bool) string {
	if multiAZ {
		return amsClusterAuthorizationRequestAvailabilityZoneMulti
	}

	return amsClusterAuthorizationRequestAvailabilityZoneSingle
}

func (q amsQuotaService) newBaseQuotaReservedResourceBuilder(kafka *dbapi.KafkaRequest) amsv1.ReservedResourceBuilder {
	rr := amsv1.NewReservedResource()
	kafkaCloudProviderID := cloudproviders.ParseCloudProviderID(kafka.CloudProvider)
	resourceType := q.getAMSReservedResourceResourceType(kafkaCloudProviderID)
	if resourceType != "" {
		rr.ResourceType(resourceType)
	}
	rr.ResourceName(ocm.RHOSAKResourceName)
	rr.BillingModel(amsv1.BillingModelMarketplace)
	rr.Count(1)
	return *rr
}

var supportedAMSBillingModels map[string]struct{} = map[string]struct{}{
	string(amsv1.BillingModelMarketplace): {},
	string(amsv1.BillingModelStandard):    {},
}

// checks if the requested billing model (pre-paid vs. consumption based) matches with the model returned from AMS
// returns a boolean indicating the match and a string value indicating if the computed billing model matched to either
// marketplace or standard billing
func (q amsQuotaService) billingModelMatches(computedBillingModel string, requestedBillingModel string) (bool, string) {
	// billing model is an optional parameter, infer the value from AMS if not provided
	if requestedBillingModel == "" && computedBillingModel == string(amsv1.BillingModelStandard) {
		return true, string(amsv1.BillingModelStandard)
	}

	if requestedBillingModel == "" && arrays.Contains(supportedMarketplaceBillingModels, computedBillingModel) {
		return true, string(amsv1.BillingModelMarketplace)
	}

	// user requested pre-paid billing and it matches the computed billing model
	if computedBillingModel == string(amsv1.BillingModelStandard) && requestedBillingModel == string(amsv1.BillingModelStandard) {
		return true, string(amsv1.BillingModelStandard)
	}

	// user requested consumption based billing and it matches the computed billing model
	if arrays.Contains(supportedMarketplaceBillingModels, computedBillingModel) && requestedBillingModel == string(amsv1.BillingModelMarketplace) {
		return true, string(amsv1.BillingModelMarketplace)
	}

	// computed and requested billing models do not match
	return false, ""
}

func (q amsQuotaService) ValidateBillingAccount(organisationId string, instanceType types.KafkaInstanceType, billingCloudAccountId string, marketplace *string) *errors.ServiceError {
	orgId, err := q.amsClient.GetOrganisationIdFromExternalId(organisationId)
	if err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, fmt.Sprintf("Error checking quota: failed to get organization with external id %v", organisationId))
	}

	quotaType := instanceType.GetQuotaType()
	quotaCosts, err := q.amsClient.GetQuotaCostsForProduct(orgId, quotaType.GetResourceName(), quotaType.GetProduct())
	if err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, fmt.Sprintf("Error checking quota: failed to get assigned quota of type %v for organization with id %v", instanceType.GetQuotaType(), orgId))
	}

	var matchingBillingAccounts = 0
	var billingAccounts []amsv1.CloudAccount

	for _, quotaCost := range quotaCosts {
		for _, cloudAccount := range quotaCost.CloudAccounts() {
			billingAccounts = append(billingAccounts, *cloudAccount)
			if cloudAccount.CloudAccountID() == billingCloudAccountId {
				if marketplace != nil && *marketplace != cloudAccount.CloudProviderID() {
					continue
				}

				// matching billing account found
				matchingBillingAccounts++
			}
		}
	}

	if len(billingAccounts) == 0 {
		return errors.InvalidBillingAccount("No billing accounts available in quota")
	}

	// only one matching billing account is expected. If there are multiple then
	// they are with different cloud providers
	if matchingBillingAccounts > 1 {
		return errors.InvalidBillingAccount("Multiple matching billing accounts found, only one expected. Available billing accounts: %v", billingAccounts)
	}
	if matchingBillingAccounts == 0 {
		return errors.InvalidBillingAccount("No matching billing account found. Provided: %s, Available: %v", billingCloudAccountId, billingAccounts)
	}

	// we found one and only one matching billing account
	return nil
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
//   - Matches the given input quotaType
//   - Contains at least one AMS RelatedResources whose billing model is one
//     of the supported Billing Models specified in supportedAMSBillingModels
//   - Has an "Allowed" value greater than 0
//
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
				}
				foundUnsupportedBillingModel = rr.BillingModel()
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
func (q amsQuotaService) getAvailableBillingModelFromKafkaInstanceType(orgId string, instanceType types.KafkaInstanceType, sizeRequired int) (string, error) {
	quotaCosts, err := q.amsClient.GetQuotaCostsForProduct(orgId, instanceType.GetQuotaType().GetResourceName(), instanceType.GetQuotaType().GetProduct())
	if err != nil {
		return "", errors.InsufficientQuotaError("%v: error getting quotas for product %s", err, instanceType.GetQuotaType().GetProduct())
	}

	billingModel := ""
	for _, qc := range quotaCosts {
		for _, rr := range qc.RelatedResources() {
			if qc.Consumed()+sizeRequired <= qc.Allowed() || rr.Cost() == 0 {
				switch rr.BillingModel() {
				case string(amsv1.BillingModelStandard):
					return rr.BillingModel(), nil
				case string(amsv1.BillingModelMarketplace):
					billingModel = rr.BillingModel()
				}
			}
		}
	}

	return billingModel, nil
}

func (q amsQuotaService) getBillingModel(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (string, error) {
	orgId, err := q.amsClient.GetOrganisationIdFromExternalId(kafka.OrganisationId)
	if err != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, err, fmt.Sprintf("Error checking quota: failed to get organization with external id %v", orgId))
	}

	kafkaInstanceSize, err := q.kafkaConfig.GetKafkaInstanceSize(kafka.InstanceType, kafka.SizeId)
	if err != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, err, "Error reserving quota")
	}

	quotaCosts, err := q.amsClient.GetQuotaCostsForProduct(orgId, instanceType.GetQuotaType().GetResourceName(), instanceType.GetQuotaType().GetProduct())
	if err != nil {
		return "", errors.InsufficientQuotaError("%v: error getting quotas for product %s", err, instanceType.GetQuotaType().GetProduct())
	}

	getCloudAccounts := func() []*amsv1.CloudAccount {
		var accounts []*amsv1.CloudAccount
		for _, quotaCost := range quotaCosts {
			accounts = append(accounts, quotaCost.CloudAccounts()...)
		}
		return accounts
	}

	// check if it there is a related resource that supports the marketplace billing model and has quota
	hasSufficientMarketplaceQuota := func() bool {
		for _, quotaCost := range quotaCosts {
			for _, rr := range quotaCost.RelatedResources() {
				if rr.BillingModel() == string(amsv1.BillingModelMarketplace) &&
					(rr.Cost() == 0 || quotaCost.Consumed()+kafkaInstanceSize.QuotaConsumed <= quotaCost.Allowed()) {
					return true
				}
			}
		}
		return false
	}

	// no billing account and marketplace provided
	// in this case we first try to assign the billing model in the old way: prefer standard and use marketplace if standard is not available
	// if marketplace was picked and there is a single cloud account of type aws, then we change the billing model to marketplace-aws and
	// assign the account id
	if kafka.BillingCloudAccountId == "" && kafka.Marketplace == "" {
		billingModel, err := q.getAvailableBillingModelFromKafkaInstanceType(orgId, instanceType, kafkaInstanceSize.CapacityConsumed)
		if err != nil {
			return "", err
		}

		cloudAccounts := getCloudAccounts()
		if billingModel == string(amsv1.BillingModelMarketplace) && len(cloudAccounts) == 1 {
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

		} else {
			// billing model was standard or can't choose a cloud account
			return billingModel, nil
		}
	}

	// billing account and marketplace (optional) provided
	// find a matching cloud account and assign it if the provider id is aws (the only supported one at the moment)
	for _, cloudAccount := range getCloudAccounts() {
		if cloudAccount.CloudAccountID() == kafka.BillingCloudAccountId && (cloudAccount.CloudProviderID() == kafka.Marketplace || kafka.Marketplace == "") {
			// check if we have quota
			if !hasSufficientMarketplaceQuota() {
				return "", errors.InsufficientQuotaError("quota does not support marketplace billing or has insufficient quota")
			}

			// assign marketplace in case it was not provided in the request
			kafka.Marketplace = cloudAccount.CloudProviderID()
			billingModel, err := getMarketplaceBillingModelForCloudProvider(cloudAccount.CloudProviderID())
			if err != nil {
				return "", err
			}

			return string(billingModel), nil
		}
	}

	return "", errors.InsufficientQuotaError("no matching marketplace quota found for product %s", instanceType.GetQuotaType().GetProduct())
}

func (q amsQuotaService) ReserveQuota(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (string, *errors.ServiceError) {
	kafkaId := kafka.ID

	rr := q.newBaseQuotaReservedResourceBuilder(kafka)

	kafkaInstanceSize, e := q.kafkaConfig.GetKafkaInstanceSize(kafka.InstanceType, kafka.SizeId)
	if e != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, e, "Error reserving quota")
	}

	bm, err := q.getBillingModel(kafka, instanceType)
	if err != nil {
		svcErr := errors.ToServiceError(err)
		return "", errors.NewWithCause(svcErr.Code, svcErr, "Error getting billing model")
	}
	if bm == "" {
		return "", errors.InsufficientQuotaError("Error getting billing model: No available billing model found")
	}

	bmMatched, matchedBillingModel := q.billingModelMatches(bm, kafka.BillingModel)
	if !bmMatched {
		return "", errors.InvalidBillingAccount("requested billing model does not match assigned. requested: %s, assigned: %s", kafka.BillingModel, bm)
	}
	kafka.BillingModel = matchedBillingModel

	rr.BillingModel(amsv1.BillingModel(bm))
	rr.Count(kafkaInstanceSize.QuotaConsumed)

	// will be empty if no marketplace account is used
	rr.BillingMarketplaceAccount(kafka.BillingCloudAccountId)

	cb, _ := amsv1.NewClusterAuthorizationRequest().
		AccountUsername(kafka.Owner).
		CloudProviderID(kafka.CloudProvider).
		ProductID(instanceType.GetQuotaType().GetProduct()).
		Managed(true).
		ClusterID(kafkaId).
		ExternalClusterID(kafkaId).
		Disconnected(false).
		BYOC(false).
		AvailabilityZone(q.getAMSClusterAuthorizationRequestAvailabilityZone(kafka.MultiAZ)).
		Reserve(true).
		Resources(&rr).
		Build()

	resp, err := q.amsClient.ClusterAuthorization(cb)
	if err != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, err, "Error reserving quota")
	}

	if !resp.Allowed() {
		return "", errors.InsufficientQuotaError("Insufficient Quota")
	}

	return resp.Subscription().ID(), nil
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
