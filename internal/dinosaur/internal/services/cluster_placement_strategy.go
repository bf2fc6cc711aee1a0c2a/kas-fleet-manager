package services

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	"github.com/pkg/errors"
)

//go:generate moq -out cluster_placement_strategy_moq.go . ClusterPlacementStrategy
type ClusterPlacementStrategy interface {
	// FindCluster finds and returns a Cluster depends on the specific impl.
	FindCluster(dinosaur *dbapi.DinosaurRequest) (*api.Cluster, error)
}

// NewClusterPlacementStrategy return a concrete strategy impl. depends on the placement configuration
func NewClusterPlacementStrategy(clusterService ClusterService, dataplaneClusterConfig *config.DataplaneClusterConfig) ClusterPlacementStrategy {
	var clusterSelection ClusterPlacementStrategy
	if dataplaneClusterConfig.IsDataPlaneManualScalingEnabled() {
		clusterSelection = &FirstSchedulableWithinLimit{dataplaneClusterConfig, clusterService}
	} else {
		clusterSelection = &FirstReadyCluster{clusterService}
	}
	return clusterSelection
}

// FirstReadyCluster finds and returns the first cluster with Ready status
type FirstReadyCluster struct {
	ClusterService ClusterService
}

func (f *FirstReadyCluster) FindCluster(dinosaur *dbapi.DinosaurRequest) (*api.Cluster, error) {
	criteria := FindClusterCriteria{
		Provider:              dinosaur.CloudProvider,
		Region:                dinosaur.Region,
		MultiAZ:               dinosaur.MultiAZ,
		Status:                api.ClusterReady,
		SupportedInstanceType: dinosaur.InstanceType,
	}

	cluster, err := f.ClusterService.FindCluster(criteria)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find cluster for dinosaur request %s", dinosaur.ID)
	}

	return cluster, nil
}

// FirstSchedulableWithinLimit finds and returns the first cluster which is schedulable and the number of
// Dinosaur clusters associated with it is within the defined limit.
type FirstSchedulableWithinLimit struct {
	DataplaneClusterConfig *config.DataplaneClusterConfig
	ClusterService         ClusterService
}

func (f *FirstSchedulableWithinLimit) FindCluster(dinosaur *dbapi.DinosaurRequest) (*api.Cluster, error) {
	criteria := FindClusterCriteria{
		Provider:              dinosaur.CloudProvider,
		Region:                dinosaur.Region,
		MultiAZ:               dinosaur.MultiAZ,
		Status:                api.ClusterReady,
		SupportedInstanceType: dinosaur.InstanceType,
	}

	//#1
	clusterObj, err := f.ClusterService.FindAllClusters(criteria)
	if err != nil {
		return nil, err
	}

	dataplaneClusterConfig := f.DataplaneClusterConfig.ClusterConfig

	//#2 - collect schedulable clusters
	clusterSchIds := []string{}
	for _, cluster := range clusterObj {
		isSchedulable := dataplaneClusterConfig.IsClusterSchedulable(cluster.ClusterID)
		if isSchedulable {
			clusterSchIds = append(clusterSchIds, cluster.ClusterID)
		}
	}

	if len(clusterSchIds) == 0 {
		return nil, nil
	}

	//search for limit
	clusterWithinLimit, errf := f.findClusterDinosaurInstanceCount(clusterSchIds)
	if errf != nil {
		return nil, errf
	}

	//#3 which schedulable cluster is also within the limit
	//we want to make sure the order of the ids configuration is always respected: e.g the first cluster in the configuration that passes all the checks should be picked first
	for _, schClusterid := range clusterSchIds {
		cnt := clusterWithinLimit[schClusterid]
		if dataplaneClusterConfig.IsNumberOfDinosaurWithinClusterLimit(schClusterid, cnt+1) {
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

// findClusterDinosaurInstanceCount searches DB for the number of Dinosaur instance associated with each OSD Clusters
func (f *FirstSchedulableWithinLimit) findClusterDinosaurInstanceCount(clusterIDs []string) (map[string]int, error) {
	if instanceLst, err := f.ClusterService.FindDinosaurInstanceCount(clusterIDs); err != nil {
		return nil, errors.Wrapf(err, "failed to found dinosaur instance count for cluster %s", clusterIDs)
	} else {
		clusterWithinLimitMap := make(map[string]int)
		for _, c := range instanceLst {
			clusterWithinLimitMap[c.Clusterid] = c.Count
		}
		return clusterWithinLimitMap, nil
	}
}
