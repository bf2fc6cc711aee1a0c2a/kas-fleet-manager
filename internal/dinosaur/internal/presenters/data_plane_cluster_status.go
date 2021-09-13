package presenters

import (
	"sort"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
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

	// We try to get the versions from status.Strimzi and if it has not been defined
	// we try to fallback to status.StrimziVersions.
	if status.DinosaurOperator != nil {
		for _, val := range status.DinosaurOperator {
			strimziVersion := api.StrimziVersion{
				Version: val.Version,
				Ready:   val.Ready,
			}
			res = append(res, api.StrimziVersion(strimziVersion))
		}

	} else { // fall back to StrimziVersions.
		for _, val := range status.DinosaurOperatorVersions {
			strimziVersion := api.StrimziVersion{
				Version: val,
				Ready:   true,
			}
			res = append(res, api.StrimziVersion(strimziVersion))
		}
	}

	var errors errors.ErrorList

	sort.Slice(res, func(i, j int) bool {
		compareRes, err := res[i].Compare(res[j])
		if err != nil {
			errors = append(errors, err)
		}
		return compareRes == -1
	})

	if errors != nil {
		return nil, errors
	}

	return res, nil
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
