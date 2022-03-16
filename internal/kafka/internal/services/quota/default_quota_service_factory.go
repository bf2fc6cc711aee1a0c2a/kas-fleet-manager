package quota

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/quota_management"
)

// DefaultQuotaServiceFactory the default implementation for ProviderFactory
type DefaultQuotaServiceFactory struct {
	quotaServiceContainer map[api.QuotaType]services.QuotaService
}

func NewDefaultQuotaServiceFactory(
	amsClient ocm.AMSClient,
	connectionFactory *db.ConnectionFactory,
	quotaManagementListConfig *quota_management.QuotaManagementListConfig,
	kafkaConfig *config.KafkaConfig,
) services.QuotaServiceFactory {
	quotaServiceContainer := map[api.QuotaType]services.QuotaService{
		api.AMSQuotaType:                 &amsQuotaService{amsClient: amsClient, kafkaConfig: kafkaConfig},
		api.QuotaManagementListQuotaType: &QuotaManagementListService{connectionFactory: connectionFactory, quotaManagementList: quotaManagementListConfig, kafkaConfig: kafkaConfig},
	}
	return &DefaultQuotaServiceFactory{quotaServiceContainer: quotaServiceContainer}
}

func (factory *DefaultQuotaServiceFactory) GetQuotaService(quoataType api.QuotaType) (services.QuotaService, *errors.ServiceError) {
	if quoataType == api.UndefinedQuotaType {
		quoataType = api.QuotaManagementListQuotaType
	}

	quotaService, ok := factory.quotaServiceContainer[quoataType]
	if !ok {
		return nil, errors.GeneralError("invalid quota service type: %v", quoataType)
	}

	return quotaService, nil
}
