package utils

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

const (
	CloudProviderAWS   = "aws"
	CloudProviderRHM   = "rhm"
	CloudProviderAzure = "azure"
)

var supportedCloudProviders = []string{CloudProviderAWS, CloudProviderRHM, CloudProviderAzure}

/********************************************************************************************************************
 * This is the root of the billing resolver chain.
 * It demands the resolution of the billing model to the defined resolvers and stops the chain as soon as one of
 * the resolvers is able to manage the request.
 ********************************************************************************************************************/

type BillingModelResolver interface {
	SupportRequest(kafka *dbapi.KafkaRequest) bool
	Resolve(orgID string, kafka *dbapi.KafkaRequest) (BillingModelDetails, error)
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

func (bmr *billingModelResolver) Resolve(orgID string, kafka *dbapi.KafkaRequest) (BillingModelDetails, error) {

	resolvers := []BillingModelResolver{
		&kafkaBillingModelResolver{
			quotaConfigProvider: bmr.QuotaConfigProvider,
			kafkaConfig:         bmr.KafkaConfig,
		},
		&simpleBillingModelResolver{
			quotaConfigProvider: bmr.QuotaConfigProvider,
			kafkaConfig:         bmr.KafkaConfig,
		},
		&cloudAccountBillingModelResolver{
			quotaConfigProvider: bmr.QuotaConfigProvider,
			kafkaConfig:         bmr.KafkaConfig,
		},
	}

	if idx, resolver := arrays.FindFirst(resolvers, func(x BillingModelResolver) bool { return x.SupportRequest(kafka) }); idx != -1 {
		return resolver.Resolve(orgID, kafka)
	}

	return BillingModelDetails{}, errors.GeneralError("no billing model resolver found for the received kafka request")
}

func (bmr *billingModelResolver) SupportRequest(kafka *dbapi.KafkaRequest) bool {
	return true
}
