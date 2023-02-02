package presenters

import (
	"encoding/json"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func PresentProcessorDeployment(from *dbapi.ProcessorDeployment, resolvedSecrets bool) (*private.ProcessorDeployment, *errors.ServiceError) {
	// prepare conditions
	var conditions []private.MetaV1Condition
	if from.Status.Conditions != nil {
		err := from.Status.Conditions.Unmarshal(&conditions)
		if err != nil {
			return &private.ProcessorDeployment{}, errors.GeneralError("invalid status conditions: %v", err)
		}
	}

	// prepare operators
	var operators private.ConnectorDeploymentStatusOperators
	if from.Status.Operators != nil {
		err := from.Status.Operators.Unmarshal(&operators)
		if err != nil {
			return &private.ProcessorDeployment{}, errors.GeneralError("invalid status operators: %v", err)
		}
	}

	// prepare shard metadata
	if len(from.ProcessorShardMetadata.ShardMetadata) == 0 {
		return &private.ProcessorDeployment{}, errors.GeneralError("unable to load processor deployment shard metadata: %+v with id %d, for processor deployment: %s", from.ProcessorShardMetadata, from.ProcessorShardMetadataID, from.ID)
	}
	shardMetadataJson, errShardMetadataConversion := from.ProcessorShardMetadata.ShardMetadata.Object()
	if errShardMetadataConversion != nil {
		return &private.ProcessorDeployment{}, errors.GeneralError("failed to convert shard metadata to json for: %+v, for processor deployment: %s", from.ProcessorShardMetadata, from.ID)
	}

	// present connector
	presentedProcessor, err := PresentProcessor(&from.Processor)
	if err != nil {
		return &private.ProcessorDeployment{}, err
	}

	// present reference
	reference := PresentReference(from.ID, from)

	return &private.ProcessorDeployment{
		Id:   reference.Id,
		Kind: reference.Kind,
		Href: reference.Href,
		Metadata: private.ProcessorDeploymentAllOfMetadata{
			CreatedAt:       from.CreatedAt,
			UpdatedAt:       from.UpdatedAt,
			ResourceVersion: from.Version,
			ResolvedSecrets: resolvedSecrets,
			Annotations:     presentedProcessor.Annotations,
		},
		Spec: private.ProcessorDeploymentSpec{
			ProcessorId:              presentedProcessor.Id,
			OperatorId:               from.OperatorID,
			NamespaceId:              from.NamespaceID,
			ProcessorTypeId:          presentedProcessor.ProcessorTypeId,
			ProcessorResourceVersion: from.ProcessorVersion,
			ShardMetadata:            shardMetadataJson,
			Definition:               presentedProcessor.Definition,
			ErrorHandler: private.ErrorHandler{
				Log:             presentedProcessor.ErrorHandler.Log,
				Stop:            presentedProcessor.ErrorHandler.Stop,
				DeadLetterQueue: private.ErrorHandlerDeadLetterQueueDeadLetterQueue{Topic: presentedProcessor.ErrorHandler.DeadLetterQueue.Topic},
			},
			DesiredState: private.ProcessorDesiredState(presentedProcessor.DesiredState),
			Kafka: private.KafkaConnectionSettings{
				Id:  presentedProcessor.Kafka.Id,
				Url: presentedProcessor.Kafka.Url,
			},
			ServiceAccount: private.ServiceAccount{
				ClientId:     presentedProcessor.ServiceAccount.ClientId,
				ClientSecret: presentedProcessor.ServiceAccount.ClientSecret,
			},
		},
		Status: private.ProcessorDeploymentStatus{
			Phase:           private.ProcessorState(from.Status.Phase),
			ResourceVersion: from.Status.Version,
			Conditions:      conditions,
			Operators:       operators,
		},
	}, nil
}

func ConvertProcessorDeploymentStatus(from private.ProcessorDeploymentStatus) (dbapi.ProcessorDeploymentStatus, *errors.ServiceError) {
	conditions, err := json.Marshal(from.Conditions)
	if err != nil {
		return dbapi.ProcessorDeploymentStatus{}, errors.BadRequest("invalid conditions: %v", err)
	}
	operators, err := json.Marshal(from.Operators)
	if err != nil {
		return dbapi.ProcessorDeploymentStatus{}, errors.BadRequest("invalid operators: %v", err)
	}
	return dbapi.ProcessorDeploymentStatus{
		Phase:            dbapi.ProcessorStatusPhase(from.Phase),
		Version:          from.ResourceVersion,
		Conditions:       conditions,
		Operators:        operators,
		UpgradeAvailable: from.Operators.Available.Id != "" && from.Operators.Available.Id != from.Operators.Assigned.Id,
	}, nil
}
