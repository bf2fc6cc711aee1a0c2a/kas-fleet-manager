package internal

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/common"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/routes"
	services2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/workers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	oworker "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/goava/di"
)

func NewServiceInjector(container *di.Container) config.ServiceInjector {
	return serviceInjector{parent: container}
}

type serviceInjector struct {
	parent *di.Container
}

func (s serviceInjector) Injections() (common.InjectionMap, error) {
	connectorsConfig := &config.ConnectorsConfig{}
	if err := s.parent.Resolve(&connectorsConfig); err != nil {
		return nil, err
	}

	return common.InjectionMap{
		"Config":                  di.ProvideValue(connectorsConfig),
		"ConnectorsService":       di.Provide(services2.NewConnectorsService, di.As(new(services2.ConnectorsService))),
		"ConnectorTypesService":   di.Provide(services2.NewConnectorTypesService, di.As(new(services2.ConnectorTypesService))),
		"ConnectorClusterService": di.Provide(services2.NewConnectorClusterService, di.As(new(services2.ConnectorClusterService))),
		"ConnectorTypesHandler":   di.Provide(handlers.NewConnectorTypesHandler),
		"ConnectorsHandler":       di.Provide(handlers.NewConnectorsHandler),
		"ConnectorClusterHandler": di.Provide(handlers.NewConnectorClusterHandler),
		"RouteLoader":             di.Provide(routes.NewRouteLoader),
		"ConnectorManager":        di.Provide(workers.NewConnectorManager, di.As(new(oworker.Worker))),
	}, nil
}
