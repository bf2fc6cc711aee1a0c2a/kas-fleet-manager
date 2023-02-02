package presenters

import (
	"encoding/json"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func ConvertProcessorRequest(id string, from *public.ProcessorRequest) (*dbapi.Processor, *errors.ServiceError) {
	definition, err := json.Marshal(from.Definition)
	if err != nil {
		return nil, errors.BadRequest("invalid processor definition: %v", err)
	}
	errorHandler, err := json.Marshal(from.ErrorHandler)
	if err != nil {
		return nil, errors.BadRequest("invalid processor error handler: %v", err)
	}

	return &dbapi.Processor{
		Model: db.Model{
			ID: id,
		},
		Name:            from.Name,
		NamespaceId:     from.NamespaceId,
		ProcessorTypeId: from.ProcessorTypeId,
		Definition:      definition,
		ErrorHandler:    errorHandler,
		DesiredState:    dbapi.ProcessorDesiredState(from.DesiredState),
		Channel:         string(from.Channel),
		Annotations:     ConvertProcessorAnnotations(id, from.Annotations),
		Kafka: dbapi.KafkaConnectionSettings{
			KafkaID:         from.Kafka.Id,
			BootstrapServer: from.Kafka.Url,
		},
		ServiceAccount: dbapi.ServiceAccount{
			ClientId:     from.ServiceAccount.ClientId,
			ClientSecret: from.ServiceAccount.ClientSecret,
		},
		Status: dbapi.ProcessorStatus{
			Model: db.Model{ID: id},
		},
	}, nil
}
