package services

import (
	"context"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
)

type DataPlaneKafkaService interface {
	UpdateDataPlaneKafkaService(ctx context.Context, clusterId string, status []*api.DataPlaneKafkaStatus) *errors.ServiceError
}

type dataPlaneKafkaService struct {
	kafkaService KafkaService
}

func NewDataPlaneKafkaService(kafkaSrv KafkaService) *dataPlaneKafkaService {
	return &dataPlaneKafkaService{
		kafkaService: kafkaSrv,
	}
}

func (d *dataPlaneKafkaService) UpdateDataPlaneKafkaService(ctx context.Context, clusterId string, status []*api.DataPlaneKafkaStatus) *errors.ServiceError {
	return nil
}
