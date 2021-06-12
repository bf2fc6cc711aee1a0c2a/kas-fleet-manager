package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/private"
)

func ConvertConnectorClusterStatus(from private.ConnectorClusterStatus) dbapi.ConnectorClusterStatus {
	return dbapi.ConnectorClusterStatus{
		Conditions: ConvertConditions(from.Conditions),
		Phase:      from.Phase,
		Operators:  ConvertOperatorStatus(from.Operators),
	}
}

func PresentConnectorClusterStatus(from dbapi.ConnectorClusterStatus) private.ConnectorClusterStatus {
	return private.ConnectorClusterStatus{
		Conditions: PresentConditions(from.Conditions),
		Phase:      from.Phase,
		Operators:  PresentOperators(from.Operators),
	}
}

func ConvertConditions(in []private.MetaV1Condition) []dbapi.Condition {
	out := make([]dbapi.Condition, len(in))
	for i, v := range in {
		var t string
		if v.LastTransitionTime != "" {
			t = v.LastTransitionTime
		} else {
			t = v.DeprecatedLastTransitionTime
		}
		out[i] = dbapi.Condition{
			Type:               v.Type,
			Reason:             v.Reason,
			Message:            v.Message,
			Status:             v.Status,
			LastTransitionTime: t,
		}
	}
	return out
}
func PresentConditions(in []dbapi.Condition) []private.MetaV1Condition {
	out := make([]private.MetaV1Condition, len(in))
	for i, v := range in {
		out[i] = private.MetaV1Condition{
			Type:                         v.Type,
			Reason:                       v.Reason,
			Message:                      v.Message,
			Status:                       v.Status,
			LastTransitionTime:           v.LastTransitionTime,
			DeprecatedLastTransitionTime: v.LastTransitionTime,
		}
	}
	return out
}

func ConvertOperatorStatus(in []private.ConnectorClusterStatusOperators) []dbapi.OperatorStatus {
	out := make([]dbapi.OperatorStatus, len(in))
	for i, v := range in {
		out[i] = dbapi.OperatorStatus{
			Id:        v.Operator.Id,
			Type:      v.Operator.Type,
			Version:   v.Operator.Version,
			Namespace: v.Namespace,
			Status:    v.Status,
		}
	}
	return out
}

func PresentOperators(in []dbapi.OperatorStatus) []private.ConnectorClusterStatusOperators {
	out := make([]private.ConnectorClusterStatusOperators, len(in))
	for i, v := range in {
		out[i] = private.ConnectorClusterStatusOperators{
			Operator: private.ConnectorOperator{
				Id:      v.Id,
				Type:    v.Type,
				Version: v.Version,
			},
			Namespace: v.Namespace,
			Status:    v.Status,
		}
	}
	return out
}
