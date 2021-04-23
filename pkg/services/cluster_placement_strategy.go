package services

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
)

//go:generate moq -out cluster_placement_strategy_moq.go . ClusterPlacementStrategy
type ClusterPlacementStrategy interface {
	// FindCluster finds and returns a Cluster depends on the specific impl.
	FindCluster(kafka *api.KafkaRequest) (*api.Cluster, error)
}

// NewClusterPlacementStrategy return a concrete strategy impl. depends on the placement configuration
func NewClusterPlacementStrategy(configService ConfigService, clusterService ClusterService) ClusterPlacementStrategy {
	var clusterSelection ClusterPlacementStrategy
	if configService.GetConfig().OSDClusterConfig.IsDataPlaneManualScalingEnabled() {
		clusterSelection = &FirstSchedulableWithinLimit{configService, clusterService}
	} else {
		clusterSelection = &FirstReadyCluster{configService, clusterService}
	}
	return clusterSelection
}

// FirstReadyCluster finds and returns the first cluster with Ready status
type FirstReadyCluster struct {
	ConfigService  ConfigService
	ClusterService ClusterService
}

func (f *FirstReadyCluster) FindCluster(kafka *api.KafkaRequest) (*api.Cluster, error) {
	criteria := FindClusterCriteria{
		Provider: kafka.CloudProvider,
		Region:   kafka.Region,
		MultiAZ:  kafka.MultiAZ,
		Status:   api.ClusterReady,
	}

	cluster, err := f.ClusterService.FindCluster(criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to find cluster for kafka request %s: %w", kafka.ID, err)
	}

	return cluster, nil
}

// FirstSchedulableWithinLimit finds and returns the first cluster which is schedulable and the number of
// Kafka clusters associated with it is within the defined limit.
type FirstSchedulableWithinLimit struct {
	ConfigService  ConfigService
	ClusterService ClusterService
}

func (f *FirstSchedulableWithinLimit) FindCluster(kafka *api.KafkaRequest) (*api.Cluster, error) {
	criteria := FindClusterCriteria{
		Provider: kafka.CloudProvider,
		Region:   kafka.Region,
		MultiAZ:  kafka.MultiAZ,
		Status:   api.ClusterReady,
	}

	//#1
	clusterObj, err := f.ClusterService.FindAllClusters(criteria)
	if err != nil {
		return nil, err
	}

	osdClusterConfig := f.ConfigService.GetConfig().OSDClusterConfig.ClusterConfig

	//#2 - collect schedulable clusters
	clusterSchIds := []string{}
	for _, cluster := range clusterObj {
		isSchedulable := osdClusterConfig.IsClusterSchedulable(cluster.ClusterID)
		if isSchedulable {
			clusterSchIds = append(clusterSchIds, cluster.ClusterID)
		}
	}

	if len(clusterSchIds) == 0 {
		return nil, nil
	}

	//search for limit
	clusterWithinLimit, errf := f.findClusterKafkaInstanceCount(clusterSchIds)
	if errf != nil {
		return nil, errf
	}

	//#3 which schedulable cluster is also within the limit
	//we want to make sure the order of the ids configuration is always respected: e.g the first cluster in the configuration that passes all the checks should be picked first
	for _, schClusterid := range clusterSchIds {
		cnt := clusterWithinLimit[schClusterid]
		if osdClusterConfig.IsNumberOfKafkaWithinClusterLimit(schClusterid, cnt+1) {
			return searchClusterObjInArray(clusterObj, schClusterid), nil
		}
	}

	//no cluster available
	return nil, nil
}

func searchClusterObjInArray(clusters []*api.Cluster, clusterId string) *api.Cluster {
	for _, cluster := range clusters {
		if cluster.ClusterID == clusterId {
			return cluster
		}
	}
	return nil
}

// findClusterKafkaInstanceCount searches DB for the number of Kafka instance associated with each OSD Clusters
func (f *FirstSchedulableWithinLimit) findClusterKafkaInstanceCount(clusterIDs []string) (map[string]int, error) {
	if instanceLst, err := f.ClusterService.FindKafkaInstanceCount(clusterIDs); err != nil {
		return nil, fmt.Errorf("failed to found kafka instance count for cluster %s : %w", clusterIDs, err)
	} else {
		clusterWithinLimitMap := make(map[string]int)
		for _, c := range instanceLst {
			clusterWithinLimitMap[c.Clusterid] = c.Count
		}
		return clusterWithinLimitMap, nil
	}
}
