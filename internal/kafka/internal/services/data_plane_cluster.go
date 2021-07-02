package services

import (
	"context"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/goava/di"
	"strconv"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/golang/glog"
)

// Number of compute nodes in a Multi-AZ must be a multiple of
// this number
const multiAZClusterNodeScalingMultiple = 3

type DataPlaneClusterService interface {
	UpdateDataPlaneClusterStatus(ctx context.Context, clusterID string, status *dbapi.DataPlaneClusterStatus) *errors.ServiceError
	GetDataPlaneClusterConfig(ctx context.Context, clusterID string) (*dbapi.DataPlaneClusterConfig, *errors.ServiceError)
}

var _ DataPlaneClusterService = &dataPlaneClusterService{}

const dataPlaneClusterStatusCondReadyName = "Ready"

type dataPlaneClusterService struct {
	di.Inject
	clusterService         ClusterService
	kafkaConfig            *config.KafkaConfig
	observabilityConfig    *config.ObservabilityConfiguration
	dataplaneClusterConfig *config.DataplaneClusterConfig
}

type dataPlaneComputeNodesKafkaCapacityAttributes struct {
	Connections int
	Partitions  int
}

func NewDataPlaneClusterService(config dataPlaneClusterService) *dataPlaneClusterService {
	return &config
}

func (d *dataPlaneClusterService) GetDataPlaneClusterConfig(ctx context.Context, clusterID string) (*dbapi.DataPlaneClusterConfig, *errors.ServiceError) {
	cluster, svcErr := d.clusterService.FindClusterByID(clusterID)
	if svcErr != nil {
		return nil, svcErr
	}
	if cluster == nil {
		// 404 is used for authenticated requests. So to distinguish the errors, we use 400 here
		return nil, errors.BadRequest("Cluster agent with ID '%s' not found", clusterID)
	}

	return &dbapi.DataPlaneClusterConfig{
		Observability: dbapi.DataPlaneClusterConfigObservability{
			AccessToken: d.observabilityConfig.ObservabilityConfigAccessToken,
			Channel:     d.observabilityConfig.ObservabilityConfigChannel,
			Repository:  d.observabilityConfig.ObservabilityConfigRepo,
			Tag:         d.observabilityConfig.ObservabilityConfigTag,
		},
	}, nil
}

func (d *dataPlaneClusterService) UpdateDataPlaneClusterStatus(ctx context.Context, clusterID string, status *dbapi.DataPlaneClusterStatus) *errors.ServiceError {
	cluster, svcErr := d.clusterService.FindClusterByID(clusterID)
	if svcErr != nil {
		return svcErr
	}
	if cluster == nil {
		// 404 is used for authenticated requests. So to distinguish the errors, we use 400 here
		return errors.BadRequest("Cluster agent with ID '%s' not found", clusterID)
	}

	if !d.clusterCanProcessStatusReports(cluster) {
		glog.V(10).Infof("Cluster with ID '%s' is in '%s' state. Ignoring status report...", clusterID, cluster.Status)
		return nil
	}

	fleetShardOperatorReady, err := d.isFleetShardOperatorReady(status)
	if err != nil {
		return errors.ToServiceError(err)
	}
	if !fleetShardOperatorReady {
		if cluster.Status != api.ClusterWaitingForKasFleetShardOperator {
			err := d.clusterService.UpdateStatus(*cluster, api.ClusterWaitingForKasFleetShardOperator)
			if err != nil {
				return errors.ToServiceError(err)
			}
			metrics.UpdateClusterStatusSinceCreatedMetric(*cluster, api.ClusterWaitingForKasFleetShardOperator)
		}
		glog.V(10).Infof("KAS Fleet Shard Operator not ready for Cluster ID '%s", clusterID)
		return nil
	}

	// We calculate the status based on the stats received by the KAS Fleet operator
	// BEFORE performing the scaling actions. If scaling actions are performed later
	// then it will be reflected on the next data plane cluster status report
	err = d.setClusterStatus(cluster, status)
	if err != nil {
		return errors.ToServiceError(err)
	}

	if d.dataplaneClusterConfig.IsDataPlaneAutoScalingEnabled() {
		computeNodeScalingInProgress, err := d.computeNodeScalingActionInProgress(cluster, status)
		if err != nil {
			return errors.ToServiceError(err)
		}
		if computeNodeScalingInProgress {
			glog.V(10).Infof("Cluster '%s' compute nodes scaling currently in progress. Omitting scaling actions evaluation...", cluster.ClusterID)
			return nil
		}

		_, err = d.updateDataPlaneClusterNodes(cluster, status)
		if err != nil {
			return errors.ToServiceError(err)
		}
	}

	return nil
}

func (d *dataPlaneClusterService) computeNodeScalingActionInProgress(cluster *api.Cluster, status *dbapi.DataPlaneClusterStatus) (bool, error) {
	nodesInfo, err := d.clusterService.GetComputeNodes(cluster.ClusterID)
	if err != nil {
		return false, errors.ToServiceError(err)
	}

	glog.V(10).Infof("Cluster ID %s has %d desired compute nodes and %d existing compute nodes", cluster.ClusterID, nodesInfo.Desired, nodesInfo.Actual)

	return nodesInfo.Actual != nodesInfo.Desired, nil
}

// updateDataPlaneClusterNodes performs node scale-up and scale-down actions on
// the data plane cluster based on the reported status by the kas fleet shard
// operator. The state of the number of desired nodes after the scaling
// actions is returned
func (d *dataPlaneClusterService) updateDataPlaneClusterNodes(cluster *api.Cluster, status *dbapi.DataPlaneClusterStatus) (int, error) {
	// Current assumptions of how the kas-fleet-shard-operator works:
	// 1 - Reported "current" will always be equal or higher than
	//     "workload minimum" because workload minimum is the nodes required
	//     to support the CURRENT workload, which by definition is already
	//     running and alocated to nodes
	// 2 - Reported "current" might be higher than "ceiling"
	// 3 - Reported "floor" will always be less or equal than the reported "current"
	// 4 - Due to points 1,2,3 "floor" will always be less or equal than the reported "workload minimum"
	// 5 - Due to points 1,2,3 "ceil" will always be greater or equal than the reported "workload minimum"
	// 6 - Reported "ceil" cannot be smaller than "floor"
	// 7 - Reported "ceil" and reported "floor" will always be greater than 0
	// 8 - When a scale-up threshold or scale-down threshold is crossed and a scaling
	//     action is performed, the opposite threshold won't be crossed because of that
	//     action. This is, thresholds will be separate enough
	// 9 - resizeInfo.nodeDelta will always be greater than 0
	// TODO should we have a method that checks this assumptions and return an error
	// when some of those don't happen just to cover ourselves from possible
	// incorrect inputs from the other side?

	currentNodes := status.NodeInfo.Current
	desiredNodesAfterScaleActions := currentNodes

	// We scale up when the number of current nodes is not greater than the
	// restricted ceiling and at least one of the scale up thresholds has been crossed
	scaleUpNeeded, err := d.scaleUpNeeded(cluster, status)
	if err != nil {
		return currentNodes, err
	}
	if scaleUpNeeded {
		nodesToScaleUp := d.calculateDesiredNodesToScaleUp(cluster, status)
		desiredNodesAfterScaleActions = currentNodes + nodesToScaleUp
		glog.V(10).Infof("Increasing reported current number of nodes '%v' by '%v'", currentNodes, nodesToScaleUp)
		_, err := d.clusterService.SetComputeNodes(cluster.ClusterID, desiredNodesAfterScaleActions)
		if err != nil {
			return currentNodes, err
		}
		return desiredNodesAfterScaleActions, nil
	}

	// We evaluate scale down when the number of current nodes is strictly greater
	// than the minimum number of nodes needed to support the current workload
	// and when all the scale down thresholds have been crossed
	scaleDownNeeded, err := d.scaleDownNeeded(cluster, status)
	if err != nil {
		return currentNodes, err
	}
	if scaleDownNeeded {
		nodesToScaleDown := d.calculateDesiredNodesToScaleDown(cluster, status)
		desiredNodesAfterScaleActions = currentNodes - nodesToScaleDown
		if desiredNodesAfterScaleActions > 0 {
			glog.V(10).Infof("Decreasing reported current number of nodes '%v' by '%v'", currentNodes, nodesToScaleDown)
			_, err := d.clusterService.SetComputeNodes(cluster.ClusterID, desiredNodesAfterScaleActions)
			if err != nil {
				return currentNodes, err
			}
		}
	}

	return desiredNodesAfterScaleActions, nil
}

func (d *dataPlaneClusterService) setClusterStatus(cluster *api.Cluster, status *dbapi.DataPlaneClusterStatus) error {
	remainingCapacity := true
	if d.dataplaneClusterConfig.IsDataPlaneAutoScalingEnabled() {
		var err error
		remainingCapacity, err = d.kafkaClustersCapacityAvailable(status, d.minimumKafkaCapacity())
		if err != nil {
			return err
		}
	}

	if remainingCapacity && cluster.Status != api.ClusterReady {
		clusterIsWaitingForFleetShardOperator := cluster.Status == api.ClusterWaitingForKasFleetShardOperator
		err := d.clusterService.UpdateStatus(*cluster, api.ClusterReady)
		if err != nil {
			return err
		}
		if clusterIsWaitingForFleetShardOperator {
			metrics.UpdateClusterCreationDurationMetric(metrics.JobTypeClusterCreate, time.Since(cluster.CreatedAt))
		}
		metrics.UpdateClusterStatusSinceCreatedMetric(*cluster, api.ClusterReady)
	}

	if !remainingCapacity {
		var desiredStatus api.ClusterStatus
		if status.NodeInfo.Current >= d.getRestrictedCeiling(cluster, status) {
			desiredStatus = api.ClusterFull
		} else {
			desiredStatus = api.ClusterComputeNodeScalingUp
		}

		if cluster.Status != desiredStatus {
			err := d.clusterService.UpdateStatus(*cluster, desiredStatus)
			if err != nil {
				return err
			}
		}
		metrics.UpdateClusterStatusSinceCreatedMetric(*cluster, desiredStatus)
	}

	return nil
}

func (d *dataPlaneClusterService) isFleetShardOperatorReady(status *dbapi.DataPlaneClusterStatus) (bool, error) {
	for _, cond := range status.Conditions {
		if cond.Type == dataPlaneClusterStatusCondReadyName {
			condVal, err := strconv.ParseBool(cond.Status)
			if err != nil {
				return false, err
			}
			return condVal, nil
		}
	}
	return false, nil
}

func (d *dataPlaneClusterService) clusterCanProcessStatusReports(cluster *api.Cluster) bool {
	return cluster.Status == api.ClusterReady ||
		cluster.Status == api.ClusterComputeNodeScalingUp ||
		cluster.Status == api.ClusterFull ||
		cluster.Status == api.ClusterWaitingForKasFleetShardOperator
}

// calculateDesiredNodesToScaleUp returns the desired number of nodes to scale
// up based on the reported status and the cluster characteristics and
// restrictions. The result of this method should only be used
// when it's been checked that scale up can be performed with the scaleUpNeeded
// method
func (d *dataPlaneClusterService) calculateDesiredNodesToScaleUp(cluster *api.Cluster, status *dbapi.DataPlaneClusterStatus) int {
	nodesToScaleUp := status.ResizeInfo.NodeDelta
	if cluster.MultiAZ {
		// We round-up to the nearest Multi-AZ node multiple
		nodesToScaleUp = shared.RoundUp(nodesToScaleUp, multiAZClusterNodeScalingMultiple)
	}
	return nodesToScaleUp
}

// calculateDesiredNodesToScaleDown returns the desired number of nodes to
// scale down based on the reported status and the cluster characteristics
// and restrictions. The result of this method should only be used
// when it's been checked that scale down can be performed with the scaleDownNeeded
// method
func (d *dataPlaneClusterService) calculateDesiredNodesToScaleDown(cluster *api.Cluster, status *dbapi.DataPlaneClusterStatus) int {
	nodesToScaleDown := status.ResizeInfo.NodeDelta
	// We round-up to the nearest Multi-AZ node multiple
	if cluster.MultiAZ {
		nodesToScaleDown = shared.RoundUp(nodesToScaleDown, multiAZClusterNodeScalingMultiple)
	}

	return nodesToScaleDown
}

// scaleUpNeeded returns whether the cluster needs to be scaled up based on
// the reported status.
func (d *dataPlaneClusterService) scaleUpNeeded(cluster *api.Cluster, status *dbapi.DataPlaneClusterStatus) (bool, error) {
	anyScaleUpthresholdCrossed, err := d.anyComputeNodesScaleUpThresholdsCrossed(status)
	if err != nil {
		return false, err
	}

	restrictedCeiling := d.getRestrictedCeiling(cluster, status)
	nodesToScaleUp := d.calculateDesiredNodesToScaleUp(cluster, status)
	desiredNodesAfterScaleUp := status.NodeInfo.Current + nodesToScaleUp
	nodesToScaleUpIsWithinLimits := desiredNodesAfterScaleUp <= restrictedCeiling

	// We detect scaling up as needed when the number of current nodes is strictly
	// less than the ceiling and at least one of the scale up thresholds has
	// been crossed
	return anyScaleUpthresholdCrossed && status.NodeInfo.Current < restrictedCeiling && nodesToScaleUpIsWithinLimits, nil
}

// scaleDownNeeded returns whether the cluster needs to be scaled down based
// on the reported status.
func (d *dataPlaneClusterService) scaleDownNeeded(cluster *api.Cluster, status *dbapi.DataPlaneClusterStatus) (bool, error) {
	allScaleDownThresholdsCrossed, err := d.allComputeNodesScaleDownThresholdsCrossed(status)
	if err != nil {
		return false, err
	}

	restrictedFloor := d.getRestrictedFloor(cluster, status)
	nodesToScaleDown := d.calculateDesiredNodesToScaleDown(cluster, status)
	desiredNodesAfterScaleDown := status.NodeInfo.Current - nodesToScaleDown

	nodesToScaleDownIsWithinLimits := desiredNodesAfterScaleDown > 0 &&
		desiredNodesAfterScaleDown >= status.NodeInfo.CurrentWorkLoadMinimum &&
		desiredNodesAfterScaleDown >= restrictedFloor &&
		desiredNodesAfterScaleDown >= multiAZClusterNodeScalingMultiple // Is this last condition needed?

	// We detect scaling down as needed when the number of current nodes is
	// strictly higher than the number of nodes needed to support the current
	// workload, all of the scale down thresholds have been crossed and that the nodes
	// to scale down are within the limits
	return status.NodeInfo.Current > status.NodeInfo.CurrentWorkLoadMinimum &&
		allScaleDownThresholdsCrossed &&
		nodesToScaleDownIsWithinLimits, nil
}

// getRestrictedCeiling returns the maximum ceiling value that can be used
// according to Multi-AZ restrictions in case the cluster is Multi-AZ. The
// ceiling value is rounded-down to the nearest Multi-AZ node multiple
func (d *dataPlaneClusterService) getRestrictedCeiling(cluster *api.Cluster, status *dbapi.DataPlaneClusterStatus) int {
	restrictedCeiling := status.NodeInfo.Ceiling
	if cluster.MultiAZ {
		restrictedCeiling = shared.RoundDown(restrictedCeiling, multiAZClusterNodeScalingMultiple)
	}
	return restrictedCeiling
}

// getRestrictedCeiling returns the minimum ceiling value that can be used
// according to Multi-AZ restrictions in case the cluster is Multi-AZ. The
// floor value is rounded-up to the nearest Multi-AZ node multiple
func (d *dataPlaneClusterService) getRestrictedFloor(cluster *api.Cluster, status *dbapi.DataPlaneClusterStatus) int {
	restrictedFloor := status.NodeInfo.Floor
	if cluster.MultiAZ {
		restrictedFloor = shared.RoundUp(restrictedFloor, multiAZClusterNodeScalingMultiple)
	}
	return restrictedFloor
}

// anyComputeNodesScaleUpThresholdsCrossed returns true when at least one of the
// reported remaining kafka attributes is smaller than its corresponding
// scale-up capacity threshold
func (d *dataPlaneClusterService) anyComputeNodesScaleUpThresholdsCrossed(status *dbapi.DataPlaneClusterStatus) (bool, error) {
	scaleUpThresholds := d.scaleUpThresholds(status)
	connectionsThresholdCrossed := status.Remaining.Connections < scaleUpThresholds.Connections
	partitionsThresholdCrossed := status.Remaining.Partitions < scaleUpThresholds.Partitions

	return connectionsThresholdCrossed || partitionsThresholdCrossed, nil
}

// allComputeNodesScaleDownThresholdsCrossed returns true when all of the reported remaining
// kafka attributes is higher than its corresponding scale-down capacity threshold
func (d *dataPlaneClusterService) allComputeNodesScaleDownThresholdsCrossed(status *dbapi.DataPlaneClusterStatus) (bool, error) {
	scaleDownThresholds := d.scaleDownThresholds(status)
	connectionsThresholdCrossed := status.Remaining.Connections > scaleDownThresholds.Connections
	partitionsThresholdCrossed := status.Remaining.Partitions > scaleDownThresholds.Partitions

	return connectionsThresholdCrossed && partitionsThresholdCrossed, nil
}

// kafkaClustersCapacityAvailable returns true when all the reported remaining
// kafka attributes are higher than their corresponding capacity attribute
// provided in minCapacity
func (d *dataPlaneClusterService) kafkaClustersCapacityAvailable(status *dbapi.DataPlaneClusterStatus, minCapacity *dataPlaneComputeNodesKafkaCapacityAttributes) (bool, error) {
	remainingConnectionsCapacity := status.Remaining.Connections
	minConnectionsCapacity := minCapacity.Connections
	connectionsCapacityAvailable := remainingConnectionsCapacity >= minConnectionsCapacity

	remainingPartitionsCapacity := status.Remaining.Partitions
	minPartitionsCapacity := minCapacity.Partitions
	partitionsCapacityAvailable := remainingPartitionsCapacity >= minPartitionsCapacity

	return connectionsCapacityAvailable && partitionsCapacityAvailable, nil
}

func (d *dataPlaneClusterService) scaleDownThresholds(status *dbapi.DataPlaneClusterStatus) *dataPlaneComputeNodesKafkaCapacityAttributes {
	var res *dataPlaneComputeNodesKafkaCapacityAttributes = &dataPlaneComputeNodesKafkaCapacityAttributes{}

	res.Connections = status.ResizeInfo.Delta.Connections
	res.Partitions = status.ResizeInfo.Delta.Partitions

	return res
}

func (d *dataPlaneClusterService) scaleUpThresholds(status *dbapi.DataPlaneClusterStatus) *dataPlaneComputeNodesKafkaCapacityAttributes {
	return d.minimumKafkaCapacity()
}

// minimumKafkaCapacity returns the minimum Kafka Capacity attributes needed
// to consider that a kafka cluster has capacity available
func (d *dataPlaneClusterService) minimumKafkaCapacity() *dataPlaneComputeNodesKafkaCapacityAttributes {
	return &dataPlaneComputeNodesKafkaCapacityAttributes{
		Connections: d.kafkaConfig.KafkaCapacity.TotalMaxConnections,
		Partitions:  d.kafkaConfig.KafkaCapacity.MaxPartitions,
	}
}
