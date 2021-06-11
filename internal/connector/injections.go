package connector

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/common"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/goava/di"
)

func EnvInjections() common.InjectionMap {
	return common.InjectionMap{
		"ConfigModule":    di.Provide(config.NewConnectorsConfig, di.As(new(config.ConfigModule))),
		"ServiceInjector": di.Provide(internal.NewServiceInjector),
	}
}
