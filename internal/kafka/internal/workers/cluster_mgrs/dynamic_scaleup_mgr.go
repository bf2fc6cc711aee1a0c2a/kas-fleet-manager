package cluster_mgrs

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/google/uuid"
)

const (
	DynamicScaleUpWorkerType = "dynamic_scale_up"
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
			WorkerType: DynamicScaleUpWorkerType,
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
	var errs []error
	if !m.DataplaneClusterConfig.IsDataPlaneAutoScalingEnabled() {
		glog.Infoln("dynamic scaling is disabled. Dynamic scale up reconcile event skipped")
		return nil
	}
	glog.Infoln("running dynamic scale up reconcile event")
	defer m.logReconcileEventEnd()

	// TODO remove this method call and the method itself the new dynamic scale up
	// logic is ready
	err := m.reconcileClustersForRegions()
	if err != nil {
		errs = append(errs, err...)
	}

	// TODO remove the "if false" condition once the new dynamic scale up
	// logic is ready
	if false {
		err := m.processDynamicScaleUpReconcileEvent()
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func (m *DynamicScaleUpManager) logReconcileEventEnd() {
	glog.Infoln("dynamic scale up reconcile event finished")
}

// reconcileClustersForRegions creates an OSD cluster for each supported cloud provider and region where no cluster exists.
func (m *DynamicScaleUpManager) reconcileClustersForRegions() []error {
	var errs []error
	glog.Infoln("reconcile cloud providers and regions")
	var providers []string
	var regions []string
	status := api.StatusForValidCluster
	//gather the supported providers and regions
	providerList := m.ClusterProvidersConfig.ProvidersConfig.SupportedProviders
	for _, v := range providerList {
		providers = append(providers, v.Name)
		for _, r := range v.Regions {
			regions = append(regions, r.Name)
		}
	}

	// get a list of clusters in Map group by their provider and region.
	grpResult, err := m.ClusterService.ListGroupByProviderAndRegion(providers, regions, status)
	if err != nil {
		errs = append(errs, errors.Wrapf(err, "failed to find cluster with criteria"))
		return errs
	}

	grpResultMap := make(map[string]*services.ResGroupCPRegion)
	for _, v := range grpResult {
		grpResultMap[v.Provider+"."+v.Region] = v
	}

	// create all the missing clusters in the supported provider and regions.
	for _, p := range providerList {
		for _, v := range p.Regions {
			if _, exist := grpResultMap[p.Name+"."+v.Name]; !exist {
				clusterRequest := api.Cluster{
					CloudProvider:         p.Name,
					Region:                v.Name,
					MultiAZ:               true,
					Status:                api.ClusterAccepted,
					ProviderType:          api.ClusterProviderOCM,
					SupportedInstanceType: api.AllInstanceTypeSupport.String(), // TODO - make sure we use the appropriate instance type.
				}
				if err := m.ClusterService.RegisterClusterJob(&clusterRequest); err != nil {
					errs = append(errs, errors.Wrapf(err, "Failed to auto-create cluster request in %s, region: %s", p.Name, v.Name))
					return errs
				} else {
					glog.Infof("Auto-created cluster request in %s, region: %s, Id: %s ", p.Name, v.Name, clusterRequest.ID)
				}
			} //
		} //region
	} //provider
	return errs
}

func (m *DynamicScaleUpManager) processDynamicScaleUpReconcileEvent() error {
	kafkaStreamingUnitCountPerClusterList, err := m.ClusterService.FindStreamingUnitCountByClusterAndInstanceType()
	if err != nil {
		return err
	}

	for _, provider := range m.ClusterProvidersConfig.ProvidersConfig.SupportedProviders {
		for _, region := range provider.Regions {
			for supportedInstanceTypeName := range region.SupportedInstanceTypes {
				supportedInstanceTypeConfig := region.SupportedInstanceTypes[supportedInstanceTypeName]
				shouldScaleUp, err := m.shouldScaleUpNewClusterForInstanceType(provider.Name, region.Name, supportedInstanceTypeName, &supportedInstanceTypeConfig, kafkaStreamingUnitCountPerClusterList)
				if err != nil {
					return err
				}
				if shouldScaleUp {
					err := m.registerNewClusterForInstanceType(provider.Name, region.Name, supportedInstanceTypeName)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

// shouldScaleUpNewClusterForInstanceType indicates whether a new data plane
// cluster should be created for a given instance type in the given provider
// and region.
// It returns true if all the following conditions happen:
// 1. If specified, the streaming units limit for the given instance type in
//    the provider's region has not been reached
// 2. There is no scale up action ongoing. A scale up action is ongoing
//    if there is at least one cluster in the following states: 'provisioning',
//    'provisioned', 'accepted', 'waiting_for_kas_fleetshard_operator'
// 3. At least one of the two following conditions are true:
//      * No cluster in the provider's region has enough capacity to allocate
//        the biggest instance size of the given instance type
//      * The free capacity (in streaming units) for the given instance type in
//        the provider's region is smaller or equal than the defined slack
//        capacity (also in streaming units) of the given instance type. Free
//        capacity is defined as max(total) capacity - consumed capacity.
//        For the calculation of the max capacity:
//        * Clusters in deprovisioning and cleanup state are excluded, as
//          clusters into those states don't accept kafka instances anymore.
//        * Clusters that are still not ready to accept kafka instance but that
//          should eventually accept them (like accepted state for example)
//          are included
// Otherwise false is returned.
// Note: Clusters in the 'failed' state are excluded from the calculations.
func (m *DynamicScaleUpManager) shouldScaleUpNewClusterForInstanceType(
	provider string, region string,
	instanceTypeName string,
	instanceTypeConfig *config.InstanceTypeConfig,
	kafkaStreamingUnitCountPerClusterList services.KafkaStreamingUnitCountPerClusterList) (bool, error) {

	kafkaInstanceTypeConfig, err := m.KafkaConfig.SupportedInstanceTypes.Configuration.GetKafkaInstanceTypeByID(instanceTypeName)
	if err != nil {
		return false, err
	}
	biggestKafkaInstanceSizeCapacityConsumption := -1
	maxKafkaInstanceSizeConfig := kafkaInstanceTypeConfig.GetBiggestCapacityConsumedSize()
	if maxKafkaInstanceSizeConfig != nil {
		biggestKafkaInstanceSizeCapacityConsumption = maxKafkaInstanceSizeConfig.CapacityConsumed
	}

	consumedStreamingUnitsInRegion := 0
	maxStreamingUnitsInRegion := 0
	atLeastOneClusterHasCapacityForLargestInstanceType := false

	scaleUpActionIsOngoing := false
	clusterStatesTowardReadyState := []string{
		api.ClusterProvisioning.String(), api.ClusterProvisioned.String(),
		api.ClusterAccepted.String(), api.ClusterWaitingForKasFleetShardOperator.String(),
	}

	for _, kafkaStreamingUnitCountPerCluster := range kafkaStreamingUnitCountPerClusterList {
		if kafkaStreamingUnitCountPerCluster.CloudProvider == provider &&
			kafkaStreamingUnitCountPerCluster.Region == region &&
			kafkaStreamingUnitCountPerCluster.InstanceType == instanceTypeName {

			if arrays.Contains(clusterStatesTowardReadyState, kafkaStreamingUnitCountPerCluster.Status) {
				scaleUpActionIsOngoing = true
			}

			if kafkaStreamingUnitCountPerCluster.FreeStreamingUnits() >= int32(biggestKafkaInstanceSizeCapacityConsumption) {
				atLeastOneClusterHasCapacityForLargestInstanceType = true
			}

			consumedStreamingUnitsInRegion = consumedStreamingUnitsInRegion + int(kafkaStreamingUnitCountPerCluster.Count)
			if kafkaStreamingUnitCountPerCluster.Status != api.ClusterDeprovisioning.String() &&
				kafkaStreamingUnitCountPerCluster.Status != api.ClusterCleanup.String() {
				maxStreamingUnitsInRegion = maxStreamingUnitsInRegion + int(kafkaStreamingUnitCountPerCluster.MaxUnits)
			}
		}
	}

	freeStreamingUnitsInRegion := maxStreamingUnitsInRegion - consumedStreamingUnitsInRegion
	capacitySlackInRegion := instanceTypeConfig.MinAvailableCapacitySlackStreamingUnits
	notEnoughCapacitySlackInRegion := false
	if freeStreamingUnitsInRegion < capacitySlackInRegion { // TODO is it <= or < ?
		notEnoughCapacitySlackInRegion = true
	}

	streamingUnitsLimitInRegion := instanceTypeConfig.Limit
	regionLimitReached := false
	if streamingUnitsLimitInRegion != nil && consumedStreamingUnitsInRegion >= *streamingUnitsLimitInRegion {
		regionLimitReached = true
	}

	if !regionLimitReached && !scaleUpActionIsOngoing {
		if !atLeastOneClusterHasCapacityForLargestInstanceType || (atLeastOneClusterHasCapacityForLargestInstanceType && notEnoughCapacitySlackInRegion) {
			return true, nil
		}
	}

	return false, nil
}

// registerNewClusterForInstanceType triggers a new data plane cluster
// registration for a given instance type in a provider region.
func (m *DynamicScaleUpManager) registerNewClusterForInstanceType(provider string, region string, instanceTypeName string) error {
	// If the provided instance type to support is standard the new cluster
	// to register will be MultiAZ. Otherwise will be single AZ
	newClusterMultiAZ := instanceTypeName == api.StandardTypeSupport.String()

	clusterRequest := &api.Cluster{
		CloudProvider:         provider,
		Region:                region,
		SupportedInstanceType: instanceTypeName,
		MultiAZ:               newClusterMultiAZ,
		Status:                api.ClusterAccepted,
		ProviderType:          api.ClusterProviderOCM,
	}

	err := m.ClusterService.RegisterClusterJob(clusterRequest)
	if err != nil {
		return err
	}

	return nil
}
