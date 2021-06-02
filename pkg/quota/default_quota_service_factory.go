package quota

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
)

// DefaultQuotaServiceFactory the default implementation for ProviderFactory
type DefaultQuotaServiceFactory struct {
	quoataServiceContainer map[api.QuotaType]services.QuotaService
}

func NewDefaultQuotaServiceFactory(ocmClient ocm.Client, connectionFactory *db.ConnectionFactory,
	configService services.ConfigService) services.QuotaServiceFactory {
	quoataServiceContainer := map[api.QuotaType]services.QuotaService{
		api.AMSQuotaType:       &amsQuotaService{ocmClient: ocmClient},
		api.AllowListQuotaType: &allowListQuotaService{connectionFactory: connectionFactory, configService: configService},
	}
	return &DefaultQuotaServiceFactory{quoataServiceContainer: quoataServiceContainer}
}

func (factory *DefaultQuotaServiceFactory) GetQuotaService(quoataType api.QuotaType) (services.QuotaService, *errors.ServiceError) {
	if quoataType == api.UndefinedQuotaType {
		quoataType = api.AllowListQuotaType
	}

	quotaService, ok := factory.quoataServiceContainer[quoataType]
	if !ok {
		return nil, errors.GeneralError("invalid quota service type: %v", quoataType)
	}

	return quotaService, nil
}
