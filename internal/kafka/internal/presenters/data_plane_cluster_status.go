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

	dynamicCapacityInfo := map[string]api.DynamicCapacityInfo{}
	for key, capacity := range status.Capacity {
		dynamicCapacityInfo[key] = api.DynamicCapacityInfo{
			MaxUnits:       capacity.MaxUnits,
			RemainingUnits: capacity.RemainingUnits,
		}
	}

	return &dbapi.DataPlaneClusterStatus{
		Conditions:               conds,
		AvailableStrimziVersions: availableStrimziVersions,
		DynamicCapacityInfo:      dynamicCapacityInfo,
	}, nil
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
	capacity := map[string]private.DataplaneClusterAgentConfigSpecCapacity{}
	for k, v := range config.DynamicCapacityInfo {
		capacity[k] = private.DataplaneClusterAgentConfigSpecCapacity{
			MaxNodes: v.MaxNodes,
		}
	}

	return private.DataplaneClusterAgentConfig{
		Spec: private.DataplaneClusterAgentConfigSpec{
			Observability: private.DataplaneClusterAgentConfigSpecObservability{
				AccessToken: &accessToken,
				Channel:     config.Observability.Channel,
				Repository:  config.Observability.Repository,
				Tag:         config.Observability.Tag,
			},
			Capacity: capacity,
			Net: private.DataplaneClusterAgentConfigSpecNet{
				Private: config.NetworkConfiguration.Private,
			},
		},
	}
}
