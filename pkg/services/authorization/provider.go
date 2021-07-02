package authorization

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/provider"
	"github.com/goava/di"
	sdkClient "github.com/openshift-online/ocm-sdk-go"
)

func ConfigProviders() di.Option {
	return di.Options(
		di.Provide(provider.Func(ServiceProviders)),
	)
}

func ServiceProviders() di.Option {
	return di.Options(
		di.Provide(NewAuthorization),
	)
}

func NewAuthorization(OCM *config.OCMConfig, connection *sdkClient.Connection) Authorization {
	if OCM.EnableMock {
		return NewMockAuthroization()
	} else {
		return NewOCMAuthorization(connection)
	}
}
