package presenters

import (
	"encoding/json"

	admin "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func PresentConnectorDeployment(from dbapi.ConnectorDeployment, resolvedSecrets bool) (private.ConnectorDeployment, *errors.ServiceError) {
	// prepare conditions
	var conditions []private.MetaV1Condition
	if from.Status.Conditions != nil {
		err := from.Status.Conditions.Unmarshal(&conditions)
		if err != nil {
			return private.ConnectorDeployment{}, errors.GeneralError("invalid status conditions: %v", err)
		}
	}

	// prepare operators
	var operators private.ConnectorDeploymentStatusOperators
	if from.Status.Operators != nil {
		err := from.Status.Operators.Unmarshal(&operators)
		if err != nil {
			return private.ConnectorDeployment{}, errors.GeneralError("invalid status operators: %v", err)
		}
	}

	// prepare shard metadata
	if len(from.ConnectorShardMetadata.ShardMetadata) == 0 {
		return private.ConnectorDeployment{}, errors.GeneralError("Unable to load connector type shard metadata: %+v with id %d, for connector deployment: %s", from.ConnectorShardMetadata, from.ConnectorShardMetadataID, from.ID)
	}
	shardMetadataJson, errShardMetadataConversion := from.ConnectorShardMetadata.ShardMetadata.Object()
	if errShardMetadataConversion != nil {
		return private.ConnectorDeployment{}, errors.GeneralError("Failed to convert shard metadata to json for: %+v, for connector deployment: %s", from.ConnectorShardMetadata, from.ID)
	}

	// present connector
	presentedConnector, err := PresentConnector(&from.Connector)
	if err != nil {
		return private.ConnectorDeployment{}, err
	}

	// present reference
	reference := PresentReference(from.ID, from)

	return private.ConnectorDeployment{
		Id:   reference.Id,
		Kind: reference.Kind,
		Href: reference.Href,
		Metadata: private.ConnectorDeploymentAllOfMetadata{
			CreatedAt:       from.CreatedAt,
			UpdatedAt:       from.UpdatedAt,
			ResourceVersion: from.Version,
			ResolvedSecrets: resolvedSecrets,
		},
		Spec: private.ConnectorDeploymentSpec{
			ConnectorId:              presentedConnector.Id,
			OperatorId:               from.OperatorID,
			NamespaceId:              from.NamespaceID,
			ConnectorResourceVersion: from.ConnectorVersion,
			ShardMetadata:            shardMetadataJson,
			ConnectorSpec:            presentedConnector.Connector,
			DesiredState:             private.ConnectorDesiredState(presentedConnector.DesiredState),
			Kafka: private.KafkaConnectionSettings{
				Id:  presentedConnector.Kafka.Id,
				Url: presentedConnector.Kafka.Url,
			},
			SchemaRegistry: private.SchemaRegistryConnectionSettings{
				Id:  presentedConnector.SchemaRegistry.Id,
				Url: presentedConnector.SchemaRegistry.Url,
			},
			ServiceAccount: private.ServiceAccount{
				ClientId:     presentedConnector.ServiceAccount.ClientId,
				ClientSecret: presentedConnector.ServiceAccount.ClientSecret,
			},
			ConnectorTypeId: presentedConnector.ConnectorTypeId,
		},
		Status: private.ConnectorDeploymentStatus{
			Phase:           private.ConnectorState(from.Status.Phase),
			ResourceVersion: from.Status.Version,
			Conditions:      conditions,
			Operators:       operators,
		},
	}, nil
}

func PresentConnectorDeploymentAdminView(from dbapi.ConnectorDeployment, clusterId string) (admin.ConnectorDeploymentAdminView, *errors.ServiceError) {
	// present the deployment
	fromPresentedConnectorDeployment, err := PresentConnectorDeployment(from, false)
	if err != nil {
		return admin.ConnectorDeploymentAdminView{}, err
	}

	// build conditions
	var conditions []admin.MetaV1Condition
	if len(fromPresentedConnectorDeployment.Status.Conditions) > 0 {
		for _, condition := range fromPresentedConnectorDeployment.Status.Conditions {
			conditions = append(conditions, admin.MetaV1Condition{
				Type:               condition.Type,
				Reason:             condition.Reason,
				Message:            condition.Message,
				LastTransitionTime: condition.LastTransitionTime,
			})
		}
	}

	// build shard metadata status
	deploymentAdminStatusShardMetadata := admin.ConnectorDeploymentAdminStatusShardMetadata{
		Assigned: admin.ConnectorShardMetadata{
			Channel:         from.ConnectorShardMetadata.Channel,
			ConnectorTypeId: from.ConnectorShardMetadata.ConnectorTypeId,
			Revision:        from.ConnectorShardMetadata.Revision,
		},
	}
	if from.ConnectorShardMetadata.LatestRevision != nil {
		deploymentAdminStatusShardMetadata.Available = admin.ConnectorShardMetadata{
			Channel:         from.ConnectorShardMetadata.Channel,
			ConnectorTypeId: from.ConnectorShardMetadata.ConnectorTypeId,
			Revision:        *from.ConnectorShardMetadata.LatestRevision,
		}
	}

	view := admin.ConnectorDeploymentAdminView{
		Id: fromPresentedConnectorDeployment.Id,

		Metadata: admin.ConnectorDeploymentAdminViewAllOfMetadata{
			CreatedAt:       fromPresentedConnectorDeployment.Metadata.CreatedAt,
			UpdatedAt:       fromPresentedConnectorDeployment.Metadata.UpdatedAt,
			ResourceVersion: fromPresentedConnectorDeployment.Metadata.ResourceVersion,
			ResolvedSecrets: fromPresentedConnectorDeployment.Metadata.ResolvedSecrets,
		},

		Spec: admin.ConnectorDeploymentAdminSpec{
			ConnectorId:              fromPresentedConnectorDeployment.Spec.ConnectorId,
			ConnectorResourceVersion: fromPresentedConnectorDeployment.Spec.ConnectorResourceVersion,
			ConnectorTypeId:          fromPresentedConnectorDeployment.Spec.ConnectorTypeId,
			ClusterId:                clusterId,
			NamespaceId:              fromPresentedConnectorDeployment.Spec.NamespaceId,
			OperatorId:               fromPresentedConnectorDeployment.Spec.OperatorId,
			DesiredState:             admin.ConnectorDesiredState(fromPresentedConnectorDeployment.Spec.DesiredState),
			ShardMetadata:            fromPresentedConnectorDeployment.Spec.ShardMetadata,
		},

		Status: admin.ConnectorDeploymentAdminStatus{
			Phase:           admin.ConnectorState(fromPresentedConnectorDeployment.Status.Phase),
			ResourceVersion: fromPresentedConnectorDeployment.Status.ResourceVersion,
			Operators: admin.ConnectorDeploymentAdminStatusOperators{
				Assigned: admin.ConnectorOperator{
					Id:      fromPresentedConnectorDeployment.Status.Operators.Assigned.Id,
					Type:    fromPresentedConnectorDeployment.Status.Operators.Assigned.Type,
					Version: fromPresentedConnectorDeployment.Status.Operators.Assigned.Version,
				},
				Available: admin.ConnectorOperator{
					Id:      fromPresentedConnectorDeployment.Status.Operators.Available.Id,
					Type:    fromPresentedConnectorDeployment.Status.Operators.Available.Type,
					Version: fromPresentedConnectorDeployment.Status.Operators.Available.Version,
				},
			},
			Conditions:    conditions,
			ShardMetadata: deploymentAdminStatusShardMetadata,
		},
	}

	reference := PresentReference(view.Id, view)
	view.Kind = reference.Kind
	view.Href = reference.Href
	return view, nil
}

func ConvertConnectorDeploymentStatus(from private.ConnectorDeploymentStatus) (dbapi.ConnectorDeploymentStatus, *errors.ServiceError) {
	conditions, err := json.Marshal(from.Conditions)
	if err != nil {
		return dbapi.ConnectorDeploymentStatus{}, errors.BadRequest("invalid conditions: %v", err)
	}
	operators, err := json.Marshal(from.Operators)
	if err != nil {
		return dbapi.ConnectorDeploymentStatus{}, errors.BadRequest("invalid operators: %v", err)
	}
	return dbapi.ConnectorDeploymentStatus{
		Phase:            dbapi.ConnectorStatusPhase(from.Phase),
		Version:          from.ResourceVersion,
		Conditions:       conditions,
		Operators:        operators,
		UpgradeAvailable: from.Operators.Available.Id != "" && from.Operators.Available.Id != from.Operators.Assigned.Id,
	}, nil
}
