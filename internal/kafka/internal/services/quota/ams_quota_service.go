package quota

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"strings"

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

var _ services.QuotaService = &amsQuotaService{}

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

func (q amsQuotaService) newBaseQuotaReservedResourceBuilder(kafka *dbapi.KafkaRequest, kafkaBillingModel config.KafkaBillingModel) amsv1.ReservedResourceBuilder {
	rr := amsv1.NewReservedResource()
	kafkaCloudProviderID := cloudproviders.ParseCloudProviderID(kafka.CloudProvider)
	resourceType := q.getAMSReservedResourceResourceType(kafkaCloudProviderID)
	if resourceType != "" {
		rr.ResourceType(resourceType)
	}
	rr.ResourceName(kafkaBillingModel.AMSResource)
	rr.Count(1)
	return *rr
}

// supportedAMSRelatedResourceBillingModels returns the supported AMS billing
// models for AMS RelatedResources. An AMS RelatedResource is not to be confused
// with an AMS ReservedResource. The set of Billing models shown/accepted is
// different between them
var supportedAMSRelatedResourceBillingModels = map[string]struct{}{
	string(amsv1.BillingModelMarketplace): {},
	string(amsv1.BillingModelStandard):    {},
}

// checks if the requested billing model (pre-paid vs. consumption based) matches with the model returned from AMS
// returns a boolean indicating the match and a string value indicating if the computed billing model matched to either
// marketplace or standard billing
func (q amsQuotaService) billingModelMatches(computedBillingModel string, requestedBillingModel string, kafkaBillingModel config.KafkaBillingModel) (bool, string) {
	// billing model is an optional parameter, infer the value from AMS if not provided
	if requestedBillingModel == "" && computedBillingModel == string(amsv1.BillingModelStandard) {
		return true, string(amsv1.BillingModelStandard)
	}

	if requestedBillingModel == "" && arrays.Contains(kafkaBillingModel.AMSBillingModels, computedBillingModel) {
		return true, string(amsv1.BillingModelMarketplace)
	}

	// user requested pre-paid billing and it matches the computed billing model
	if requestedBillingModel == computedBillingModel && computedBillingModel == string(amsv1.BillingModelStandard) {
		return true, string(amsv1.BillingModelStandard)
	}

	// user requested consumption based billing and it matches the computed billing model
	if kafkaBillingModel.HasSupportForMarketplace() && // kafkaBillingModel must support marketplaces
		arrays.Contains(supportedMarketplaceBillingModels, computedBillingModel) && // computed billing model must be a marketplace
		arrays.Contains(kafkaBillingModel.AMSBillingModels, computedBillingModel) && // computed billing model must be one of the supported billing model for the kafka billing model
		requestedBillingModel == string(amsv1.BillingModelMarketplace) {
		return true, string(amsv1.BillingModelMarketplace)
	}

	// computed and requested billing models do not match
	return false, ""
}

func (q amsQuotaService) validateBillingAccount(organisationId string, instanceType types.KafkaInstanceType, kafkaBillingModel config.KafkaBillingModel, billingCloudAccountId string, marketplace *string) *errors.ServiceError {
	orgId, err := q.amsClient.GetOrganisationIdFromExternalId(organisationId)
	if err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, fmt.Sprintf("error checking quota: failed to get organization with external id %v", organisationId))
	}

	quotaCosts, err := q.amsClient.GetQuotaCostsForProduct(orgId, kafkaBillingModel.AMSResource, kafkaBillingModel.AMSProduct)
	if err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, fmt.Sprintf("error checking quota: failed to get assigned quota of type %s/%s for organization with id %v", instanceType.String(), kafkaBillingModel.ID, orgId))
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
		return errors.InvalidBillingAccount("no billing accounts available in quota")
	}

	// only one matching billing account is expected. If there are multiple then
	// they are with different cloud providers
	if matchingBillingAccounts > 1 {
		return errors.InvalidBillingAccount("multiple matching billing accounts found, only one expected. Available billing accounts: %v", billingAccounts)
	}
	if matchingBillingAccounts == 0 {
		return errors.InvalidBillingAccount("no matching billing account found. Provided: %s, Available: %v", billingCloudAccountId, billingAccounts)
	}

	// we found one and only one matching billing account
	return nil
}

func (q amsQuotaService) ValidateBillingAccount(organisationId string, instanceType types.KafkaInstanceType, billingModelID string, billingCloudAccountId string, marketplace *string) *errors.ServiceError {
	kafkaBillingModels := []config.KafkaBillingModel{}
	if billingModelID != "" {
		kafkaBillingModel, err := q.kafkaConfig.GetBillingModelByID(instanceType.String(), billingModelID)
		if err != nil {
			return errors.NewWithCause(errors.ErrorGeneral, err, fmt.Sprintf("Error checking quota: %v", err.Error()))
		}
		kafkaBillingModels = []config.KafkaBillingModel{kafkaBillingModel}
	} else {
		kbmList, err := q.kafkaConfig.GetBillingModels(instanceType.String())
		if err != nil {
			return errors.NewWithCause(errors.ErrorGeneral, err, fmt.Sprintf("Error checking quota: %v", err.Error()))
		}
		kafkaBillingModels = kbmList
	}

	history := map[string]int{}

	if arrays.AnyMatch(kafkaBillingModels, func(kbm config.KafkaBillingModel) bool {
		if _, ok := history[kbm.AMSResource+kbm.AMSProduct]; ok {
			return false
		}
		history[kbm.AMSResource+kbm.AMSProduct] = 1
		return q.validateBillingAccount(organisationId, instanceType, kbm, billingCloudAccountId, marketplace) == nil
	}) {
		return nil
	}

	return errors.InvalidBillingAccount("we have not been able to validate your billingAccountID")
}

// TODO: added the `billing model` parameter to the QuotaService interface when adding KafkaBillingModel support to the QUOTA-LIST
// The parameter is currently unused for AMSQuota management: will be used in a subsequent PR when KafkaBillingModel support will be added.
func (q amsQuotaService) CheckIfQuotaIsDefinedForInstanceType(username string, externalId string, instanceType types.KafkaInstanceType, kafkaBillingModel config.KafkaBillingModel) (bool, *errors.ServiceError) {
	orgId, err := q.amsClient.GetOrganisationIdFromExternalId(externalId)
	if err != nil {
		return false, errors.NewWithCause(errors.ErrorGeneral, err, fmt.Sprintf("error checking quota: failed to get organization with external id %v", externalId))
	}

	hasQuota, err := q.hasConfiguredQuotaCost(orgId, kafkaBillingModel)
	if err != nil {
		return false, errors.NewWithCause(errors.ErrorGeneral, err, fmt.Sprintf("error checking quota: failed to get assigned quota of type %v for organization with id %v", kafkaBillingModel.AMSProduct, orgId))
	}

	return hasQuota, nil
}

// hasConfiguredQuotaCost returns true if the given organizationID has at least
// one AMS QuotaCost that complies with the following conditions:
//   - Matches the given input quotaType
//   - Contains at least one AMS RelatedResources whose billing model is one
//     of the supported Billing Models specified in
//     supportedAMSRelatedResourceBillingModels
//   - Has a "MaxAllowedInstances" value greater than 0
//
// An error is returned if the given organizationID has a QuotaCost
// with an unsupported billing model and there are no supported billing models
func (q amsQuotaService) hasConfiguredQuotaCost(organizationID string, kafkaBillingModel config.KafkaBillingModel) (bool, error) {
	quotaCosts, err := q.amsClient.GetQuotaCostsForProduct(organizationID, kafkaBillingModel.AMSResource, kafkaBillingModel.AMSProduct)
	if err != nil {
		return false, err
	}

	var foundUnsupportedBillingModel string
	for _, qc := range quotaCosts {
		if qc.Allowed() > 0 {
			for _, rr := range qc.RelatedResources() {
				if _, isCompatibleBillingModel := supportedAMSRelatedResourceBillingModels[rr.BillingModel()]; isCompatibleBillingModel {
					return true, nil
				}
				foundUnsupportedBillingModel = rr.BillingModel()
			}
		}
	}

	if foundUnsupportedBillingModel != "" {
		return false, errors.GeneralError("product %s only has unsupported allowed billing models. Last one found: %s", kafkaBillingModel.AMSProduct, foundUnsupportedBillingModel)
	}

	return false, nil
}

// getAvailableBillingModelFromKafkaInstanceType gets the billing model of a
// kafka instance type by looking at the resource name and product of the
// instanceType. Only QuotaCosts that have available quota, or that contain a
// RelatedResource with "cost" 0 are considered. Only
// "standard" and "marketplace" billing models are considered. If both are
// detected "standard" is returned.
func (q amsQuotaService) getAvailableBillingModelFromKafkaInstanceType(orgId string, instanceType types.KafkaInstanceType, sizeRequired int) (config.KafkaBillingModel, error) {

	quotaCostCache := map[string][]*amsv1.QuotaCost{}

	getQuotaCosts := func(orgId string, resourceName string, productName string) ([]*amsv1.QuotaCost, error) {
		cacheKey := fmt.Sprintf("%s-%s-%s", orgId, resourceName, productName)
		if res, ok := quotaCostCache[cacheKey]; ok {
			return res, nil
		}
		quotaCosts, err := q.amsClient.GetQuotaCostsForProduct(orgId, resourceName, productName)
		if err != nil {
			return nil, err
		}
		quotaCostCache[cacheKey] = quotaCosts
		return quotaCosts, nil
	}

	// first check if there is quota for 'standard' billing model
	bm, err := q.kafkaConfig.GetBillingModelByID(instanceType.String(), string(amsv1.BillingModelStandard))
	if err == nil {
		quotaCosts, err := getQuotaCosts(orgId, bm.AMSResource, bm.AMSProduct)
		if err != nil {
			return config.KafkaBillingModel{}, errors.InsufficientQuotaError("%v: error getting quotas for product %s", err, bm.AMSProduct)
		}
		for _, qc := range quotaCosts {
			if idx, _ := arrays.FindFirst(qc.RelatedResources(), func(rr *amsv1.RelatedResource) bool {
				return rr.BillingModel() == string(amsv1.BillingModelStandard) && (qc.Consumed()+sizeRequired <= qc.Allowed() || rr.Cost() == 0)
			}); idx != -1 {
				return bm, nil
			}
		}
	}

	// no quota for standard found. Check if quota for any other defined KafkaBillingModel is present
	billingModels, _ := q.kafkaConfig.GetBillingModels(instanceType.String())
	for _, bm := range billingModels {
		if bm.ID == string(amsv1.BillingModelStandard) {
			// skip 'standard' since we already checked this before
			continue
		}
		quotaCosts, err := getQuotaCosts(orgId, bm.AMSResource, bm.AMSProduct)
		if err != nil {
			return config.KafkaBillingModel{}, errors.InsufficientQuotaError("%v: error getting quotas for product %s", err, bm.AMSProduct)
		}
		for _, qc := range quotaCosts {
			if arrays.AnyMatch(qc.RelatedResources(), func(rr *amsv1.RelatedResource) bool {
				return bm.HasSupportForAMSBillingModel(rr.BillingModel()) && (qc.Consumed()+sizeRequired <= qc.Allowed() || rr.Cost() == 0)
			}) {
				return bm, nil
			}

			//for _, rr := range qc.RelatedResources() {
			//	if qc.Consumed()+sizeRequired <= qc.Allowed() || rr.Cost() == 0 {
			//		//if arrays.AnyMatch(bm.AMSBillingModels, arrays.StringEqualsIgnoreCasePredicate(rr.BillingModel())) {
			//		if bm.HasSupportForAMSBillingModel(rr.BillingModel()) {
			//			return bm, nil
			//		}
			//	}
			//}
		}
	}

	return config.KafkaBillingModel{}, nil
}

func (q amsQuotaService) getBillingModel(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (config.KafkaBillingModel, string, error) {
	orgId, err := q.amsClient.GetOrganisationIdFromExternalId(kafka.OrganisationId)
	if err != nil {
		return config.KafkaBillingModel{}, "", errors.NewWithCause(errors.ErrorGeneral, err, fmt.Sprintf("error checking quota: failed to get organization with external id %v", orgId))
	}

	kafkaInstanceSize, err := q.kafkaConfig.GetKafkaInstanceSize(kafka.InstanceType, kafka.SizeId)
	if err != nil {
		return config.KafkaBillingModel{}, "", errors.NewWithCause(errors.ErrorGeneral, err, "error reserving quota")
	}

	getCloudAccounts := func(billingModel config.KafkaBillingModel) []*amsv1.CloudAccount {
		var accounts []*amsv1.CloudAccount
		quotaCosts, err := q.amsClient.GetQuotaCostsForProduct(orgId, billingModel.AMSResource, billingModel.AMSProduct)
		if err != nil {
			// TODO: manage the error
			//return config.KafkaBillingModel{}, "", errors.InsufficientQuotaError("%v: error getting quotas for product %s", err, instanceType.GetQuotaType().GetProduct())
		}
		for _, quotaCost := range quotaCosts {
			accounts = append(accounts, quotaCost.CloudAccounts()...)
		}
		return accounts
	}

	// check if it there is a related resource that supports the marketplace billing model and has quota
	hasSufficientMarketplaceQuota := func(billingModel config.KafkaBillingModel) bool {
		if arrays.NoneMatch(billingModel.AMSBillingModels, func(bm string) bool { return strings.HasPrefix(bm, "marketplace-") }) {
			// passed in billing model does not support marketplace
			return false
		}
		quotaCosts, err := q.amsClient.GetQuotaCostsForProduct(orgId, billingModel.AMSResource, billingModel.AMSProduct)
		if err != nil {
			// TODO: manage the error
			//return config.KafkaBillingModel{}, "", errors.InsufficientQuotaError("%v: error getting quotas for product %s", err, instanceType.GetQuotaType().GetProduct())
		}

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
		kafkaBillingModel, err := q.getAvailableBillingModelFromKafkaInstanceType(orgId, instanceType, kafkaInstanceSize.CapacityConsumed)
		if err != nil {
			return config.KafkaBillingModel{}, "", err
		}
		cloudAccounts := getCloudAccounts(kafkaBillingModel)
		if kafkaBillingModel.HasSupportForMarketplace() && len(cloudAccounts) == 1 {
			// at this point we know that there is only one cloud account
			cloudAccount := cloudAccounts[0]

			if arrays.Contains(supportedCloudProviders, cloudAccount.CloudProviderID()) {
				kafka.BillingCloudAccountId = cloudAccount.CloudAccountID()
				kafka.Marketplace = cloudAccount.CloudProviderID()

				billingModel, err := getMarketplaceBillingModelForCloudProvider(cloudAccount.CloudProviderID())
				if err != nil {
					return config.KafkaBillingModel{}, "", err
				}

				//if arrays.NoneMatch(kafkaBillingModel.AMSBillingModels, arrays.StringEqualsIgnoreCasePredicate(string(billingModel))) {
				if !kafkaBillingModel.HasSupportForAMSBillingModel(string(billingModel)) {
					return config.KafkaBillingModel{}, "", errors.InsufficientQuotaError("provided cloudAccount is not compatible with the available AMS billing models for '%s' (%v)", instanceType.String(), kafkaBillingModel.AMSBillingModels)
				}
				return kafkaBillingModel, string(billingModel), nil
			}
		} else {
			// AMS billing model is 'standard' or cloud account can't be selected
			if kafkaBillingModel.ID == string(amsv1.BillingModelStandard) {
				return kafkaBillingModel, string(amsv1.BillingModelStandard), nil
			}

			if kafkaBillingModel.ID == string(amsv1.BillingModelMarketplace) {
				return kafkaBillingModel, string(amsv1.BillingModelMarketplace), nil
			}
			// TODO: this should be logged
			return config.KafkaBillingModel{}, "", errors.InsufficientQuotaError("unable to detect the billing model for '%s/%s'. Available ams billing models: (%v)", instanceType.String(), kafkaBillingModel.ID, kafkaBillingModel.AMSBillingModels)
		}
	}

	billingModels, _ := q.kafkaConfig.GetBillingModels(instanceType.String())

	// billing account and marketplace (optional) provided
	// find a matching cloud account and assign it if the provider id is aws (the only supported one at the moment)
	for _, bm := range billingModels {

		//if arrays.NoneMatch(bm.AMSBillingModels, func(amsBm string) bool { return strings.HasPrefix(amsBm, "marketplace") }) {
		if !bm.HasSupportForMarketplace() {
			// this kafka billing model does not support marketplaces
			continue
		}

		for _, cloudAccount := range getCloudAccounts(bm) {
			if cloudAccount.CloudAccountID() == kafka.BillingCloudAccountId && (cloudAccount.CloudProviderID() == kafka.Marketplace || kafka.Marketplace == "") {
				// check if we have quota
				//billingModels, _ := q.kafkaConfig.GetBillingModels(instanceType.String())
				//idx, kafkaBillingModel := arrays.FindFirst(billingModels, func(bm config.KafkaBillingModel) bool { return hasSufficientMarketplaceQuota(bm) })
				//
				//if idx == -1 {
				//	return config.KafkaBillingModel{}, "", errors.InsufficientQuotaError("quota does not support marketplace billing or has insufficient quota")
				//}
				if !hasSufficientMarketplaceQuota(bm) {
					// no enough quota for current kafka billing model. Try next
					continue
				}

				// assign marketplace in case it was not provided in the request
				kafka.Marketplace = cloudAccount.CloudProviderID()
				billingModel, err := getMarketplaceBillingModelForCloudProvider(cloudAccount.CloudProviderID())
				if err != nil {
					return config.KafkaBillingModel{}, "", err
				}

				//if arrays.NoneMatch(bm.AMSBillingModels, arrays.StringEqualsIgnoreCasePredicate(string(billingModel))) {
				if !bm.HasSupportForAMSBillingModel(string(billingModel)) {
					return config.KafkaBillingModel{}, "", errors.InsufficientQuotaError("unable to find a cloudAccount compatible with the available AMS billing models for '%s' (%v)", instanceType.String(), bm.AMSBillingModels)
				}

				return bm, string(billingModel), nil
			}
		}
	}

	return config.KafkaBillingModel{}, "", errors.InsufficientQuotaError("no matching marketplace quota found for instance type %s", instanceType.String())
}

func (q amsQuotaService) ReserveQuota(kafka *dbapi.KafkaRequest) (string, *errors.ServiceError) {
	instanceType := types.KafkaInstanceType(kafka.InstanceType)
	kafkaId := kafka.ID

	kafkaInstanceSize, e := q.kafkaConfig.GetKafkaInstanceSize(kafka.InstanceType, kafka.SizeId)
	if e != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, e, "error reserving quota")
	}

	kafkaBillingModel, bm, err := q.getBillingModel(kafka, instanceType)
	if err != nil {
		svcErr := errors.ToServiceError(err)
		return "", errors.NewWithCause(svcErr.Code, svcErr, "error getting billing model")
	}
	if bm == "" {
		return "", errors.InsufficientQuotaError("error getting billing model: No available billing model found")
	}

	bmMatched, matchedBillingModel := q.billingModelMatches(bm, kafka.DesiredKafkaBillingModel, kafkaBillingModel)
	if !bmMatched {
		return "", errors.InvalidBillingAccount("requested billing model does not match assigned. requested: %s, assigned: %s", kafka.DesiredKafkaBillingModel, bm)
	}
	// TODO find a better place to update it as it is a side-effect in nested code
	kafka.DesiredKafkaBillingModel = matchedBillingModel

	// For Kafka requests to be provisioned on GCP currently the only supported
	// AMS billing models are standard or Red Hat Marketplace ("marketplace").
	// If the AMS billing model to be requested is none of those we return an error.
	// TODO change the logic to send "marketplace-rhm" instead of the deprecated
	// "marketplace" once the billing dependencies are updated to deal with it.
	if kafka.CloudProvider == cloudproviders.GCP.String() &&
		bm != string(amsv1.BillingModelStandard) &&
		bm != string(amsv1.BillingModelMarketplace) {
		return "", errors.GeneralError("failed to reserve quota: unsupported billing model %q for Kafka %q in cloud provider %q", bm, kafka.ID, kafka.CloudProvider)
	}
	rr := q.newBaseQuotaReservedResourceBuilder(kafka, kafkaBillingModel)
	rr.BillingModel(amsv1.BillingModel(bm))
	rr.Count(kafkaInstanceSize.QuotaConsumed)

	// will be empty if no marketplace account is used
	rr.BillingMarketplaceAccount(kafka.BillingCloudAccountId)

	cb, _ := amsv1.NewClusterAuthorizationRequest().
		AccountUsername(kafka.Owner).
		CloudProviderID(kafka.CloudProvider).
		ProductID(kafkaBillingModel.AMSProduct).
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
		return "", errors.NewWithCause(errors.ErrorGeneral, err, "error reserving quota")
	}

	if !resp.Allowed() {
		return "", errors.InsufficientQuotaError("Insufficient Quota")
	}

	// TODO find a better place to update it as it is a side-effect in nested code
	kafka.ActualKafkaBillingModel = matchedBillingModel

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
