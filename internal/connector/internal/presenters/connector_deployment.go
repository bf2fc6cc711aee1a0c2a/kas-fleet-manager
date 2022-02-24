package presenters

import (
	"encoding/json"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func PresentConnectorDeployment(from dbapi.ConnectorDeployment) (private.ConnectorDeployment, *errors.ServiceError) {
	var conditions []private.MetaV1Condition
	if from.Status.Conditions != nil {
		err := json.Unmarshal([]byte(from.Status.Conditions), &conditions)
		if err != nil {
			return private.ConnectorDeployment{}, errors.GeneralError("invalid status conditions: %v", err)
		}
	}

	var operators private.ConnectorDeploymentStatusOperators
	if from.Status.Operators != nil {
		err := json.Unmarshal([]byte(from.Status.Operators), &operators)
		if err != nil {
			return private.ConnectorDeployment{}, errors.GeneralError("invalid status operators: %v", err)
		}
	}

	reference := PresentReference(from.ID, from)
	return private.ConnectorDeployment{
		Id:   reference.Id,
		Kind: reference.Kind,
		Href: reference.Href,
		Metadata: private.ConnectorDeploymentAllOfMetadata{
			CreatedAt:       from.CreatedAt,
			UpdatedAt:       from.UpdatedAt,
			ResourceVersion: from.Version,
		},
		Spec: private.ConnectorDeploymentSpec{
			ConnectorId:              from.ConnectorID,
			OperatorId:               from.OperatorID,
			NamespaceId:              from.NamespaceID,
			NamespaceName:            from.NamespaceName,
			ConnectorResourceVersion: from.ConnectorVersion,
		},
		Status: private.ConnectorDeploymentStatus{
			Phase:           from.Status.Phase,
			ResourceVersion: from.Status.Version,
			Conditions:      conditions,
			Operators:       operators,
		},
	}, nil
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
		Phase:            from.Phase,
		Version:          from.ResourceVersion,
		Conditions:       conditions,
		Operators:        operators,
		UpgradeAvailable: from.Operators.Available.Id != "" && from.Operators.Available.Id != from.Operators.Assigned.Id,
	}, nil
}
