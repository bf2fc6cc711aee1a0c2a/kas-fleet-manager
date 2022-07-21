package cluster_mgrs

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
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

	ClusterService services.ClusterService
}

var _ workers.Worker = &DynamicScaleUpManager{}

func NewDynamicScaleUpManager(
	reconciler workers.Reconciler,
	dataplaneClusterConfig *config.DataplaneClusterConfig,
	clusterProvidersConfig *config.ProviderConfig,
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
	if !m.DataplaneClusterConfig.IsDataPlaneAutoScalingEnabled() {
		glog.Infoln("dynamic scaling is disabled. Dynamic scale up reconcile event skipped")
		return nil
	}
	glog.Infoln("running dynamic scale up reconcile event")
	defer m.logReconcileEventEnd()

	errs := m.reconcileClustersForRegions()

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
