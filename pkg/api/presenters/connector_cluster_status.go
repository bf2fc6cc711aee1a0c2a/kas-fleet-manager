package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
)

func ConvertConnectorClusterStatus(from openapi.ConnectorClusterStatus) api.ConnectorClusterStatus {
	return api.ConnectorClusterStatus{
		Conditions: ConvertConditions(from.Conditions),
		Phase:      from.Phase,
		Operators:  ConvertOperatorStatus(from.Operators),
	}
}

func PresentConnectorClusterStatus(from api.ConnectorClusterStatus) openapi.ConnectorClusterStatus {
	return openapi.ConnectorClusterStatus{
		Conditions: PresentConditions(from.Conditions),
		Phase:      from.Phase,
		Operators:  PresentOperators(from.Operators),
	}
}

func ConvertConditions(in []openapi.MetaV1Condition) []api.Condition {
	out := make([]api.Condition, len(in))
	for i, v := range in {
		var t string
		if v.LastTransitionTime != "" {
			t = v.LastTransitionTime
		} else {
			t = v.DeprecatedLastTransitionTime
		}
		out[i] = api.Condition{
			Type:               v.Type,
			Reason:             v.Reason,
			Message:            v.Message,
			Status:             v.Status,
			LastTransitionTime: t,
		}
	}
	return out
}
func PresentConditions(in []api.Condition) []openapi.MetaV1Condition {
	out := make([]openapi.MetaV1Condition, len(in))
	for i, v := range in {
		out[i] = openapi.MetaV1Condition{
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

func ConvertOperatorStatus(in []openapi.ConnectorClusterStatusOperators) []api.OperatorStatus {
	out := make([]api.OperatorStatus, len(in))
	for i, v := range in {
		out[i] = api.OperatorStatus{
			Id:        v.Operator.Id,
			Type:      v.Operator.Type,
			Version:   v.Operator.Version,
			Namespace: v.Namespace,
			Status:    v.Status,
		}
	}
	return out
}

func PresentOperators(in []api.OperatorStatus) []openapi.ConnectorClusterStatusOperators {
	out := make([]openapi.ConnectorClusterStatusOperators, len(in))
	for i, v := range in {
		out[i] = openapi.ConnectorClusterStatusOperators{
			Operator: openapi.ConnectorOperator{
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
