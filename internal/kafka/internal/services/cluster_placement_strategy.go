package services

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

//go:generate moq -out cluster_placement_strategy_moq.go . ClusterPlacementStrategy
type ClusterPlacementStrategy interface {
	// FindCluster finds and returns a Cluster depends on the specific impl.
	FindCluster(kafka *dbapi.KafkaRequest) (*api.Cluster, *errors.ServiceError)
}

// NewClusterPlacementStrategy return a concrete strategy impl. depends on the placement configuration
func NewClusterPlacementStrategy(clusterService ClusterService, dataplaneClusterConfig *config.DataplaneClusterConfig, kafkaConfig *config.KafkaConfig) ClusterPlacementStrategy {
	var clusterSelection ClusterPlacementStrategy
	if dataplaneClusterConfig.IsDataPlaneManualScalingEnabled() {
		clusterSelection = &FirstSchedulableWithinLimit{dataplaneClusterConfig, clusterService, kafkaConfig}
	} else {
		clusterSelection = &FirstReadyCluster{clusterService}
	}
	return clusterSelection
}

// FirstReadyCluster finds and returns the first cluster with Ready status
type FirstReadyCluster struct {
	ClusterService ClusterService
}

func (f *FirstReadyCluster) FindCluster(kafka *dbapi.KafkaRequest) (*api.Cluster, *errors.ServiceError) {
	criteria := FindClusterCriteria{
		Provider:              kafka.CloudProvider,
		Region:                kafka.Region,
		MultiAZ:               kafka.MultiAZ,
		Status:                api.ClusterReady,
		SupportedInstanceType: kafka.InstanceType,
	}

	cluster, err := f.ClusterService.FindCluster(criteria)
	if err != nil {
		return nil, err
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

func (f *FirstSchedulableWithinLimit) FindCluster(kafka *dbapi.KafkaRequest) (*api.Cluster, *errors.ServiceError) {
	criteria := FindClusterCriteria{
		Provider:              kafka.CloudProvider,
		Region:                kafka.Region,
		MultiAZ:               kafka.MultiAZ,
		Status:                api.ClusterReady,
		SupportedInstanceType: kafka.InstanceType,
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
	clusterWithinLimit, errf := f.findClusterKafkaInstanceCount(clusterSchIds)
	if errf != nil {
		return nil, errf
	}

	kafkaInstanceSize, e := f.KafkaConfig.GetKafkaInstanceSize(kafka.InstanceType, kafka.SizeId)
	if e != nil {
		return nil, errors.NewWithCause(errors.ErrorInstancePlanNotSupported, err, "failed to find cluster with criteria '%v'", criteria)
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
func (f *FirstSchedulableWithinLimit) findClusterKafkaInstanceCount(clusterIDs []string) (map[string]int, *errors.ServiceError) {
	if instanceLst, err := f.ClusterService.FindKafkaInstanceCount(clusterIDs); err != nil {
		return nil, errors.NewWithCause(err.Code, err, "failed to find kafka instance count for clusters '%v'", clusterIDs)
	} else {
		clusterWithinLimitMap := make(map[string]int)
		for _, c := range instanceLst {
			clusterWithinLimitMap[c.Clusterid] = c.Count
		}
		return clusterWithinLimitMap, nil
	}
}
