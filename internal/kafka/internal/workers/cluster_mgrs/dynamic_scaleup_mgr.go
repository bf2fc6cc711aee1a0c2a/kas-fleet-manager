package cluster_mgrs

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	fleeterrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/golang/glog"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/google/uuid"
)

const (
	dynamicScaleUpWorkerType = "dynamic_scale_up"
)

type DynamicScaleUpManager struct {
	workers.BaseWorker

	DataplaneClusterConfig *config.DataplaneClusterConfig
	ClusterProvidersConfig *config.ProviderConfig
	KafkaConfig            *config.KafkaConfig

	ClusterService services.ClusterService
}

var _ workers.Worker = &DynamicScaleUpManager{}

func NewDynamicScaleUpManager(
	reconciler workers.Reconciler,
	dataplaneClusterConfig *config.DataplaneClusterConfig,
	clusterProvidersConfig *config.ProviderConfig,
	kafkaConfig *config.KafkaConfig,
	clusterService services.ClusterService,
) *DynamicScaleUpManager {

	return &DynamicScaleUpManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: dynamicScaleUpWorkerType,
			Reconciler: reconciler,
		},

		DataplaneClusterConfig: dataplaneClusterConfig,
		ClusterProvidersConfig: clusterProvidersConfig,
		KafkaConfig:            kafkaConfig,

		ClusterService: clusterService,
	}
}

func (m *DynamicScaleUpManager) Start() {
	m.StartWorker(m)
}

func (m *DynamicScaleUpManager) Stop() {
	m.StopWorker(m)
}

func (m *DynamicScaleUpManager) Reconcile() []error {
	var errList fleeterrors.ErrorList
	if !m.DataplaneClusterConfig.IsDataPlaneAutoScalingEnabled() {
		glog.Infoln("dynamic scaling is disabled. Dynamic scale up reconcile event skipped")
		return nil
	}

	glog.Infoln("running dynamic scale up reconcile event")

	err := m.processDynamicScaleUpReconcileEvent()
	if err != nil {
		errList.AddErrors(err)
	}

	glog.Infoln("dynamic scale up reconcile event finished")
	return errList.ToErrorSlice()
}

func (m *DynamicScaleUpManager) processDynamicScaleUpReconcileEvent() error {
	var errList fleeterrors.ErrorList
	kafkaStreamingUnitCountPerClusterList, err := m.ClusterService.FindStreamingUnitCountByClusterAndInstanceType()
	if err != nil {
		errList.AddErrors(err)
		return errList
	}

	for _, provider := range m.ClusterProvidersConfig.ProvidersConfig.SupportedProviders {
		for _, region := range provider.Regions {
			for supportedInstanceTypeName := range region.SupportedInstanceTypes {
				currLocator := supportedInstanceTypeLocator{
					provider:         provider.Name,
					region:           region.Name,
					instanceTypeName: supportedInstanceTypeName,
				}
				supportedInstanceTypeConfig := region.SupportedInstanceTypes[supportedInstanceTypeName]
				var dynamicScaleUpProcessor dynamicScaleUpProcessor = &standardDynamicScaleUpProcessor{
					locator:                               currLocator,
					instanceTypeConfig:                    &supportedInstanceTypeConfig,
					kafkaStreamingUnitCountPerClusterList: kafkaStreamingUnitCountPerClusterList,
					supportedKafkaInstanceTypesConfig:     &m.KafkaConfig.SupportedInstanceTypes.Configuration,
					clusterService:                        m.ClusterService,
					dryRun:                                !m.DataplaneClusterConfig.EnableDynamicScaleUpManagerScaleUpTrigger,
				}
				glog.Infof("evaluating dynamic scale up for locator '%+v'", currLocator)
				shouldScaleUp, err := dynamicScaleUpProcessor.ShouldScaleUp()
				if err != nil {
					errList.AddErrors(err)
					continue
				}
				if shouldScaleUp {
					glog.Infof("data plane scale up need detected for locator '%+v'", currLocator)
					err := dynamicScaleUpProcessor.ScaleUp()
					if err != nil {
						errList.AddErrors(err)
						continue
					}
				}
			}
		}
	}

	if errList.IsEmpty() {
		return nil
	}

	return errList
}

// supportedInstanceTypeLocator is a data structure
// that contains all the information to help locate a
// supported instance type in a region's cluster
type supportedInstanceTypeLocator struct {
	provider         string
	region           string
	instanceTypeName string
}

func (l *supportedInstanceTypeLocator) Equal(other supportedInstanceTypeLocator) bool {
	return l.provider == other.provider &&
		l.region == other.region &&
		l.instanceTypeName == other.instanceTypeName
}

// dynamicScaleUpExecutor is able to perform dynamic ScaleUp execution actions
type dynamicScaleUpExecutor interface {
	ScaleUp() error
}

// dynamicScaleUpEvaluator is able to perform dynamic ScaleUp evaluation actions
type dynamicScaleUpEvaluator interface {
	ShouldScaleUp() (bool, error)
}

// dynamicScaleUpProcessor is able to process dynamic ScaleUp reconcile events
type dynamicScaleUpProcessor interface {
	dynamicScaleUpExecutor
	dynamicScaleUpEvaluator
}

// noopDynamicScaleUpProcessor is a dynamicScaleUpProcessor where
// the scale up is a noop and it always returns that no scale up action
// is needed
type noopDynamicScaleUpProcessor struct {
}

var _ dynamicScaleUpEvaluator = &noopDynamicScaleUpProcessor{}
var _ dynamicScaleUpExecutor = &noopDynamicScaleUpProcessor{}
var _ dynamicScaleUpProcessor = &noopDynamicScaleUpProcessor{}

func (p *noopDynamicScaleUpProcessor) ScaleUp() error {
	return nil
}

func (p *noopDynamicScaleUpProcessor) ShouldScaleUp() (bool, error) {
	return false, nil
}

// standardDynamicScaleUpProcessor is the default dynamicScaleUpProcessor
// used when dynamic scaling is enabled.
// It assumes the provided kafkaStreamingUnitCountPerClusterList does not
// contain any element with a Status attribute with value 'failed'
type standardDynamicScaleUpProcessor struct {
	locator            supportedInstanceTypeLocator
	instanceTypeConfig *config.InstanceTypeConfig
	// kafkaStreamingUnitCountPerClusterList must not contain any element
	// with a Status attribute with value 'failed'
	kafkaStreamingUnitCountPerClusterList services.KafkaStreamingUnitCountPerClusterList
	supportedKafkaInstanceTypesConfig     *config.SupportedKafkaInstanceTypesConfig
	clusterService                        services.ClusterService

	// dryRun controls whether the ScaleUp method performs real actions.
	// Useful when you don't want to trigger a real scale up.
	dryRun bool
}

var _ dynamicScaleUpEvaluator = &standardDynamicScaleUpProcessor{}
var _ dynamicScaleUpExecutor = &standardDynamicScaleUpProcessor{}
var _ dynamicScaleUpProcessor = &standardDynamicScaleUpProcessor{}

// ShouldScaleUp indicates whether a new data plane
// cluster should be created for a given instance type in the given provider
// and region.
// It returns true if all the following conditions happen:
//  1. If specified, the streaming units limit for the given instance type in
//     the provider's region has not been reached
//  2. There is no scale up action ongoing. A scale up action is ongoing
//     if there is at least one cluster in the following states: 'provisioning',
//     'provisioned', 'accepted', 'waiting_for_kas_fleetshard_operator'
//  3. At least one of the two following conditions are true:
//     * No cluster in the provider's region has enough capacity to allocate
//     the biggest instance size of the given instance type
//     * The free capacity (in streaming units) for the given instance type in
//     the provider's region is smaller or equal than the defined slack
//     capacity (also in streaming units) of the given instance type. Free
//     capacity is defined as max(total) capacity - consumed capacity.
//     For the calculation of the max capacity:
//     * Clusters in deprovisioning and cleanup state are excluded, as
//     clusters into those states don't accept kafka instances anymore.
//     * Clusters that are still not ready to accept kafka instance but that
//     should eventually accept them (like accepted state for example)
//     are included
//
// Otherwise false is returned.
// Note: This method assumes kafkaStreamingUnitCountPerClusterList does not
//
//	contain elements with the Status attribute with the 'failed' value.
//	Thus, if the type is constructed with the assumptions being true, it
//	can be considered as the 'failed' state Clusters are not included in
//	the calculations.
func (p *standardDynamicScaleUpProcessor) ShouldScaleUp() (bool, error) {
	summaryCalculator := instanceTypeConsumptionSummaryCalculator{
		locator:                               p.locator,
		kafkaStreamingUnitCountPerClusterList: p.kafkaStreamingUnitCountPerClusterList,
		supportedKafkaInstanceTypesConfig:     p.supportedKafkaInstanceTypesConfig,
	}

	instanceTypeConsumptionInRegionSummary, err := summaryCalculator.Calculate()
	if err != nil {
		return false, err
	}
	glog.Infof("consumption summary for locator '%+v': '%+v'", p.locator, instanceTypeConsumptionInRegionSummary)

	regionLimitReached := p.regionLimitReached(instanceTypeConsumptionInRegionSummary)
	if regionLimitReached {
		glog.Infof("region limit for locator '%+v' reached. No cluster scale up action should be performed", p.locator)
		return false, nil
	}

	ongoingScaleUpActionInRegion := p.ongoingScaleUpAction(instanceTypeConsumptionInRegionSummary)
	if ongoingScaleUpActionInRegion {
		glog.Infof("ongoing data plane cluster scale up action limit for locator '%+v' reached. No cluster scale up action should be performed", p.locator)
		return false, nil
	}

	freeCapacityForBiggestInstanceSizeInRegion := p.freeCapacityForBiggestInstanceSize(instanceTypeConsumptionInRegionSummary)
	if !freeCapacityForBiggestInstanceSizeInRegion {
		glog.Infof("biggest kafka instance size for locator '%+v' cannot fit in any cluster. Cluster scale up action should be performed", p.locator)
		return true, nil
	}

	enoughCapacitySlackInRegion := p.enoughCapacitySlackInRegion(instanceTypeConsumptionInRegionSummary)
	if !enoughCapacitySlackInRegion {
		glog.Infof("there is not enough capacity slack for locator '%+v'. Cluster scale up action should be performed", p.locator)
		return true, nil
	}

	glog.Infof("no conditions for cluster scale up action have been detected for locator '%+v'. No cluster scale up action should be performed", p.locator)
	return false, nil
}

// ScaleUp triggers a new data plane cluster
// registration for a given instance type in a provider region.
func (p *standardDynamicScaleUpProcessor) ScaleUp() error {
	if p.dryRun {
		glog.Infoln("scale up running in dryRun mode. No action is taken.")
		return nil
	}

	glog.Infof("registering new data plane cluster for locator '%+v'", p.locator)
	// If the provided instance type to support is standard the new cluster
	// to register will be MultiAZ. Otherwise will be single AZ
	newClusterMultiAZ := p.locator.instanceTypeName == api.StandardTypeSupport.String()

	clusterRequest := &api.Cluster{
		CloudProvider:         p.locator.provider,
		Region:                p.locator.region,
		SupportedInstanceType: p.locator.instanceTypeName,
		MultiAZ:               newClusterMultiAZ,
		Status:                api.ClusterAccepted,
		ProviderType:          api.ClusterProviderOCM,
	}
	glog.V(10).Infof("registering new cluster job creation with attributes: %+v", clusterRequest)

	err := p.clusterService.RegisterClusterJob(clusterRequest)
	if err != nil {
		return err
	}
	glog.Infof("cluster creation job for locator '%+v' registered successfully", p.locator)

	return nil
}

func (p *standardDynamicScaleUpProcessor) enoughCapacitySlackInRegion(summary instanceTypeConsumptionSummary) bool {
	freeStreamingUnitsInRegion := summary.freeStreamingUnits
	capacitySlackInRegion := p.instanceTypeConfig.MinAvailableCapacitySlackStreamingUnits
	glog.V(10).Infof("configured minimum capacity slack for locator %+v is: '%v'", p.locator, capacitySlackInRegion)

	// Note: if capacitySlackInRegion is 0 we always return that there is enough
	// capacity slack in region.
	return freeStreamingUnitsInRegion >= capacitySlackInRegion
}

func (p *standardDynamicScaleUpProcessor) regionLimitReached(summary instanceTypeConsumptionSummary) bool {
	streamingUnitsLimitInRegion := p.instanceTypeConfig.Limit
	consumedStreamingUnitsInRegion := summary.consumedStreamingUnits

	if streamingUnitsLimitInRegion == nil {
		glog.V(10).Infof("region limit for locator '%+v' is nil", p.locator)
		return false
	}

	glog.V(10).Infof("region limit in streaming units for locator '%+v': '%+v'", p.locator, *streamingUnitsLimitInRegion)
	return consumedStreamingUnitsInRegion >= *streamingUnitsLimitInRegion
}

func (p *standardDynamicScaleUpProcessor) freeCapacityForBiggestInstanceSize(summary instanceTypeConsumptionSummary) bool {
	return summary.biggestInstanceSizeCapacityAvailable
}

func (p *standardDynamicScaleUpProcessor) ongoingScaleUpAction(summary instanceTypeConsumptionSummary) bool {
	return summary.ongoingScaleUpAction
}

// instanceTypeConsumptionSummary contains a consumption summary
// of an instance in a provider's region
type instanceTypeConsumptionSummary struct {
	// maxStreamingUnits is the total capacity for the instance type in
	// streaming units
	maxStreamingUnits int
	// freeStreamingUnits is the free capacity for the instance type in
	// streaming units
	freeStreamingUnits int
	// consumedStreamingUnits is the consumed capacity for the instance type in
	// streaming units
	consumedStreamingUnits int
	// ongoingScaleUpAction designates whether a data plane cluster in the
	// provider's region, which supports the instance type, is being created
	ongoingScaleUpAction bool
	// biggestInstanceSizeCapacityAvailable indicates whether there is capacity
	// to allocate at least one unit of the biggest kafka instance size of
	// the instance type
	biggestInstanceSizeCapacityAvailable bool
}

// instanceTypeConsumptionSummaryCalculator calculates a consumption summary
// for the provided supportedInstanceTypeLocator based on the
// provided KafkaStreamingUnitCountPerClusterList
type instanceTypeConsumptionSummaryCalculator struct {
	locator                               supportedInstanceTypeLocator
	kafkaStreamingUnitCountPerClusterList services.KafkaStreamingUnitCountPerClusterList
	supportedKafkaInstanceTypesConfig     *config.SupportedKafkaInstanceTypesConfig
}

// Calculate returns a instanceTypeConsumptionSummary containing a consumption
// summary for the provided supportedInstanceTypeLocator
// For the calculation of the max streaming units capacity:
//   - Clusters in deprovisioning and cleanup state are excluded, as
//     clusters into those states don't accept kafka instances anymore.
//   - Clusters that are still not ready to accept kafka instance but that
//     should eventually accept them (like accepted state for example)
//     are included
//
// For the calculation of whether a scale up actions is ongoing:
//   - A scale up action is ongoing if there is at least one cluster in the
//     following states: 'provisioning', 'provisioned', 'accepted',
//     'waiting_for_kas_fleetshard_operator'
func (i *instanceTypeConsumptionSummaryCalculator) Calculate() (instanceTypeConsumptionSummary, error) {
	biggestKafkaInstanceSizeCapacityConsumption, err := i.getBiggestCapacityConsumedSize()
	if err != nil {
		return instanceTypeConsumptionSummary{}, err
	}

	consumedStreamingUnitsInRegion := 0
	maxStreamingUnitsInRegion := 0
	atLeastOneClusterHasCapacityForBiggestInstanceType := false
	scaleUpActionIsOngoing := false

	clusterStatesTowardReadyState := []string{
		api.ClusterProvisioning.String(), api.ClusterProvisioned.String(),
		api.ClusterAccepted.String(), api.ClusterWaitingForKasFleetShardOperator.String(),
	}
	clusterStatesTowardDeletion := []string{api.ClusterDeprovisioning.String(), api.ClusterCleanup.String()}

	for _, kafkaStreamingUnitCountPerCluster := range i.kafkaStreamingUnitCountPerClusterList {
		currLocator := supportedInstanceTypeLocator{
			provider:         kafkaStreamingUnitCountPerCluster.CloudProvider,
			region:           kafkaStreamingUnitCountPerCluster.Region,
			instanceTypeName: kafkaStreamingUnitCountPerCluster.InstanceType,
		}

		if !i.locator.Equal(currLocator) {
			glog.V(10).Infof("cluster consumption '%+v' does not match locator '%+v'. Discarded from summary consumption calculation", kafkaStreamingUnitCountPerCluster, i.locator)
			continue
		}
		glog.V(10).Infof("summary consumption evaluation of cluster consumption '%+v' for locator '%+v'", kafkaStreamingUnitCountPerCluster, i.locator)

		if arrays.Contains(clusterStatesTowardReadyState, kafkaStreamingUnitCountPerCluster.Status) {
			scaleUpActionIsOngoing = true
		}

		if kafkaStreamingUnitCountPerCluster.FreeStreamingUnits() >= int32(biggestKafkaInstanceSizeCapacityConsumption) {
			atLeastOneClusterHasCapacityForBiggestInstanceType = true
		}

		consumedStreamingUnitsInRegion = consumedStreamingUnitsInRegion + int(kafkaStreamingUnitCountPerCluster.Count)
		if !arrays.Contains(clusterStatesTowardDeletion, kafkaStreamingUnitCountPerCluster.Status) {
			maxStreamingUnitsInRegion = maxStreamingUnitsInRegion + int(kafkaStreamingUnitCountPerCluster.MaxUnits)
		}
	}

	freeStreamingUnitsInRegion := maxStreamingUnitsInRegion - consumedStreamingUnitsInRegion

	return instanceTypeConsumptionSummary{
		maxStreamingUnits:                    maxStreamingUnitsInRegion,
		consumedStreamingUnits:               consumedStreamingUnitsInRegion,
		freeStreamingUnits:                   freeStreamingUnitsInRegion,
		ongoingScaleUpAction:                 scaleUpActionIsOngoing,
		biggestInstanceSizeCapacityAvailable: atLeastOneClusterHasCapacityForBiggestInstanceType,
	}, nil
}

func (i *instanceTypeConsumptionSummaryCalculator) getBiggestCapacityConsumedSize() (int, error) {
	kafkaInstanceTypeConfig, err := i.supportedKafkaInstanceTypesConfig.GetKafkaInstanceTypeByID(i.locator.instanceTypeName)
	if err != nil {
		return -1, err
	}
	biggestKafkaInstanceSizeCapacityConsumption := -1
	maxKafkaInstanceSizeConfig := kafkaInstanceTypeConfig.GetBiggestCapacityConsumedSize()
	if maxKafkaInstanceSizeConfig != nil {
		biggestKafkaInstanceSizeCapacityConsumption = maxKafkaInstanceSizeConfig.CapacityConsumed
	}
	glog.V(10).Infof("capacity consumption value of biggest kafka size of locator '%+v': '%v'", i.locator, biggestKafkaInstanceSizeCapacityConsumption)
	return biggestKafkaInstanceSizeCapacityConsumption, nil
}
