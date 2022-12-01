package quota

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cloudproviders"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services/quota/internal/utils"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

type amsQuotaService struct {
	amsClient   ocm.AMSClient
	kafkaConfig *config.KafkaConfig
}

var _ services.QuotaService = &amsQuotaService{}

const (
	amsReservedResourceResourceTypeClusterAWS = "cluster.aws"
	amsReservedResourceResourceTypeClusterGCP = "cluster.gcp"

	amsClusterAuthorizationRequestAvailabilityZoneSingle = "single"
	amsClusterAuthorizationRequestAvailabilityZoneMulti  = "multi"
)

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
	return (requestedBillingModel == "" || shared.StringEqualsIgnoreCase(kafkaBillingModel.ID, requestedBillingModel)) && kafkaBillingModel.HasSupportForAMSBillingModel(computedBillingModel), kafkaBillingModel.ID
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
	var kafkaBillingModels []config.KafkaBillingModel
	if billingModelID != "" {
		kafkaBillingModel, err := q.kafkaConfig.GetBillingModelByID(instanceType.String(), billingModelID)
		if err != nil {
			return errors.NewWithCause(errors.ErrorGeneral, err, fmt.Sprintf("error checking quota: %v", err.Error()))
		}
		kafkaBillingModels = []config.KafkaBillingModel{kafkaBillingModel}
	} else {
		kbmList, err := q.kafkaConfig.GetBillingModels(instanceType.String())
		if err != nil {
			return errors.NewWithCause(errors.ErrorGeneral, err, fmt.Sprintf("error checking quota: %v", err.Error()))
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

func (q amsQuotaService) getBillingModel(kafka *dbapi.KafkaRequest) (config.KafkaBillingModel, string, error) {
	orgId, err := q.amsClient.GetOrganisationIdFromExternalId(kafka.OrganisationId)
	if err != nil {
		return config.KafkaBillingModel{}, "", errors.NewWithCause(errors.ErrorGeneral, err, fmt.Sprintf("error checking quota: failed to get organization with external id %v", orgId))
	}

	resolver := utils.NewBillingModelResolver(q.amsClient, q.kafkaConfig)

	resolvedBillingModel, err := resolver.Resolve(orgId, kafka)

	if err != nil {
		return config.KafkaBillingModel{}, "", errors.NewWithCause(errors.ErrorInsufficientQuota, err, "unable to detect billing model")
	}

	return resolvedBillingModel.KafkaBillingModel, resolvedBillingModel.AMSBillingModel, nil
}

func (q amsQuotaService) ReserveQuota(kafka *dbapi.KafkaRequest) (string, *errors.ServiceError) {
	kafkaId := kafka.ID

	kafkaInstanceSize, e := q.kafkaConfig.GetKafkaInstanceSize(kafka.InstanceType, kafka.SizeId)
	if e != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, e, "error reserving quota")
	}

	kafkaBillingModel, bm, err := q.getBillingModel(kafka)
	if err != nil {
		svcErr := errors.ToServiceError(err)
		return "", errors.NewWithCause(svcErr.Code, svcErr, "error getting billing model")
	}
	if bm == "" {
		return "", errors.InsufficientQuotaError("error getting billing model: No available billing model found")
	}

	bmMatched, _ := q.billingModelMatches(bm, kafka.DesiredKafkaBillingModel, kafkaBillingModel)
	if !bmMatched {
		return "", errors.InvalidBillingAccount("requested billing model does not match assigned. requested: %s, assigned: %s", kafka.DesiredKafkaBillingModel, bm)
	}
	// TODO find a better place to update it as it is a side-effect in nested code
	kafka.DesiredKafkaBillingModel = kafkaBillingModel.ID

	// TODO: is this needed?
	if kafka.CloudProvider == cloudproviders.GCP.String() && bm == "marketplace-rhm" {
		bm = "marketplace"
	}
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
	kafka.ActualKafkaBillingModel = kafkaBillingModel.ID

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
