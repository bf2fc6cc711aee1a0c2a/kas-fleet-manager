package mocks

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
)

func BuildValidDataPlaneClusterConfigKasFleetshardOperatorOLMConfig() *config.DataplaneClusterConfig {
	dataplaneClusterConfig := config.DataplaneClusterConfig{
		KasFleetshardOperatorOLMConfig: config.OperatorInstallationConfig{
			Namespace:           "namespace-name",
			IndexImage:          "index-image-1",
			SubscriptionChannel: "alpha",
			Package:             "package-1",
		},
	}
	return &dataplaneClusterConfig
}
