package modules

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector"
	"github.com/goava/di"
)

func ConnectorsConfigProviders() di.Option {
	return connector.ConfigProviders()
}