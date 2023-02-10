package services

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	apiErrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
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
		clusterSelection = &FirstReadyCluster{clusterService, kafkaConfig}
	}
	return clusterSelection
}

// findDataPlaneClusterByIdIfItHasCapacityAvailable finds and returns the desired data plane cluster if
// if it has capacity available.
type findDataPlaneClusterByIdIfItHasCapacityAvailable struct {
	clusterService ClusterService
	kafkaConfig    *config.KafkaConfig
}

// FindCluster finds the data plane cluster that's desired for the given kafka.
// The desired data plane cluster is expressed by the "ClusterID" field in the received kafka request.
// Once the cluster is found, it has to match the following rules:
// 1. The cluster has to be in ready state.
// 2. It also also has to be in the same organization as the kafka request.
// 3. It must have remaining capacity to receive the Kafka.
// Capacity capacity is evaluated based on the MaxUnits stored in DynamicCapacityInfo and the actual used capacity.
func (f *findDataPlaneClusterByIdIfItHasCapacityAvailable) FindCluster(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
	cluster, err := f.clusterService.FindClusterByID(kafka.ClusterID)
	if err != nil || cluster == nil {
		return nil, apiErrors.GeneralError("failed to get cluster with id: %q", kafka.ClusterID)
	}

	if cluster.OrganizationID != kafka.OrganisationId {
		return nil, apiErrors.Unauthorized("user with organization ID: '%s' is unauthorized to create kafka on a cluster belonging to different organization", kafka.OrganisationId)
	}

	if cluster.Status != api.ClusterStatus(statusReady.String()) {
		return nil, apiErrors.BadRequest("cluster with id: %s is not ready to accept kafkas", kafka.ClusterID)
	}

	kafkaSizeConsumption, sizeErr := f.kafkaConfig.GetKafkaInstanceSize(kafka.InstanceType, kafka.SizeId)
	if sizeErr != nil {
		return nil, sizeErr
	}

	capacityInfo, ok := cluster.RetrieveDynamicCapacityInfo()[kafka.InstanceType]

	if !ok {
		return nil, errors.Errorf("instance type %q not supported on selected cluster %q", kafka.InstanceType, kafka.ClusterID)
	}

	streamingUnitCounts, streamingUnitComputationErr := f.clusterService.ComputeConsumedStreamingUnitCountPerInstanceType(cluster.ClusterID)
	if streamingUnitComputationErr != nil {
		return nil, streamingUnitComputationErr
	}

	usedCapacity := streamingUnitCounts[types.KafkaInstanceType(kafka.InstanceType)]
	futureRemainingCapacity := int64(capacityInfo.MaxUnits) - (usedCapacity + int64(kafkaSizeConsumption.CapacityConsumed))
	if futureRemainingCapacity >= 0 {
		return cluster, nil
	}

	return nil, nil
}

// FirstReadyCluster finds and returns the first cluster with Ready status
type FirstReadyCluster struct {
	ClusterService ClusterService
	kafkaConfig    *config.KafkaConfig
}

func (f *FirstReadyCluster) FindCluster(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
	if kafka.DesiredBillingModelIsEnterprise() {
		enterpriseKafkaPlacementStrategy := findDataPlaneClusterByIdIfItHasCapacityAvailable{
			clusterService: f.ClusterService,
			kafkaConfig:    f.kafkaConfig,
		}
		return enterpriseKafkaPlacementStrategy.FindCluster(kafka)
	}

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
	dataplaneClusterConfig *config.DataplaneClusterConfig
	clusterService         ClusterService
	kafkaConfig            *config.KafkaConfig
}

func (f *FirstSchedulableWithinLimit) FindCluster(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
	if kafka.DesiredBillingModelIsEnterprise() {
		enterpriseKafkaPlacementStrategy := findDataPlaneClusterByIdIfItHasCapacityAvailable{
			clusterService: f.clusterService,
			kafkaConfig:    f.kafkaConfig,
		}
		return enterpriseKafkaPlacementStrategy.FindCluster(kafka)
	}

	criteria := FindClusterCriteria{
		Provider:              kafka.CloudProvider,
		Region:                kafka.Region,
		MultiAZ:               kafka.MultiAZ,
		Status:                api.ClusterReady,
		SupportedInstanceType: kafka.InstanceType,
	}

	kafkaInstanceSize, e := f.kafkaConfig.GetKafkaInstanceSize(kafka.InstanceType, kafka.SizeId)
	if e != nil {
		return nil, errors.Wrapf(e, "failed to get kafka instance size for cluster with criteria '%v'", criteria)
	}

	//#1 we retrieve all the clusters that match the criteria
	clusters, err := f.clusterService.FindAllClusters(criteria)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find all clusters with criteria '%v'", criteria)
	}

	//#2 - collect cluster IDs of managed clusters that are chedulable
	clusterIDs := f.findClusterIDsOfManagedClustersThatAreSchedulable(clusters)
	if len(clusterIDs) == 0 {
		return nil, nil
	}

	//search for current consumed streaming unit per cluster
	consumedStreamingUnitPerClusterID, errf := f.findConsumedStreamingUnitPerClusterID(clusterIDs)
	if errf != nil {
		return nil, errors.Wrapf(err, "failed to find cluster kafka instance count for clusters '%v'", clusterIDs)
	}

	//#3 which schedulable cluster is also within the limit
	//we want to make sure the order of the ids configuration is always respected: e.g the first cluster in the configuration that passes all the checks should be picked first
	for _, clusterID := range clusterIDs {
		currentStreamingUnitConsumption := consumedStreamingUnitPerClusterID[clusterID]
		futureStreamingUnitConsumptionInTheCluster := currentStreamingUnitConsumption + kafkaInstanceSize.CapacityConsumed
		numberOfKafkaIsWithinLimit := f.dataplaneClusterConfig.ClusterConfig.IsNumberOfStreamingUnitsWithinClusterLimit(clusterID, futureStreamingUnitConsumptionInTheCluster)
		if numberOfKafkaIsWithinLimit {
			return searchForClusterFromClustersList(clusters, clusterID), nil
		}
	}

	//no cluster available
	return nil, nil
}

// findClusterIDsOfManagedClustersThatAreSchedulable returns the clusterIDs of managed clusters that are schedulable
func (f *FirstSchedulableWithinLimit) findClusterIDsOfManagedClustersThatAreSchedulable(clusterObj []*api.Cluster) []string {
	clusterSchIds := []string{}
	for _, cluster := range clusterObj {
		if cluster.ClusterType != api.ManagedDataPlaneClusterType.String() {
			continue
		}

		isSchedulable := f.dataplaneClusterConfig.ClusterConfig.IsClusterSchedulable(cluster.ClusterID)
		if isSchedulable {
			clusterSchIds = append(clusterSchIds, cluster.ClusterID)
		}
	}
	return clusterSchIds
}

func searchForClusterFromClustersList(clusters []*api.Cluster, clusterId string) *api.Cluster {
	_, cluster := arrays.FindFirst(clusters, func(cluster *api.Cluster) bool {
		return cluster.ClusterID == clusterId && cluster.ClusterType == api.ManagedDataPlaneClusterType.String()
	})

	return cluster
}

// findConsumedStreamingUnitPerClusterID searches DB for the number of consumed streaming units associated with each clusters given by cluster ids
func (f *FirstSchedulableWithinLimit) findConsumedStreamingUnitPerClusterID(clusterIDs []string) (map[string]int, error) {
	instanceLst, err := f.clusterService.FindKafkaInstanceCount(clusterIDs)
	if err != nil {
		return nil, err
	}

	consumedStreamingUnitPerClusterID := make(map[string]int)
	for _, c := range instanceLst {
		consumedStreamingUnitPerClusterID[c.ClusterID] = c.Count
	}
	return consumedStreamingUnitPerClusterID, nil
}

// FirstReadyWithCapacity finds and returns the first cluster in a Ready status with remaining capacity
type FirstReadyWithCapacity struct {
	clusterService ClusterService
	kafkaConfig    *config.KafkaConfig
}

func (f *FirstReadyWithCapacity) FindCluster(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
	if kafka.DesiredBillingModelIsEnterprise() {
		enterpriseKafkaPlacementStrategy := findDataPlaneClusterByIdIfItHasCapacityAvailable{
			clusterService: f.clusterService,
			kafkaConfig:    f.kafkaConfig,
		}
		return enterpriseKafkaPlacementStrategy.FindCluster(kafka)
	}

	// Find all clusters that match with the criteria
	criteria := FindClusterCriteria{
		Provider:              kafka.CloudProvider,
		Region:                kafka.Region,
		MultiAZ:               kafka.MultiAZ,
		Status:                api.ClusterReady,
		SupportedInstanceType: kafka.InstanceType,
	}

	clusters, findAllClusterErr := f.clusterService.FindAllClusters(criteria)
	if findAllClusterErr != nil || len(clusters) == 0 {
		return nil, errors.Wrapf(findAllClusterErr, "failed to find all clusters with criteria '%v'", criteria)
	}

	// Get total number of streaming unit used per region and instance type
	streamingUnitCountPerRegionList, countStreamingUnitErr := f.clusterService.FindStreamingUnitCountByClusterAndInstanceType()
	if countStreamingUnitErr != nil {
		return nil, errors.Wrapf(countStreamingUnitErr, "failed to get count of streaming units by cluster and instance type for criteria '%v'", criteria)
	}

	instanceSize, getInstanceSizeErr := f.kafkaConfig.GetKafkaInstanceSize(kafka.InstanceType, kafka.SizeId)
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
