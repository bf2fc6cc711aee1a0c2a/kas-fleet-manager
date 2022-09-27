package presenters

import (
	admin "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/private"
)

func ConvertConnectorClusterStatus(from private.ConnectorClusterStatus) dbapi.ConnectorClusterStatus {
	return dbapi.ConnectorClusterStatus{
		Conditions: ConvertConditions(from.Conditions),
		Phase:      dbapi.ConnectorClusterPhaseEnum(from.Phase),
		Operators:  ConvertOperatorStatus(from.Operators),
		Platform: dbapi.ConnectorClusterPlatform{
			ID:      from.Platform.Id,
			Type:    from.Platform.Type,
			Version: from.Platform.Version,
		},
	}
}

func PresentConnectorClusterAdminStatus(from dbapi.ConnectorClusterStatus) admin.ConnectorClusterAdminStatus {
	return admin.ConnectorClusterAdminStatus{
		Conditions: PresentAdminConditions(from.Conditions),
		State:      admin.ConnectorClusterState(from.Phase),
		Operators:  PresentAdminOperators(from.Operators),
		Platform: admin.ConnectorClusterPlatform{
			Id:      from.Platform.ID,
			Type:    from.Platform.Type,
			Version: from.Platform.Version,
		},
	}
}

func ConvertConditions(in []private.MetaV1Condition) []dbapi.Condition {
	if len(in) == 0 {
		return nil
	}
	out := make([]dbapi.Condition, len(in))
	for i, v := range in {
		out[i] = dbapi.Condition{
			Type:               v.Type,
			Reason:             v.Reason,
			Message:            v.Message,
			Status:             v.Status,
			LastTransitionTime: v.LastTransitionTime,
		}
	}
	return out
}

func PresentAdminConditions(in []dbapi.Condition) []admin.MetaV1Condition {
	out := make([]admin.MetaV1Condition, len(in))
	for i, v := range in {
		out[i] = admin.MetaV1Condition{
			Type:               v.Type,
			Reason:             v.Reason,
			Message:            v.Message,
			Status:             v.Status,
			LastTransitionTime: v.LastTransitionTime,
		}
	}
	return out
}

func ConvertOperatorStatus(in []private.ConnectorClusterStatusOperators) []dbapi.OperatorStatus {
	if len(in) == 0 {
		return nil
	}
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

func PresentAdminOperators(in []dbapi.OperatorStatus) []admin.ConnectorClusterAdminStatusOperators {
	out := make([]admin.ConnectorClusterAdminStatusOperators, len(in))
	for i, v := range in {
		out[i] = admin.ConnectorClusterAdminStatusOperators{
			Operator: admin.ConnectorOperator{
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
