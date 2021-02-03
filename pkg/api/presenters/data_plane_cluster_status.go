package presenters

import (
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
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
			Ceiling:                int(status.NodeInfo.Ceiling),
			Floor:                  int(status.NodeInfo.Floor),
			Current:                int(status.NodeInfo.Current),
			CurrentWorkLoadMinimum: int(status.NodeInfo.CurrentWorkLoadMinimum),
		},
		ResizeInfo: api.DataPlaneClusterStatusResizeInfo{
			NodeDelta: int(status.ResizeInfo.NodeDelta),
			Delta: api.DataPlaneClusterStatusCapacity{
				IngressEgressThroughputPerSec: status.ResizeInfo.Delta.IngressEgressThroughputPerSec,
				Connections:                   int(status.ResizeInfo.Delta.Connections),
				DataRetentionSize:             status.ResizeInfo.Delta.DataRetentionSize,
				Partitions:                    int(status.ResizeInfo.Delta.MaxPartitions),
			},
		},
		Remaining: api.DataPlaneClusterStatusCapacity{
			IngressEgressThroughputPerSec: status.Remaining.IngressEgressThroughputPerSec,
			Connections:                   int(status.Remaining.Connections),
			DataRetentionSize:             status.Remaining.DataRetentionSize,
			Partitions:                    int(status.Remaining.Partitions),
		},
	}
}
