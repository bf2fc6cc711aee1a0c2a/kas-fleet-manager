package kafka

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/common"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/routes"
	coreConfig "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"

	"github.com/goava/di"
)

func EnvInjections() common.InjectionMap {
	return common.InjectionMap{
		"ServiceInjector": di.Provide(newServiceInjector),
	}
}

func newServiceInjector(container *di.Container) coreConfig.ServiceInjector {
	return serviceInjector{parent: container}
}

type serviceInjector struct {
	parent *di.Container
}

func (s serviceInjector) Injections() (common.InjectionMap, error) {
	return common.InjectionMap{
		"RouteLoader": di.Provide(routes.NewRouteLoader),
	}, nil
}
