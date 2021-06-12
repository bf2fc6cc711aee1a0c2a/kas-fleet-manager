package vault

import (
	common "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/common"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/goava/di"
)

func EnvInjections() common.InjectionMap {
	return common.InjectionMap{
		"Config":          di.Provide(NewConfig, di.As(new(config.ConfigModule))),
		"ServiceInjector": di.Provide(newServiceInjector),
	}
}

func newServiceInjector(container *di.Container) config.ServiceInjector {
	return serviceInjector{parent: container}
}

type serviceInjector struct {
	parent *di.Container
}

func (s serviceInjector) Injections() (common.InjectionMap, error) {
	conf := &Config{}
	if err := s.parent.Resolve(&conf); err != nil {
		return nil, err
	}

	return common.InjectionMap{
		"VaultConfig":  di.ProvideValue(conf),
		"VaultService": di.Provide(NewVaultService),
	}, nil
}
