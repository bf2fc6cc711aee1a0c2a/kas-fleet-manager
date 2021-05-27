package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
)

func ConvertConnectorClusterStatus(from openapi.ConnectorClusterStatus) api.ConnectorClusterStatus {
	return api.ConnectorClusterStatus{
		Conditions: ConvertConditions(from.Conditions),
		Phase:      from.Phase,
		Operators:  ConvertOperators(from.Operators),
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

func ConvertOperators(in []openapi.ConnectorClusterStatusOperators) []api.Operators {
	out := make([]api.Operators, len(in))
	for i, v := range in {
		out[i] = api.Operators{
			Id:        v.Id,
			Version:   v.Version,
			Namespace: v.Namespace,
			Status:    v.Status,
		}
	}
	return out
}

func PresentOperators(in []api.Operators) []openapi.ConnectorClusterStatusOperators {
	out := make([]openapi.ConnectorClusterStatusOperators, len(in))
	for i, v := range in {
		out[i] = openapi.ConnectorClusterStatusOperators{
			Id:        v.Id,
			Version:   v.Version,
			Namespace: v.Namespace,
			Status:    v.Status,
		}
	}
	return out
}
