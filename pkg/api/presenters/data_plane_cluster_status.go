package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
)

func ConvertDataPlaneClusterStatus(status openapi.DataPlaneClusterUpdateStatusRequest) *api.DataPlaneClusterStatus {
	conds := []api.DataPlaneClusterStatusCondition{}
	for _, statusCond := range status.Conditions {
		conds = append(conds, api.DataPlaneClusterStatusCondition{
			Type:    statusCond.Type,
			Status:  statusCond.Status,
			Reason:  statusCond.Reason,
			Message: statusCond.Message,
		})
	}
	return &api.DataPlaneClusterStatus{
		Conditions: conds,
		NodeInfo: api.DataPlaneClusterStatusNodeInfo{
			Ceiling:                int(*status.NodeInfo.Ceiling),
			Floor:                  int(*status.NodeInfo.Floor),
			Current:                int(*status.NodeInfo.Current),
			CurrentWorkLoadMinimum: int(*status.NodeInfo.CurrentWorkLoadMinimum),
		},
		ResizeInfo: api.DataPlaneClusterStatusResizeInfo{
			NodeDelta: int(*status.ResizeInfo.NodeDelta),
			Delta: api.DataPlaneClusterStatusCapacity{
				IngressEgressThroughputPerSec: *status.ResizeInfo.Delta.IngressEgressThroughputPerSec,
				Connections:                   int(*status.ResizeInfo.Delta.Connections),
				DataRetentionSize:             *status.ResizeInfo.Delta.DataRetentionSize,
				Partitions:                    int(*status.ResizeInfo.Delta.Partitions),
			},
		},
		Remaining: api.DataPlaneClusterStatusCapacity{
			IngressEgressThroughputPerSec: *status.Remaining.IngressEgressThroughputPerSec,
			Connections:                   int(*status.Remaining.Connections),
			DataRetentionSize:             *status.Remaining.DataRetentionSize,
			Partitions:                    int(*status.Remaining.Partitions),
		},
	}
}

func PresentDataPlaneClusterConfig(config *api.DataPlaneClusterConfig) openapi.DataplaneClusterAgentConfig {
	return openapi.DataplaneClusterAgentConfig{
		Spec: openapi.DataplaneClusterAgentConfigSpec{
			Observability: openapi.DataplaneClusterAgentConfigSpecObservability{
				AccessToken: config.Observability.AccessToken,
				Channel:     config.Observability.Channel,
				Repository:  config.Observability.Repository,
				Tag:         config.Observability.Tag,
			},
		},
	}
}
