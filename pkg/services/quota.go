package services

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

//go:generate moq -out quotaservice_moq.go . QuotaService
type QuotaService interface {
	ReserveQuota(productID string, clusterID string, kafkaID string, owner string, reserve bool, availability string) (bool, string, *errors.ServiceError)
	DeleteQuota(id string) *errors.ServiceError
	ListReservedKafkaQuota(productID string, orgID string) ([]string, *errors.ServiceError)
	IsQuotaReserved(productID string, orgID string) (bool, *errors.ServiceError)
}

type quotaService struct {
	ocmClient ocm.Client
}

var _ QuotaService = quotaService{}

func NewQuotaService(ocmClient ocm.Client) *quotaService {
	return &quotaService{
		ocmClient: ocmClient,
	}
}

func newQuotaResource(resourceType string, resourceName string, availability string, byoc bool) amsv1.ReservedResourceBuilder {
	rr := amsv1.ReservedResourceBuilder{}
	rr.ResourceType(resourceType)  //cluster.aws
	rr.BYOC(byoc)                  //false
	rr.ResourceName(resourceName)  //"rhosak"
	rr.BillingModel("marketplace") // "marketplace" or "standard"
	rr.AvailabilityZoneType(availability)
	return rr
}

func (q quotaService) ReserveQuota(productID string, clusterID string, kafkaID string, owner string, reserve bool, availability string) (bool, string, *errors.ServiceError) {
	rr := newQuotaResource("cluster.aws", "rhosak", availability, false)

	cb, _ := amsv1.NewClusterAuthorizationRequest().
		AccountUsername(owner).
		ProductID(productID).
		Managed(false).
		ClusterID(kafkaID). //cluster can't be nil
		ExternalClusterID(clusterID).
		Disconnected(false).
		BYOC(false).
		AvailabilityZone(availability).
		Reserve(reserve). //
		Resources(&rr).
		Build()

	resp, err := q.ocmClient.ClusterAuthorization(cb)
	if err != nil {
		return false, "", errors.GeneralError("%v", err)
	}

	if resp.Allowed() {
		return true, resp.Subscription().ID(), nil
	} else {
		return false, "", nil
	}
}

func (q quotaService) DeleteQuota(SubscriptionId string) *errors.ServiceError {
	_, err := q.ocmClient.DeleteSubscription(SubscriptionId)
	if err != nil {
		return errors.GeneralError("failed to delete the quota: %v", err)
	}
	return nil
}

//Todo discussion required
func (q quotaService) ListReservedKafkaQuota(productID string, orgID string) ([]string, *errors.ServiceError) {
	query := fmt.Sprintf("plan_id is '%s' and status='%s' and organization_id='%s'", productID, "Active", orgID)
	subs, err := q.ocmClient.FindSubscriptions(query)
	if err != nil {
		return nil, errors.GeneralError("failed to list quotas: %v", err)
	}
	i := subs.Total()
	var listCluster = make([]string, i)
	subs.Items().Range(func(index int, item *amsv1.Subscription) bool {
		listCluster[index] = item.ClusterID()
		return true
	})
	return listCluster, nil
}

//Todo discussion required
func (q quotaService) IsQuotaReserved(productID string, orgID string) (bool, *errors.ServiceError) {
	query := fmt.Sprintf("plan_id is '%s' and status='%s' and organization_id='%s'", productID, "Active", orgID)
	subs, err := q.ocmClient.FindSubscriptions(query)
	if err != nil {
		return false, errors.GeneralError("failed to check if quotas is available: %v", err)
	}
	if subs.Total() > 0 {
		return true, nil
	}
	return false, nil
}
