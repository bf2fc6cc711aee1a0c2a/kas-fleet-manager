package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
)

func ConvertDataPlaneClusterStatus(status private.DataPlaneClusterUpdateStatusRequest) (*dbapi.DataPlaneClusterStatus, error) {
	availableStrimziVersions, err := getAvailableStrimziVersions(status)
	if err != nil {
		return nil, err
	}
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
		Conditions:               conds,
		NodeInfo:                 getNodeInfo(status),
		ResizeInfo:               getResizeInfo(status),
		Remaining:                getRemaining(status),
		AvailableStrimziVersions: availableStrimziVersions,
	}, nil
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
	}
	if status.Remaining.DataRetentionSize != nil {
		remaining.DataRetentionSize = *status.Remaining.DataRetentionSize
	}
	return remaining
}

// getAvailableStrimziVersions returns a list of api.StrimziVersion sorted
// as lexicographically ascending sorted list of api.StrimziVersion.Version from the
// status content.
func getAvailableStrimziVersions(status private.DataPlaneClusterUpdateStatusRequest) ([]api.StrimziVersion, error) {
	res := []api.StrimziVersion{}

	for _, val := range status.Strimzi {
		var currKafkaVersions []api.KafkaVersion
		for _, kafkaVersion := range val.KafkaVersions {
			currKafkaVersions = append(currKafkaVersions, api.KafkaVersion{Version: kafkaVersion})
		}
		var currKafkaIBPVersions []api.KafkaIBPVersion
		for _, kafkaIBPVersion := range val.KafkaIbpVersions {
			currKafkaIBPVersions = append(currKafkaIBPVersions, api.KafkaIBPVersion{Version: kafkaIBPVersion})
		}

		strimziVersion := api.StrimziVersion{
			Version:          val.Version,
			Ready:            val.Ready,
			KafkaVersions:    currKafkaVersions,
			KafkaIBPVersions: currKafkaIBPVersions,
		}
		res = append(res, strimziVersion)
	}

	sortedRes, err := api.StrimziVersionsDeepSort(res)
	if err != nil {
		return nil, err
	}

	return sortedRes, nil
}

func PresentDataPlaneClusterConfig(config *dbapi.DataPlaneClusterConfig) private.DataplaneClusterAgentConfig {
	accessToken := config.Observability.AccessToken
	return private.DataplaneClusterAgentConfig{
		Spec: private.DataplaneClusterAgentConfigSpec{
			Observability: private.DataplaneClusterAgentConfigSpecObservability{
				AccessToken: &accessToken,
				Channel:     config.Observability.Channel,
				Repository:  config.Observability.Repository,
				Tag:         config.Observability.Tag,
			},
		},
	}
}
