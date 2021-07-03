package authorization

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/goava/di"
	sdkClient "github.com/openshift-online/ocm-sdk-go"
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

func NewAuthorization(OCM *ocm.OCMConfig, connection *sdkClient.Connection) Authorization {
	if OCM.EnableMock {
		return NewMockAuthroization()
	} else {
		return NewOCMAuthorization(connection)
	}
}
