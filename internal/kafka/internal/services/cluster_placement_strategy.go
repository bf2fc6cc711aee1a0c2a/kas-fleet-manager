package services

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/pkg/errors"
)

//go:generate moq -out cluster_placement_strategy_moq.go . ClusterPlacementStrategy
type ClusterPlacementStrategy interface {
	// FindCluster finds and returns a Cluster depends on the specific impl.
	FindCluster(kafka *dbapi.KafkaRequest) (*api.Cluster, error)
}

// NewClusterPlacementStrategy return a concrete strategy impl. depends on the placement configuration
func NewClusterPlacementStrategy(clusterService ClusterService, dataplaneClusterConfig *config.DataplaneClusterConfig, kafkaConfig *config.KafkaConfig) ClusterPlacementStrategy {
	var clusterSelection ClusterPlacementStrategy
	switch {
	case dataplaneClusterConfig.IsDataPlaneManualScalingEnabled():
		clusterSelection = &FirstSchedulableWithinLimit{dataplaneClusterConfig, clusterService, kafkaConfig}
	case dataplaneClusterConfig.IsDataPlaneAutoScalingEnabled():
		clusterSelection = &FirstReadyWithCapacity{clusterService, kafkaConfig}
	default:
		clusterSelection = &FirstReadyCluster{clusterService}
	}
	return clusterSelection
}

// FirstReadyCluster finds and returns the first cluster with Ready status
type FirstReadyCluster struct {
	ClusterService ClusterService
}

func (f *FirstReadyCluster) FindCluster(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
	criteria := FindClusterCriteria{
		Provider:              kafka.CloudProvider,
		Region:                kafka.Region,
		MultiAZ:               kafka.MultiAZ,
		Status:                api.ClusterReady,
		SupportedInstanceType: kafka.InstanceType,
	}

	cluster, err := f.ClusterService.FindCluster(criteria)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find a cluster with criteria '%v'", criteria)
	}

	return cluster, nil
}

// FirstSchedulableWithinLimit finds and returns the first cluster which is schedulable and the number of
// Kafka clusters associated with it is within the defined limit.
type FirstSchedulableWithinLimit struct {
	DataplaneClusterConfig *config.DataplaneClusterConfig
	ClusterService         ClusterService
	KafkaConfig            *config.KafkaConfig
}

func (f *FirstSchedulableWithinLimit) FindCluster(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
	criteria := FindClusterCriteria{
		Provider:              kafka.CloudProvider,
		Region:                kafka.Region,
		MultiAZ:               kafka.MultiAZ,
		Status:                api.ClusterReady,
		SupportedInstanceType: kafka.InstanceType,
	}

	kafkaInstanceSize, e := f.KafkaConfig.GetKafkaInstanceSize(kafka.InstanceType, kafka.SizeId)
	if e != nil {
		return nil, errors.Wrapf(e, "failed to get kafka instance size for cluster with criteria '%v'", criteria)
	}

	//#1
	clusterObj, err := f.ClusterService.FindAllClusters(criteria)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find all clusters with criteria '%v'", criteria)
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
	clusterWithinLimit, errf := f.findClusterKafkaInstanceCount(clusterSchIds)
	if errf != nil {
		return nil, errors.Wrapf(err, "failed to find cluster kafka instance count for clusters '%v'", clusterSchIds)
	}

	//#3 which schedulable cluster is also within the limit
	//we want to make sure the order of the ids configuration is always respected: e.g the first cluster in the configuration that passes all the checks should be picked first
	for _, schClusterid := range clusterSchIds {
		cnt := clusterWithinLimit[schClusterid]
		if dataplaneClusterConfig.IsNumberOfKafkaWithinClusterLimit(schClusterid, cnt+kafkaInstanceSize.CapacityConsumed) {
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
		return nil, err
	} else {
		clusterWithinLimitMap := make(map[string]int)
		for _, c := range instanceLst {
			clusterWithinLimitMap[c.Clusterid] = c.Count
		}
		return clusterWithinLimitMap, nil
	}
}

// FirstReadyWithCapacity finds and returns the first cluster in a Ready status with remaining capacity
type FirstReadyWithCapacity struct {
	ClusterService ClusterService
	KafkaConfig    *config.KafkaConfig
}

func (f *FirstReadyWithCapacity) FindCluster(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
	// Find all clusters that match with the criteria
	criteria := FindClusterCriteria{
		Provider:              kafka.CloudProvider,
		Region:                kafka.Region,
		MultiAZ:               kafka.MultiAZ,
		Status:                api.ClusterReady,
		SupportedInstanceType: kafka.InstanceType,
	}

	clusters, findAllClusterErr := f.ClusterService.FindAllClusters(criteria)
	if findAllClusterErr != nil || len(clusters) == 0 {
		return nil, errors.Wrapf(findAllClusterErr, "failed to find all clusters with criteria '%v'", criteria)
	}

	// Get total number of streaming unit used per region and instance type
	streamingUnitCountPerRegionList, countStreamingUnitErr := f.ClusterService.FindStreamingUnitCountByClusterAndInstanceType()
	if countStreamingUnitErr != nil {
		return nil, errors.Wrapf(countStreamingUnitErr, "failed to get count of streaming units by cluster and instance type for criteria '%v'", criteria)
	}

	instanceSize, getInstanceSizeErr := f.KafkaConfig.GetKafkaInstanceSize(kafka.InstanceType, kafka.SizeId)
	if getInstanceSizeErr != nil {
		return nil, errors.Wrapf(getInstanceSizeErr, "failed to get kafka instance size for cluster with criteria '%v'", criteria)
	}

	for _, cluster := range clusters {
		if cluster.ClusterType == api.ManagedDataPlaneClusterType.String() {
			clusterNotFull := f.isManagedClusterNotFull(cluster, streamingUnitCountPerRegionList, kafka, instanceSize)
			if clusterNotFull {
				return cluster, nil
			}
		}
	}

	// no cluster found
	return nil, nil
}

func (f *FirstReadyWithCapacity) isManagedClusterNotFull(cluster *api.Cluster, streamingUnitCountPerRegionList KafkaStreamingUnitCountPerClusterList,
	kafka *dbapi.KafkaRequest, instanceSize *config.KafkaInstanceSize) bool {
	currentStreamingUnitsUsed := streamingUnitCountPerRegionList.GetStreamingUnitCountForClusterAndInstanceType(cluster.ClusterID, kafka.InstanceType)
	capacityInfo := cluster.RetrieveDynamicCapacityInfo()
	maxStreamingUnits := capacityInfo[kafka.InstanceType].MaxUnits

	return currentStreamingUnitsUsed+instanceSize.CapacityConsumed <= int(maxStreamingUnits)
}
