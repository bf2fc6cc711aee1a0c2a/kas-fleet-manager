package presenters

import (
	"encoding/json"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func PresentConnectorDeployment(from api.ConnectorDeployment) (openapi.ConnectorDeployment, *errors.ServiceError) {
	var conditions []openapi.MetaV1Condition
	if from.Status.Conditions != nil {
		err := json.Unmarshal([]byte(from.Status.Conditions), &conditions)
		if err != nil {
			return openapi.ConnectorDeployment{}, errors.GeneralError("invalid status conditions: %v", err)
		}
	}

	var operators openapi.ConnectorDeploymentStatusOperators
	if from.Status.Operators != nil {
		err := json.Unmarshal([]byte(from.Status.Operators), &operators)
		if err != nil {
			return openapi.ConnectorDeployment{}, errors.GeneralError("invalid status operators: %v", err)
		}
	}

	reference := PresentReference(from.ID, from)
	return openapi.ConnectorDeployment{
		Id:   reference.Id,
		Kind: reference.Kind,
		Href: reference.Href,
		Metadata: openapi.ConnectorDeploymentAllOfMetadata{
			CreatedAt:       from.CreatedAt,
			UpdatedAt:       from.UpdatedAt,
			ResourceVersion: from.Version,
		},
		Spec: openapi.ConnectorDeploymentSpec{
			ConnectorId:              from.ConnectorID,
			ConnectorResourceVersion: from.ConnectorVersion,
		},
		Status: openapi.ConnectorDeploymentStatus{
			Phase:           from.Status.Phase,
			ResourceVersion: from.Status.Version,
			Conditions:      conditions,
			Operators:       operators,
		},
	}, nil
}

func ConvertConnectorDeploymentStatus(from openapi.ConnectorDeploymentStatus) (api.ConnectorDeploymentStatus, *errors.ServiceError) {
	conditions, err := json.Marshal(from.Conditions)
	if err != nil {
		return api.ConnectorDeploymentStatus{}, errors.BadRequest("invalid conditions: %v", err)
	}
	operators, err := json.Marshal(from.Operators)
	if err != nil {
		return api.ConnectorDeploymentStatus{}, errors.BadRequest("invalid operators: %v", err)
	}
	return api.ConnectorDeploymentStatus{
		Phase:            from.Phase,
		Version:          from.ResourceVersion,
		Conditions:       conditions,
		Operators:        operators,
		UpgradeAvailable: from.Operators.Available.Id != "" && from.Operators.Available.Id != from.Operators.Assigned.Id,
	}, nil
}
