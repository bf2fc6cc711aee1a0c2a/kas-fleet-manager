package services

import (
	"context"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

var _ ObservatoriumService = &observatoriumService{}

type observatoriumService struct {
	observatorium *observatorium.Client
	kafkaService  KafkaService
}

func NewObservatoriumService(observatorium *observatorium.Client, kafkaService KafkaService) ObservatoriumService {
	return &observatoriumService{
		observatorium: observatorium,
		kafkaService:  kafkaService,
	}
}

type ObservatoriumService interface {
	GetKafkaState(name string, namespaceName string) (observatorium.KafkaState, error)
	GetMetricsByKafkaId(ctx context.Context, csMetrics *observatorium.KafkaMetrics, id string, query observatorium.MetricsReqParams) (string, *errors.ServiceError)
}

func (obs observatoriumService) GetKafkaState(name string, namespaceName string) (observatorium.KafkaState, error) {
	return obs.observatorium.Service.GetKafkaState(name, namespaceName)
}

func (obs observatoriumService) GetMetricsByKafkaId(ctx context.Context, kafkasMetrics *observatorium.KafkaMetrics, id string, query observatorium.MetricsReqParams) (string, *errors.ServiceError) {
	kafkaRequest, err := obs.kafkaService.Get(ctx, id)
	if err != nil {
		return "", err
	}

	namespace, replaceErr := BuildNamespaceName(kafkaRequest)
	if replaceErr != nil {
		return kafkaRequest.ID, errors.NewWithCause(errors.ErrorGeneral, replaceErr, "failed to retrieve metrics")
	}
	replaceErr = obs.observatorium.Service.GetMetrics(kafkasMetrics, namespace, &query)
	if replaceErr != nil {
		return kafkaRequest.ID, errors.NewWithCause(errors.ErrorGeneral, replaceErr, "failed to retrieve metrics")
	}

	return kafkaRequest.ID, nil
}
