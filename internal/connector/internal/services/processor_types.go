package services

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
)

type ProcessorTypesService interface {
	PutProcessorShardMetadata(*dbapi.ProcessorShardMetadata) (int64, *errors.ServiceError)
	GetLatestProcessorShardMetadata() (*dbapi.ProcessorShardMetadata, *errors.ServiceError)
}

var _ ProcessorTypesService = &processorTypesService{}

type processorTypesService struct {
	connectionFactory *db.ConnectionFactory
}

func NewProcessorTypesService(connectionFactory *db.ConnectionFactory) *processorTypesService {
	return &processorTypesService{
		connectionFactory: connectionFactory,
	}
}

func (p *processorTypesService) PutProcessorShardMetadata(processorShardMetadata *dbapi.ProcessorShardMetadata) (int64, *errors.ServiceError) {
	// This is an over simplified implementation to ensure we store a ProcessorShardMetadata record
	dbConn := p.connectionFactory.New()
	if err := dbConn.Save(processorShardMetadata).Error; err != nil {
		return 0, errors.GeneralError("failed to create processor shard metadata %v: %v", processorShardMetadata, err)
	}
	return processorShardMetadata.ID, nil
}

func (p *processorTypesService) GetLatestProcessorShardMetadata() (*dbapi.ProcessorShardMetadata, *errors.ServiceError) {
	resource := &dbapi.ProcessorShardMetadata{}
	dbConn := p.connectionFactory.New()

	err := dbConn.
		Where(dbapi.ProcessorShardMetadata{}).
		Order("revision desc").
		First(&resource).Error

	if err != nil {
		if services.IsRecordNotFoundError(err) {
			return nil, errors.NotFound("processor type shard metadata not found")
		}
		return nil, errors.GeneralError("unable to get processor shard metadata: %s", err)
	}
	return resource, nil
}
