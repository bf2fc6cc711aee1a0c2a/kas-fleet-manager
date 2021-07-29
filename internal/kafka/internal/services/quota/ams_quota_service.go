package quota

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

type amsQuotaService struct {
	ocmClient   ocm.AMSClient
	kafkaConfig *config.KafkaConfig
}

func newQuotaResource() amsv1.ReservedResourceBuilder {
	rr := amsv1.ReservedResourceBuilder{}
	rr.ResourceType("cluster.aws") //cluster.aws
	rr.BYOC(false)                 //false
	rr.ResourceName("rhosak")      //"rhosak"
	rr.BillingModel("marketplace") // "marketplace" or "standard"
	rr.AvailabilityZoneType("single")
	return rr
}

func (q amsQuotaService) CheckQuota(kafka *dbapi.KafkaRequest) *errors.ServiceError {
	kafkaId := kafka.ID

	if kafkaId == "" {
		kafkaId = api.NewID() // use a fake to check is quota is available
	}

	rr := newQuotaResource()

	cb, _ := amsv1.NewClusterAuthorizationRequest().
		AccountUsername(kafka.Owner).
		CloudProviderID(kafka.CloudProvider).
		ProductID(q.kafkaConfig.ProductType).
		Managed(false).
		ClusterID(kafkaId).
		ExternalClusterID(kafkaId).
		Disconnected(false).
		BYOC(false).
		AvailabilityZone("single").
		Reserve(false).
		Resources(&rr).
		Build()

	resp, err := q.ocmClient.ClusterAuthorization(cb)
	if err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "Error checking quota")
	}

	if !resp.Allowed() {
		return errors.InsufficientQuotaError("Insufficient Quota")
	}

	return nil
}

func (q amsQuotaService) ReserveQuota(kafka *dbapi.KafkaRequest) (string, *errors.ServiceError) {
	kafkaId := kafka.ID

	rr := newQuotaResource()

	cb, _ := amsv1.NewClusterAuthorizationRequest().
		AccountUsername(kafka.Owner).
		CloudProviderID(kafka.CloudProvider).
		ProductID(q.kafkaConfig.ProductType).
		Managed(false).
		ClusterID(kafkaId).
		ExternalClusterID(kafkaId).
		Disconnected(false).
		BYOC(false).
		AvailabilityZone("single").
		Reserve(true).
		Resources(&rr).
		Build()

	resp, err := q.ocmClient.ClusterAuthorization(cb)
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

	_, err := q.ocmClient.DeleteSubscription(subscriptionId)
	if err != nil {
		return errors.GeneralError("failed to delete the quota: %v", err)
	}
	return nil
}
