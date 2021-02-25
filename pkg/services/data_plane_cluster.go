package services

import (
	"context"
	"strconv"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	"github.com/golang/glog"
)

// Number of compute nodes in a Multi-AZ must be a multiple of
// this number
const multiAZClusterNodeScalingMultiple = 3

// singleKafkaCluster*Capacity constants define the mapping
// between a Kafka Cluster of size model T and Kafka capacity attributes
// TODO move this to make it configurable or variable
const singleKafkaClusterConnectionsCapacity = 100
const singleKafkaClusterPartitionsCapacity = 100

// The Kafka Capacity attributes of a Kafka Cluster size model T
// TODO move this to make it configurable or variable
var singleKafkaClusterComputeNodesCapacityAttributes dataPlaneComputeNodesKafkaCapacityAttributes = dataPlaneComputeNodesKafkaCapacityAttributes{
	Connections: singleKafkaClusterConnectionsCapacity,
	Partitions:  singleKafkaClusterPartitionsCapacity,
}

// kafkaCluster*ScaleUpThreshold constants define the scale up thresholds
// for each of the Kafka Capacity attributes reported by KAS Fleet Operator
// They are defined as multiples of the values defined in the singleKafkaCluster*
// constants. Currently set to exactly one Kafka Cluster of size model T
// TODO move this to make it configurable or variable
const kafkaClusterConnectionsCapacityScaleUpThreshold = singleKafkaClusterConnectionsCapacity
const kafkaClusterPartitionsCapacityScaleUpThreshold = singleKafkaClusterPartitionsCapacity

type DataPlaneClusterService interface {
	UpdateDataPlaneClusterStatus(ctx context.Context, clusterID string, status *api.DataPlaneClusterStatus) *errors.ServiceError
}

var _ DataPlaneClusterService = &dataPlaneClusterService{}

const dataPlaneClusterStatusCondReadyName = "Ready"

type dataPlaneClusterService struct {
	ocmClient      ocm.Client
	clusterService ClusterService
}

type dataPlaneComputeNodesKafkaCapacityAttributes struct {
	Connections int
	Partitions  int
}

func NewDataPlaneClusterService(clusterService ClusterService, ocmClient ocm.Client) *dataPlaneClusterService {
	return &dataPlaneClusterService{
		ocmClient:      ocmClient,
		clusterService: clusterService,
	}
}

func (d *dataPlaneClusterService) UpdateDataPlaneClusterStatus(ctx context.Context, clusterID string, status *api.DataPlaneClusterStatus) *errors.ServiceError {
	cluster, svcErr := d.clusterService.FindClusterByID(clusterID)
	if svcErr != nil {
		return svcErr
	}
	if cluster == nil {
		// 404 is used for authenticated requests. So to distinguish the errors, we use 400 here
		return errors.BadRequest("Cluster agent with ID '%s' not found", clusterID)
	}

	if !d.clusterCanProcessStatusReports(cluster) {
		glog.V(10).Infof("Cluster with ID '%s is in '%s' state. Ignoring status report...", clusterID, cluster.Status)
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
		}
		return nil
	}

	// We calculate the status based on the stats received by the KAS Fleet operator
	// BEFORE performing the scaling actions. If scaling actions are performed later
	// then it will be reflected on the next data plane cluster status report
	err = d.setClusterStatus(cluster, status)
	if err != nil {
		return errors.ToServiceError(err)
	}

	_, err = d.updateDataPlaneClusterNodes(cluster, status)
	if err != nil {
		return errors.ToServiceError(err)
	}

	return nil
}

// updateDataPlaneClusterNodes performs node scale-up and scale-down actions on
// the data plane cluster based on the reported status by the kas fleet shard
// operator. The state of the number of desired nodes after the scaling
// actions is returned
func (d *dataPlaneClusterService) updateDataPlaneClusterNodes(cluster *api.Cluster, status *api.DataPlaneClusterStatus) (int, error) {
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
		_, err := d.clusterService.SetComputeNodes(cluster.ID, desiredNodesAfterScaleActions)
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
			_, err := d.clusterService.SetComputeNodes(cluster.ID, desiredNodesAfterScaleActions)
			if err != nil {
				return currentNodes, err
			}
		}
	}

	return desiredNodesAfterScaleActions, nil
}

func (d *dataPlaneClusterService) setClusterStatus(cluster *api.Cluster, status *api.DataPlaneClusterStatus) error {
	remainingCapacity, err := d.kafkaClustersCapacityAvailable(status, &singleKafkaClusterComputeNodesCapacityAttributes)
	if err != nil {
		return err
	}

	if remainingCapacity && cluster.Status != api.ClusterReady {
		err := d.clusterService.UpdateStatus(*cluster, api.ClusterReady)
		if err != nil {
			return err
		}
	}

	if !remainingCapacity && cluster.Status != api.ClusterFull {
		err := d.clusterService.UpdateStatus(*cluster, api.ClusterFull)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *dataPlaneClusterService) isFleetShardOperatorReady(status *api.DataPlaneClusterStatus) (bool, error) {
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
		cluster.Status == api.ClusterFull ||
		cluster.Status == api.ClusterWaitingForKasFleetShardOperator ||
		cluster.Status == api.AddonInstalled
}

// calculateDesiredNodesToScaleUp returns the desired number of nodes to scale
// up based on the reported status and the cluster characteristics and
// restrictions. The result of this method should only be used
// when it's been checked that scale up can be performed with the scaleUpNeeded
// method
func (d *dataPlaneClusterService) calculateDesiredNodesToScaleUp(cluster *api.Cluster, status *api.DataPlaneClusterStatus) int {
	nodesToScaleUp := status.ResizeInfo.NodeDelta
	if cluster.MultiAZ {
		// We round-up to the nearest Multi-AZ node multiple
		nodesToScaleUp = d.roundUp(nodesToScaleUp, multiAZClusterNodeScalingMultiple)
	}
	return nodesToScaleUp
}

// calculateDesiredNodesToScaleDown returns the desired number of nodes to
// scale down based on the reported status and the cluster characteristics
// and restrictions. The result of this method should only be used
// when it's been checked that scale down can be performed with the scaleDownNeeded
// method
func (d *dataPlaneClusterService) calculateDesiredNodesToScaleDown(cluster *api.Cluster, status *api.DataPlaneClusterStatus) int {
	nodesToScaleDown := status.ResizeInfo.NodeDelta
	// We round-up to the nearest Multi-AZ node multiple
	if cluster.MultiAZ {
		nodesToScaleDown = d.roundUp(nodesToScaleDown, multiAZClusterNodeScalingMultiple)
	}

	return nodesToScaleDown
}

// scaleUpNeeded returns whether the cluster needs to be scaled up based on
// the reported status.
func (d *dataPlaneClusterService) scaleUpNeeded(cluster *api.Cluster, status *api.DataPlaneClusterStatus) (bool, error) {
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

// roundUp rounds up number to the nearest multiple. Input has to be positive
func (d *dataPlaneClusterService) roundUp(number int, multiple int) int {
	remainder := number % multiple
	if remainder == 0 {
		return number
	}

	return number + (multiple - remainder)
}

// roundDown rounds up number to the nearest multiple. Input has to be positive
func (d *dataPlaneClusterService) roundDown(number int, multiple int) int {
	return number - (number % multiple)
}

// scaleDownNeeded returns whether the cluster needs to be scaled down based
// on the reported status.
func (d *dataPlaneClusterService) scaleDownNeeded(cluster *api.Cluster, status *api.DataPlaneClusterStatus) (bool, error) {
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
func (d *dataPlaneClusterService) getRestrictedCeiling(cluster *api.Cluster, status *api.DataPlaneClusterStatus) int {
	restrictedCeiling := status.NodeInfo.Ceiling
	if cluster.MultiAZ {
		restrictedCeiling = d.roundDown(restrictedCeiling, multiAZClusterNodeScalingMultiple)
	}
	return restrictedCeiling
}

// getRestrictedCeiling returns the minimum ceiling value that can be used
// according to Multi-AZ restrictions in case the cluster is Multi-AZ. The
// floor value is rounded-up to the nearest Multi-AZ node multiple
func (d *dataPlaneClusterService) getRestrictedFloor(cluster *api.Cluster, status *api.DataPlaneClusterStatus) int {
	restrictedFloor := status.NodeInfo.Floor
	if cluster.MultiAZ {
		restrictedFloor = d.roundUp(restrictedFloor, multiAZClusterNodeScalingMultiple)
	}
	return restrictedFloor
}

// anyComputeNodesScaleUpThresholdsCrossed returns true when at least one of the
// reported remaining kafka attributes is smaller than its corresponding
// scale-up capacity threshold
func (d *dataPlaneClusterService) anyComputeNodesScaleUpThresholdsCrossed(status *api.DataPlaneClusterStatus) (bool, error) {
	scaleUpThresholds := d.scaleUpThresholds(status)
	connectionsThresholdCrossed := status.Remaining.Connections < scaleUpThresholds.Connections
	partitionsThresholdCrossed := status.Remaining.Partitions < scaleUpThresholds.Partitions

	return connectionsThresholdCrossed || partitionsThresholdCrossed, nil
}

// allComputeNodesScaleDownThresholdsCrossed returns true when all of the reported remaining
// kafka attributes is higher than its corresponding scale-down capacity threshold
func (d *dataPlaneClusterService) allComputeNodesScaleDownThresholdsCrossed(status *api.DataPlaneClusterStatus) (bool, error) {
	scaleDownThresholds := d.scaleDownThresholds(status)
	connectionsThresholdCrossed := status.Remaining.Connections > scaleDownThresholds.Connections
	partitionsThresholdCrossed := status.Remaining.Partitions > scaleDownThresholds.Partitions

	return connectionsThresholdCrossed && partitionsThresholdCrossed, nil
}

// kafkaClustersCapacityAvailable returns true when all the reported remaining
// kafka attributes are higher than their corresponding capacity attribute
// provided in minCapacity
func (d *dataPlaneClusterService) kafkaClustersCapacityAvailable(status *api.DataPlaneClusterStatus, minCapacity *dataPlaneComputeNodesKafkaCapacityAttributes) (bool, error) {
	remainingConnectionsCapacity := status.Remaining.Connections
	minConnectionsCapacity := minCapacity.Connections
	connectionsCapacityAvailable := remainingConnectionsCapacity >= minConnectionsCapacity

	remainingPartitionsCapacity := status.Remaining.Partitions
	minPartitionsCapacity := minCapacity.Partitions
	partitionsCapacityAvailable := remainingPartitionsCapacity >= minPartitionsCapacity

	return connectionsCapacityAvailable && partitionsCapacityAvailable, nil
}

func (d *dataPlaneClusterService) scaleDownThresholds(status *api.DataPlaneClusterStatus) *dataPlaneComputeNodesKafkaCapacityAttributes {
	var res *dataPlaneComputeNodesKafkaCapacityAttributes = &dataPlaneComputeNodesKafkaCapacityAttributes{}

	res.Connections = status.ResizeInfo.Delta.Connections
	res.Partitions = status.ResizeInfo.Delta.Partitions

	return res
}

func (d *dataPlaneClusterService) scaleUpThresholds(status *api.DataPlaneClusterStatus) *dataPlaneComputeNodesKafkaCapacityAttributes {
	var res *dataPlaneComputeNodesKafkaCapacityAttributes = &dataPlaneComputeNodesKafkaCapacityAttributes{}

	res.Connections = kafkaClusterConnectionsCapacityScaleUpThreshold
	res.Partitions = kafkaClusterPartitionsCapacityScaleUpThreshold

	return res
}
