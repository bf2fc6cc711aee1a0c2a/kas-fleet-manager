package presenters

import (
	"encoding/json"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func ConvertProcessor(from *public.Processor) (*dbapi.Processor, *errors.ServiceError) {
	definition, err := json.Marshal(from.Definition)
	if err != nil {
		return nil, errors.BadRequest("invalid processor definition: %v", err)
	}
	errorHandler, err := json.Marshal(from.ErrorHandler)
	if err != nil {
		return nil, errors.BadRequest("invalid processor error handler: %v", err)
	}

	var namespaceId string
	if from.NamespaceId != "" {
		namespaceId = from.NamespaceId
	}
	return &dbapi.Processor{
		Model: db.Model{
			ID: from.Id,
		},
		Name:            from.Name,
		NamespaceId:     namespaceId,
		ProcessorTypeId: from.ProcessorTypeId,
		Owner:           from.Owner,
		Version:         from.ResourceVersion,
		DesiredState:    dbapi.ProcessorDesiredState(from.DesiredState),
		Channel:         string(from.Channel),
		Definition:      definition,
		ErrorHandler:    errorHandler,
		Kafka: dbapi.KafkaConnectionSettings{
			KafkaID:         from.Kafka.Id,
			BootstrapServer: from.Kafka.Url,
		},
		ServiceAccount: dbapi.ServiceAccount{
			ClientId:     from.ServiceAccount.ClientId,
			ClientSecret: from.ServiceAccount.ClientSecret,
		},
		Annotations: ConvertProcessorAnnotations(from.Id, from.Annotations),
		Status: dbapi.ProcessorStatus{
			Phase: dbapi.ProcessorStatusPhase(from.Status.State),
		},
	}, nil
}

func ConvertProcessorAnnotations(id string, annotations map[string]string) []dbapi.ProcessorAnnotation {
	res := make([]dbapi.ProcessorAnnotation, len(annotations))
	i := 0
	for k, v := range annotations {
		res[i].ProcessorID = id
		res[i].Key = k
		res[i].Value = v
		i++
	}

	return res
}
func PresentProcessor(from *dbapi.Processor) (public.Processor, *errors.ServiceError) {
	definition := make(map[string]interface{}, len(from.Definition))
	if err := from.Definition.Unmarshal(&definition); err != nil {
		return public.Processor{}, errors.BadRequest("invalid processors definition: %v", err)
	}
	errorHandler := public.ErrorHandler{}
	if err := from.ErrorHandler.Unmarshal(&errorHandler); err != nil {
		return public.Processor{}, errors.BadRequest("invalid processor error handler: %v", err)
	}

	namespaceId := from.NamespaceId
	reference := PresentReference(from.ID, from)
	return public.Processor{
		Id:   reference.Id,
		Kind: reference.Kind,
		Href: reference.Href,

		Name:            from.Name,
		NamespaceId:     namespaceId,
		ProcessorTypeId: from.ProcessorTypeId,

		Owner:           from.Owner,
		CreatedAt:       from.CreatedAt,
		ModifiedAt:      from.UpdatedAt,
		ResourceVersion: from.Version,
		Definition:      definition,
		ErrorHandler:    errorHandler,
		Annotations:     PresentProcessorAnnotations(from.Annotations),
		Status: public.ProcessorStatusStatus{
			State: public.ProcessorState(from.Status.Phase),
		},
		DesiredState: public.ProcessorDesiredState(from.DesiredState),
		Channel:      public.Channel(from.Channel),
		Kafka: public.KafkaConnectionSettings{
			Id:  from.Kafka.KafkaID,
			Url: from.Kafka.BootstrapServer,
		},
		ServiceAccount: public.ServiceAccount{
			ClientId:     from.ServiceAccount.ClientId,
			ClientSecret: from.ServiceAccount.ClientSecret,
		},
	}, nil
}

func PresentProcessorWithError(from *dbapi.ProcessorWithConditions) (public.Processor, *errors.ServiceError) {
	processor, err := PresentProcessor(&from.Processor)
	if err != nil {
		return processor, err
	}

	var conditions []private.MetaV1Condition
	if from.Conditions != nil {
		err := json.Unmarshal(from.Conditions, &conditions)
		if err != nil {
			return public.Processor{}, errors.GeneralError("invalid conditions: %v", err)
		}
		processor.Status.Error = getStatusError(conditions)
	}

	return processor, nil
}

func PresentProcessorAnnotations(annotations []dbapi.ProcessorAnnotation) map[string]string {
	res := make(map[string]string, len(annotations))
	for _, ann := range annotations {
		res[ann.Key] = ann.Value
	}
	return res
}
