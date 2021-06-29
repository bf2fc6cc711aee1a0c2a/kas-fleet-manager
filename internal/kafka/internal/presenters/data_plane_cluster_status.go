package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
)

func ConvertDataPlaneClusterStatus(status private.DataPlaneClusterUpdateStatusRequest) *dbapi.DataPlaneClusterStatus {
	conds := []dbapi.DataPlaneClusterStatusCondition{}
	for _, statusCond := range status.Conditions {
		conds = append(conds, dbapi.DataPlaneClusterStatusCondition{
			Type:    statusCond.Type,
			Status:  statusCond.Status,
			Reason:  statusCond.Reason,
			Message: statusCond.Message,
		})
	}

	return &dbapi.DataPlaneClusterStatus{
		Conditions: conds,
		NodeInfo:   getNodeInfo(status),
		ResizeInfo: getResizeInfo(status),
		Remaining:  getRemaining(status),
	}
}

func getNodeInfo(status private.DataPlaneClusterUpdateStatusRequest) dbapi.DataPlaneClusterStatusNodeInfo {
	var nodeInfo dbapi.DataPlaneClusterStatusNodeInfo
	if status.NodeInfo != nil {
		nodeInfo = dbapi.DataPlaneClusterStatusNodeInfo{
			Ceiling:                int(*status.NodeInfo.Ceiling),
			Floor:                  int(*status.NodeInfo.Floor),
			Current:                int(*status.NodeInfo.Current),
			CurrentWorkLoadMinimum: int(*status.NodeInfo.CurrentWorkLoadMinimum),
		}
	} else if status.DeprecatedNodeInfo != nil {
		nodeInfo = dbapi.DataPlaneClusterStatusNodeInfo{
			Ceiling:                int(*status.DeprecatedNodeInfo.Ceiling),
			Floor:                  int(*status.DeprecatedNodeInfo.Floor),
			Current:                int(*status.DeprecatedNodeInfo.Current),
			CurrentWorkLoadMinimum: int(*status.DeprecatedNodeInfo.DeprecatedCurrentWorkLoadMinimum),
		}
	}
	return nodeInfo
}

func getResizeInfo(status private.DataPlaneClusterUpdateStatusRequest) dbapi.DataPlaneClusterStatusResizeInfo {
	var resizeInfo dbapi.DataPlaneClusterStatusResizeInfo
	if status.ResizeInfo != nil {
		resizeInfo = dbapi.DataPlaneClusterStatusResizeInfo{
			NodeDelta: int(*status.ResizeInfo.NodeDelta),
			Delta: dbapi.DataPlaneClusterStatusCapacity{
				IngressEgressThroughputPerSec: *status.ResizeInfo.Delta.IngressEgressThroughputPerSec,
				Connections:                   int(*status.ResizeInfo.Delta.Connections),
				DataRetentionSize:             *status.ResizeInfo.Delta.DataRetentionSize,
				Partitions:                    int(*status.ResizeInfo.Delta.Partitions),
			},
		}
	} else if status.DeprecatedResizeInfo != nil {
		resizeInfo = dbapi.DataPlaneClusterStatusResizeInfo{
			NodeDelta: int(*status.DeprecatedResizeInfo.DeprecatedNodeDelta),
			Delta: dbapi.DataPlaneClusterStatusCapacity{
				IngressEgressThroughputPerSec: *status.DeprecatedResizeInfo.Delta.DeprecatedIngressEgressThroughputPerSec,
				Connections:                   int(*status.DeprecatedResizeInfo.Delta.Connections),
				DataRetentionSize:             *status.DeprecatedResizeInfo.Delta.DeprecatedDataRetentionSize,
				Partitions:                    int(*status.DeprecatedResizeInfo.Delta.Partitions),
			},
		}
	}
	return resizeInfo
}

func getRemaining(status private.DataPlaneClusterUpdateStatusRequest) dbapi.DataPlaneClusterStatusCapacity {
	remaining := dbapi.DataPlaneClusterStatusCapacity{
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

func PresentDataPlaneClusterConfig(config *dbapi.DataPlaneClusterConfig) private.DataplaneClusterAgentConfig {
	accessToken := config.Observability.AccessToken
	return private.DataplaneClusterAgentConfig{
		Spec: private.DataplaneClusterAgentConfigSpec{
			Observability: private.DataplaneClusterAgentConfigSpecObservability{
				AccessToken:           &accessToken,
				DeprecatedAccessToken: &accessToken,
				Channel:               config.Observability.Channel,
				Repository:            config.Observability.Repository,
				Tag:                   config.Observability.Tag,
			},
		},
	}
}
