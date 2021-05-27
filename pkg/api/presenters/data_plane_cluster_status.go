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
		NodeInfo:   getNodeInfo(status),
		ResizeInfo: getResizeInfo(status),
		Remaining:  getRemaining(status),
	}
}

func getNodeInfo(status openapi.DataPlaneClusterUpdateStatusRequest) api.DataPlaneClusterStatusNodeInfo {
	var nodeInfo api.DataPlaneClusterStatusNodeInfo
	if status.NodeInfo != nil {
		nodeInfo = api.DataPlaneClusterStatusNodeInfo{
			Ceiling:                int(*status.NodeInfo.Ceiling),
			Floor:                  int(*status.NodeInfo.Floor),
			Current:                int(*status.NodeInfo.Current),
			CurrentWorkLoadMinimum: int(*status.NodeInfo.CurrentWorkLoadMinimum),
		}
	} else if status.DeprecatedNodeInfo != nil {
		nodeInfo = api.DataPlaneClusterStatusNodeInfo{
			Ceiling:                int(*status.DeprecatedNodeInfo.Ceiling),
			Floor:                  int(*status.DeprecatedNodeInfo.Floor),
			Current:                int(*status.DeprecatedNodeInfo.Current),
			CurrentWorkLoadMinimum: int(*status.DeprecatedNodeInfo.DeprecatedCurrentWorkLoadMinimum),
		}
	}
	return nodeInfo
}

func getResizeInfo(status openapi.DataPlaneClusterUpdateStatusRequest) api.DataPlaneClusterStatusResizeInfo {
	var resizeInfo api.DataPlaneClusterStatusResizeInfo
	if status.ResizeInfo != nil {
		resizeInfo = api.DataPlaneClusterStatusResizeInfo{
			NodeDelta: int(*status.ResizeInfo.NodeDelta),
			Delta: api.DataPlaneClusterStatusCapacity{
				IngressEgressThroughputPerSec: *status.ResizeInfo.Delta.IngressEgressThroughputPerSec,
				Connections:                   int(*status.ResizeInfo.Delta.Connections),
				DataRetentionSize:             *status.ResizeInfo.Delta.DataRetentionSize,
				Partitions:                    int(*status.ResizeInfo.Delta.Partitions),
			},
		}
	} else if status.DeprecatedResizeInfo != nil {
		resizeInfo = api.DataPlaneClusterStatusResizeInfo{
			NodeDelta: int(*status.DeprecatedResizeInfo.DeprecatedNodeDelta),
			Delta: api.DataPlaneClusterStatusCapacity{
				IngressEgressThroughputPerSec: *status.DeprecatedResizeInfo.Delta.DeprecatedIngressEgressThroughputPerSec,
				Connections:                   int(*status.DeprecatedResizeInfo.Delta.Connections),
				DataRetentionSize:             *status.DeprecatedResizeInfo.Delta.DeprecatedDataRetentionSize,
				Partitions:                    int(*status.DeprecatedResizeInfo.Delta.Partitions),
			},
		}
	}
	return resizeInfo
}

func getRemaining(status openapi.DataPlaneClusterUpdateStatusRequest) api.DataPlaneClusterStatusCapacity {
	remaining := api.DataPlaneClusterStatusCapacity{
		Connections: int(*status.Remaining.Connections),
		Partitions:  int(*status.Remaining.Partitions),
	}
	if status.Remaining.IngressEgressThroughputPerSec != nil {
		remaining.IngressEgressThroughputPerSec = *status.Remaining.IngressEgressThroughputPerSec
	} else if status.Remaining.DeprecatedDataRetentionSize != nil {
		remaining.IngressEgressThroughputPerSec = *status.Remaining.DeprecatedIngressEgressThroughputPerSec
	}
	if status.Remaining.DataRetentionSize != nil {
		remaining.DataRetentionSize = *status.Remaining.DataRetentionSize
	} else if status.Remaining.DeprecatedDataRetentionSize != nil {
		remaining.DataRetentionSize = *status.Remaining.DeprecatedDataRetentionSize
	}
	return remaining
}

func PresentDataPlaneClusterConfig(config *api.DataPlaneClusterConfig) openapi.DataplaneClusterAgentConfig {
	accessToken := config.Observability.AccessToken
	return openapi.DataplaneClusterAgentConfig{
		Spec: openapi.DataplaneClusterAgentConfigSpec{
			Observability: openapi.DataplaneClusterAgentConfigSpecObservability{
				AccessToken:           &accessToken,
				DeprecatedAccessToken: &accessToken,
				Channel:               config.Observability.Channel,
				Repository:            config.Observability.Repository,
				Tag:                   config.Observability.Tag,
			},
		},
	}
}
