package authorization

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/goava/di"
)

func ConfigProviders() di.Option {
	return di.Options(
		di.Provide(environments.Func(ServiceProviders)),
	)
}

func ServiceProviders() di.Option {
	return di.Options(
		di.Provide(NewAuthorization),
	)
}

func NewAuthorization(ocmConfig *ocm.OCMConfig, ocmClientFactory *ocm.OCMClientFactory) Authorization {
	if ocmConfig.EnableMock {
		return NewMockAuthorization()
	} else {
		connection := ocmClientFactory.GetConnection(ocm.AMSClientType)
		return NewOCMAuthorization(connection)
	}
}
