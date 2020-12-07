package services

import (
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/client/observatorium"
)

var _ ObservatoriumService = &observatoriumService{}

type observatoriumService struct {
	observatorium *observatorium.Client
}

func NewObservatoriumService(observatorium *observatorium.Client) ObservatoriumService {
	return &observatoriumService{
		observatorium: observatorium,
	}
}

type ObservatoriumService interface {
	GetKafkaState(name string, namespaceName string) (observatorium.KafkaState, error)
}

func (e observatoriumService) GetKafkaState(name string, namespaceName string) (observatorium.KafkaState, error) {
	return e.observatorium.Service.GetKafkaState(name, namespaceName)
}
