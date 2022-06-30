package presenters

import (
	"encoding/json"
	admin "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func PresentConnectorDeployment(from dbapi.ConnectorDeployment) (private.ConnectorDeployment, *errors.ServiceError) {
	var conditions []private.MetaV1Condition
	if from.Status.Conditions != nil {
		err := from.Status.Conditions.Unmarshal(&conditions)
		if err != nil {
			return private.ConnectorDeployment{}, errors.GeneralError("invalid status conditions: %v", err)
		}
	}

	var operators private.ConnectorDeploymentStatusOperators
	if from.Status.Operators != nil {
		err := from.Status.Operators.Unmarshal(&operators)
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
			ConnectorResourceVersion: from.ConnectorVersion,
		},
		Status: private.ConnectorDeploymentStatus{
			Phase:           private.ConnectorState(from.Status.Phase),
			ResourceVersion: from.Status.Version,
			Conditions:      conditions,
			Operators:       operators,
		},
	}, nil
}

func PresentConnectorDeploymentAdminView(from private.ConnectorDeployment, clusterId string) (admin.ConnectorDeploymentAdminView, *errors.ServiceError) {
	var conditions []admin.MetaV1Condition
	if len(from.Status.Conditions) > 0 {
		for _, condition := range from.Status.Conditions {
			conditions = append(conditions, admin.MetaV1Condition{
				Type:               condition.Type,
				Reason:             condition.Reason,
				Message:            condition.Message,
				LastTransitionTime: condition.LastTransitionTime,
			})
		}
	}

	view := admin.ConnectorDeploymentAdminView{
		Id: from.Id,

		Metadata: admin.ConnectorDeploymentAdminViewAllOfMetadata{
			CreatedAt:       from.Metadata.CreatedAt,
			UpdatedAt:       from.Metadata.UpdatedAt,
			ResourceVersion: from.Metadata.ResourceVersion,
			ResolvedSecrets: from.Metadata.ResolvedSecrets,
		},

		Spec: admin.ConnectorDeploymentAdminSpec{
			ConnectorId:              from.Spec.ConnectorId,
			ConnectorResourceVersion: from.Spec.ConnectorResourceVersion,
			ConnectorTypeId:          from.Spec.ConnectorTypeId,
			ClusterId:                clusterId,
			NamespaceId:              from.Spec.NamespaceId,
			OperatorId:               from.Spec.OperatorId,
			DesiredState:             admin.ConnectorDesiredState(from.Spec.DesiredState),
			ShardMetadata:            from.Spec.ShardMetadata,
		},

		Status: admin.ConnectorDeploymentStatus{
			Phase:           admin.ConnectorState(from.Status.Phase),
			ResourceVersion: from.Status.ResourceVersion,
			Operators: admin.ConnectorDeploymentStatusOperators{
				Assigned: admin.ConnectorOperator{
					Id:      from.Status.Operators.Assigned.Id,
					Type:    from.Status.Operators.Assigned.Type,
					Version: from.Status.Operators.Assigned.Version,
				},
				Available: admin.ConnectorOperator{
					Id:      from.Status.Operators.Available.Id,
					Type:    from.Status.Operators.Available.Type,
					Version: from.Status.Operators.Available.Version,
				},
			},
			Conditions: conditions,
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
