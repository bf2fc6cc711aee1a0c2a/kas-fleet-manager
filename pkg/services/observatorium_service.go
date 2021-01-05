package services

import (
	"context"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/client/observatorium"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
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
	GetMetricsByKafkaId(ctx context.Context, csMetrics *observatorium.KafkaMetrics, id string, query observatorium.RangeQuery) (string, *errors.ServiceError)
}

func (obs observatoriumService) GetKafkaState(name string, namespaceName string) (observatorium.KafkaState, error) {
	return obs.observatorium.Service.GetKafkaState(name, namespaceName)
}

func (obs observatoriumService) GetMetricsByKafkaId(ctx context.Context, kafkasMetrics *observatorium.KafkaMetrics, id string, query observatorium.RangeQuery) (string, *errors.ServiceError) {
	kafkaRequest, err := obs.kafkaService.Get(ctx, id)
	if err != nil {
		return "", err
	}

	namespace, replaceErr := BuildNamespaceName(kafkaRequest)
	if replaceErr != nil {
		return kafkaRequest.ID, errors.GeneralError("failed to build namespace for kafka %s: %v", kafkaRequest.ID, replaceErr)
	}
	replaceErr = obs.observatorium.Service.GetMetrics(kafkasMetrics, namespace, &query)
	if replaceErr != nil {
		return kafkaRequest.ID, errors.GeneralError("failed to get state from observatorium for kafka %s namespace %s cluster %s: %v", kafkaRequest.ID, namespace, kafkaRequest.ClusterID, replaceErr)
	}

	return kafkaRequest.ID, nil
}
