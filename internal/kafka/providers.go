package kafka

import (
	cloudprovider "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cmd/cloudprovider"
	cluster "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cmd/cluster"
	errors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cmd/errors"
	kafka "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cmd/kafka"
	observatorium "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cmd/observatorium"
	serviceaccounts "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cmd/serviceaccounts"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/routes"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/provider"

	"github.com/goava/di"
)

func ConfigProviders() provider.Map {
	return provider.Map{
		"ServiceInjector":         di.Provide(provider.Func(ServiceProviders)),
		"ClusterCommand":          di.Provide(cluster.NewClusterCommand),
		"KafkaCommand":            di.Provide(kafka.NewKafkaCommand),
		"CloudProviderCommand":    di.Provide(cloudprovider.NewCloudProviderCommand),
		"RunObservatoriumCommand": di.Provide(observatorium.NewRunObservatoriumCommand),
		"ServiceAccountCommand":   di.Provide(serviceaccounts.NewServiceAccountCommand),
		"ErrorsCommand":           di.Provide(errors.NewErrorsCommand),
	}
}

func ServiceProviders(configContainer *di.Container) (provider.Map, error) {
	return provider.Map{
		"RouteLoader": di.Provide(routes.NewRouteLoader),
	}, nil
}
