package connector

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector"
	"github.com/goava/di"
)

func ConfigProviders(dinosaurEnabled bool) di.Option {
	return connector.ConfigProviders(dinosaurEnabled)
}
