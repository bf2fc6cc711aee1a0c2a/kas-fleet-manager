package utils

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

const (
	CloudProviderAWS   = "aws"
	CloudProviderRHM   = "rhm"
	CloudProviderAzure = "azure"
)

var supportedCloudProviders = []string{CloudProviderAWS, CloudProviderRHM, CloudProviderAzure}

type BillingModelResolver interface {
	SupportRequest(kafka *dbapi.KafkaRequest) bool
	Resolve(orgID string, kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (BillingModelDetails, error)
}

func NewBillingModelResolver(quotaConfigProvider AMSQuotaConfigProvider, kafkaConfig *config.KafkaConfig) BillingModelResolver {
	return &billingModelResolver{
		QuotaConfigProvider: &cachingQuotaConfigProvider{
			quotaProvider: quotaConfigProvider,
			cache:         map[string][]*amsv1.QuotaCost{},
		},
		KafkaConfig: kafkaConfig,
	}
}

type billingModelResolver struct {
	QuotaConfigProvider AMSQuotaConfigProvider
	KafkaConfig         *config.KafkaConfig
}

func (bmr *billingModelResolver) Resolve(orgID string, kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (BillingModelDetails, error) {

	resolvers := []BillingModelResolver{
		&simpleBillingModelResolver{
			quotaConfigProvider: bmr.QuotaConfigProvider,
			kafkaConfig:         bmr.KafkaConfig,
		},
		&cloudAccountBillingModelResolver{
			quotaConfigProvider: bmr.QuotaConfigProvider,
			kafkaConfig:         bmr.KafkaConfig,
		},
	}

	for _, resolver := range resolvers {
		if resolver.SupportRequest(kafka) {
			return resolver.Resolve(orgID, kafka, instanceType)
		}
	}

	return BillingModelDetails{}, errors.GeneralError("no billing model resolver found for the received kafka request")
}

func (bmr *billingModelResolver) SupportRequest(kafka *dbapi.KafkaRequest) bool {
	return true
}
