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
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers/kafka_mgrs"
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
		"RouteLoader":              di.Provide(routes.NewRouteLoader),
		"ClusterManager":           di.Provide(workers.NewClusterManager, di.As(new(workers.Worker))),
		"KafkaManager":             di.Provide(kafka_mgrs.NewKafkaManager, di.As(new(workers.Worker))),
		"AcceptedKafkaManager":     di.Provide(kafka_mgrs.NewAcceptedKafkaManager, di.As(new(workers.Worker))),
		"PreparingKafkaManage":     di.Provide(kafka_mgrs.NewPreparingKafkaManager, di.As(new(workers.Worker))),
		"DeletingKafkaManager":     di.Provide(kafka_mgrs.NewDeletingKafkaManager, di.As(new(workers.Worker))),
		"ProvisioningKafkaManager": di.Provide(kafka_mgrs.NewProvisioningKafkaManager, di.As(new(workers.Worker))),
		"ReadyKafkaManager":        di.Provide(kafka_mgrs.NewReadyKafkaManager, di.As(new(workers.Worker))),
	}, nil
}
