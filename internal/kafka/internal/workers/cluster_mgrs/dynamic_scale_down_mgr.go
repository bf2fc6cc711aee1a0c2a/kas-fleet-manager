package cluster_mgrs

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	fleeterrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
	"github.com/golang/glog"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/google/uuid"
)

const (
	dynamicScaleDownWorkerType = "dynamic_scale_down"
)

type DynamicScaleDownManager struct {
	workers.BaseWorker

	dataplaneClusterConfig *config.DataplaneClusterConfig
	clusterProvidersConfig *config.ProviderConfig
	kafkaConfig            *config.KafkaConfig
	clusterService         services.ClusterService
}

var _ workers.Worker = &DynamicScaleDownManager{}

func NewDynamicScaleDownManager(
	reconciler workers.Reconciler,
	dataplaneClusterConfig *config.DataplaneClusterConfig,
	clusterProvidersConfig *config.ProviderConfig,
	kafkaConfig *config.KafkaConfig,
	clusterService services.ClusterService,
) *DynamicScaleDownManager {

	return &DynamicScaleDownManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: dynamicScaleDownWorkerType,
			Reconciler: reconciler,
		},

		dataplaneClusterConfig: dataplaneClusterConfig,
		clusterProvidersConfig: clusterProvidersConfig,
		kafkaConfig:            kafkaConfig,
		clusterService:         clusterService,
	}
}

func (m *DynamicScaleDownManager) Start() {
	m.StartWorker(m)
}

func (m *DynamicScaleDownManager) Stop() {
	m.StopWorker(m)
}

func (m *DynamicScaleDownManager) Reconcile() []error {
	var errList fleeterrors.ErrorList
	if !m.dataplaneClusterConfig.IsDataPlaneAutoScalingEnabled() {
		glog.Infoln("dynamic scaling is disabled. Dynamic scale down reconcile event skipped")
		return nil
	}

	glog.Infoln("running dynamic scale down reconcile event")

	err := m.processDynamicScaleDownReconcileEvent()
	if err != nil {
		errList.AddErrors(err)
	}

	glog.Infoln("dynamic scale down reconcile event finished")
	return errList.ToErrorSlice()
}

func (m *DynamicScaleDownManager) processDynamicScaleDownReconcileEvent() error {
	var errList fleeterrors.ErrorList
	kafkaStreamingUnitCountPerClusterList, err := m.clusterService.FindStreamingUnitCountByClusterAndInstanceType()

	if err != nil {
		errList.AddErrors(err)
		return errList
	}

	type processed struct {
		indexesOfStreamingUnitForSameClusterID []int
		processed                              bool
	}

	var alreadyProcessedClusters map[string]processed = map[string]processed{}

	for i, kafkaStreamingUnitCountPerCluster := range kafkaStreamingUnitCountPerClusterList {
		clusterID := kafkaStreamingUnitCountPerCluster.ClusterId
		existing, ok := alreadyProcessedClusters[clusterID]

		if !ok {
			alreadyProcessedClusters[clusterID] = processed{
				// consider non ready cluster as already processed for scale down evaluation and thus they'll be skipped from scale down consideration
				processed:                              kafkaStreamingUnitCountPerCluster.Status != api.ClusterReady.String(),
				indexesOfStreamingUnitForSameClusterID: []int{i},
			}
		} else {
			alreadyProcessedClusters[clusterID] = processed{
				indexesOfStreamingUnitForSameClusterID: append(existing.indexesOfStreamingUnitForSameClusterID, i),
				processed:                              existing.processed,
			}
		}
	}

	for _, suCount := range kafkaStreamingUnitCountPerClusterList {
		clusterID := suCount.ClusterId
		existing, alreadyProcessed := alreadyProcessedClusters[clusterID]
		if alreadyProcessed && existing.processed { // skip already processed
			glog.V(10).Infof("cluster with cluster_id %s is already processed", suCount.ClusterId)
			continue
		}

		alreadyProcessedClusters[clusterID] = processed{
			indexesOfStreamingUnitForSameClusterID: existing.indexesOfStreamingUnitForSameClusterID,
			processed:                              true,
		}

		var regionsSupportedInstanceType config.InstanceTypeMap
		provider, ok := m.clusterProvidersConfig.ProvidersConfig.SupportedProviders.GetByName(suCount.CloudProvider)
		if ok {
			region, regionFound := provider.Regions.GetByName(suCount.Region)
			if regionFound {
				regionsSupportedInstanceType = region.SupportedInstanceTypes
			}
		}

		var dynamicScaleDownProcessor dynamicScaleDownProcessor = &standardDynamicScaleDownProcessor{
			kafkaStreamingUnitCountPerClusterList:  kafkaStreamingUnitCountPerClusterList,
			regionsSupportedInstanceType:           regionsSupportedInstanceType,
			supportedKafkaInstanceTypesConfig:      &m.kafkaConfig.SupportedInstanceTypes.Configuration,
			clusterService:                         m.clusterService,
			dryRun:                                 !m.dataplaneClusterConfig.EnableDynamicScaleDownManagerScaleDownTrigger,
			clusterID:                              clusterID,
			indexesOfStreamingUnitForSameClusterID: existing.indexesOfStreamingUnitForSameClusterID,
		}

		glog.Infof("evaluating dynamic scale down for cluster '%s'", clusterID)
		shouldScaleDown, err := dynamicScaleDownProcessor.ShouldScaleDown()
		if err != nil {
			errList.AddErrors(err)
			continue
		}
		if shouldScaleDown {
			glog.Infof("data plane scale down need detected for cluster '%s'", clusterID)
			err := dynamicScaleDownProcessor.ScaleDown()
			if err != nil {
				errList.AddErrors(err)
				continue
			}
		}
	}

	if errList.IsEmpty() {
		return nil
	}

	return errList
}

// dynamicScaleDownExecutor is able to perform dynamic ScaleDown execution actions
type dynamicScaleDownExecutor interface {
	ScaleDown() error
}

// dynamicScaleDownEvaluator is able to perform dynamic ScaleDown evaluation actions
type dynamicScaleDownEvaluator interface {
	ShouldScaleDown() (bool, error)
}

// dynamicScaleDownProcessor is able to process dynamic ScaleDown reconcile events
type dynamicScaleDownProcessor interface {
	dynamicScaleDownExecutor
	dynamicScaleDownEvaluator
}

// standardDynamicScaleDownProcessor is the default dynamicScaleDownProcessor
// used when dynamic scaling is enabled.
// It assumes the provided kafkaStreamingUnitCountPerClusterList does not
// contain any element with a Status attribute with value 'failed'
type standardDynamicScaleDownProcessor struct {
	clusterID                    string
	regionsSupportedInstanceType config.InstanceTypeMap
	// kafkaStreamingUnitCountPerClusterList must not contain any element
	// with a Status attribute with value 'failed'
	kafkaStreamingUnitCountPerClusterList services.KafkaStreamingUnitCountPerClusterList
	// indexesOfStreamingUnitForSameClusterID is the index of the straming unit in the "kafkaStreamingUnitCountPerClusterList" that has the
	// same "clusterID" as the one that's current being processed
	indexesOfStreamingUnitForSameClusterID []int
	supportedKafkaInstanceTypesConfig      *config.SupportedKafkaInstanceTypesConfig
	clusterService                         services.ClusterService

	// dryRun controls whether the ScaleDown method performs real actions.
	// Useful when you don't want to trigger a real scale down.
	dryRun bool
}

var _ dynamicScaleDownEvaluator = &standardDynamicScaleDownProcessor{}
var _ dynamicScaleDownExecutor = &standardDynamicScaleDownProcessor{}
var _ dynamicScaleDownProcessor = &standardDynamicScaleDownProcessor{}

// ShouldScaleDown indicates whether a data plane cluster can de deprovisioned.
// It returns true if all the following conditions happen:
// 1. If specified the cluster is empty i.e it does not contain any streaming unit
// 2. If the cluster can be removed without triggering a scale up action
// Otherwise false is returned.
// Note:
// 1. This method assumes kafkaStreamingUnitCountPerClusterList does not
// contain elements with the Status attribute with the 'failed' value.
// Thus, if the type is constructed with the assumptions being true, it
// can be considered as the 'failed' state Clusters are not included in
// the calculations.
// 2. Clusters in deprovisioning and cleanup state are excluded, as clusters into those states don't accept kafka instances anymore.
// 3. Clusters that are still not ready to accept kafka instance are also excluded from the capacity calculation
func (p *standardDynamicScaleDownProcessor) ShouldScaleDown() (bool, error) {
	// First let's check if the cluster is empty
	for _, i := range p.indexesOfStreamingUnitForSameClusterID {
		clusterIsNotEmpty := p.kafkaStreamingUnitCountPerClusterList[i].Count > 0
		if clusterIsNotEmpty {
			glog.Infof("cluster with cluster_id %s is not empty. It is not going to be removed", p.clusterID)
			return false, nil
		}
	}

	// let's check if the cluster can be safely removed without causing a scale up event
	if len(p.regionsSupportedInstanceType) == 0 { // if no region limits are available it means that this cluster is in a region that's not supported anymore, we can safely delete it if it is empty
		glog.Infof("no region limits are available. cluster with cluster_id %s is going to be removed as it is empty", p.clusterID)
		return true, nil
	}

	newkafkaStreamingUnitCountPerClusterList := services.KafkaStreamingUnitCountPerClusterList{}

	clusterStatesTowardReadyState := []string{
		api.ClusterProvisioning.String(), api.ClusterProvisioned.String(),
		api.ClusterAccepted.String(), api.ClusterWaitingForKasFleetShardOperator.String(),
	}

	for _, suCount := range p.kafkaStreamingUnitCountPerClusterList {
		// Ignore clusters in terraforming states for scale up evaluation as they are not ready yet.
		// We only want to delete the cluster if we've a sibling ready cluster
		if arrays.Contains(clusterStatesTowardReadyState, suCount.Status) {
			glog.V(10).Infof("ignoring cluster with cluster_id %s in terraforming state %s:", suCount.ClusterId, suCount.Status)
			continue
		}

		// Keep only clusters streaming unit count for cluster that are not going to be deleted.
		// i.e Assume that the candidate cluster is removed from the new list which will be used to perform scale up evaluation
		if suCount.ClusterId == p.clusterID {
			glog.V(10).Infof("skipping candidate cluster with cluster_id %s:", suCount.ClusterId)
			continue
		}

		newkafkaStreamingUnitCountPerClusterList = append(newkafkaStreamingUnitCountPerClusterList, suCount)
	}

	for _, i := range p.indexesOfStreamingUnitForSameClusterID {
		suCount := p.kafkaStreamingUnitCountPerClusterList[i]

		currLocator := supportedInstanceTypeLocator{
			provider:         suCount.CloudProvider,
			region:           suCount.Region,
			instanceTypeName: suCount.InstanceType,
		}

		instanceTypeConfig, ok := p.regionsSupportedInstanceType[suCount.InstanceType]
		if !ok {
			continue
		}

		dynamicScaleUpProcessor := &standardDynamicScaleUpProcessor{
			locator:                               currLocator,
			instanceTypeConfig:                    &instanceTypeConfig,
			kafkaStreamingUnitCountPerClusterList: newkafkaStreamingUnitCountPerClusterList,
			supportedKafkaInstanceTypesConfig:     p.supportedKafkaInstanceTypesConfig,
			clusterService:                        p.clusterService,
			dryRun:                                true,
		}

		glog.Infof("evaluating whether deleting the cluster with cluster_id %s would trigger scale up for locator '%+v'", p.clusterID, currLocator)
		shouldScaleUp, err := dynamicScaleUpProcessor.ShouldScaleUp()
		if err != nil {
			glog.Infof("scale up evaluation results returned an error. Not removing cluster with cluster_id %s:", suCount.ClusterId)
			return false, err
		}

		if shouldScaleUp {
			glog.Infof("scale up will be needed if the cluster with cluster_id %s is removed. The decision is not to remove it", suCount.ClusterId)
			return false, nil
		}
	}

	return true, nil
}

// ScaleDown marks the cluster as deprovisioning.
func (p *standardDynamicScaleDownProcessor) ScaleDown() error {
	if p.dryRun {
		glog.Infof("scale down running in dryRun mode. No action is taken for cluster with cluster_id %s.", p.clusterID)
		return nil
	}

	glog.Infof("marking the cluster with cluster_id %s as deprovisioning", p.clusterID)
	err := p.clusterService.UpdateStatus(api.Cluster{ClusterID: p.clusterID}, api.ClusterDeprovisioning)
	if err != nil {
		glog.Infof("marking the cluster with cluster_id %s as deprovisioning returned without errors", p.clusterID)
		return err
	}

	// now we mark the in memory streaming unit as deprosivisioning so that they won't be considered for capacity calculation anymore
	for _, i := range p.indexesOfStreamingUnitForSameClusterID {
		p.kafkaStreamingUnitCountPerClusterList[i].Status = api.ClusterDeprovisioning.String()
	}

	glog.Infof("cluster with cluster_id %s marked as 'deprovisioning' successfully", p.clusterID)
	return nil
}
