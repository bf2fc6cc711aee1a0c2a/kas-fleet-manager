package cluster_mgrs

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	fleeterrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/google/uuid"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/golang/glog"

	"github.com/pkg/errors"
)

const (
	deprovisioningClustersWorkerType = "deprovisioning_clusters"
)

// DeprovisioningClustersManager represents a cluster manager that periodically reconciles data plane clusters in deprovisioning state.
type DeprovisioningClustersManager struct {
	workers.BaseWorker
	clusterService         services.ClusterService
	dataplaneClusterConfig *config.DataplaneClusterConfig
}

// NewDeprovisioningClustersManager creates a new cluster manager to reconcile data plane clusters in deprovisioning state.
func NewDeprovisioningClustersManager(reconciler workers.Reconciler, clusterService services.ClusterService, dataplaneClusterConfig *config.DataplaneClusterConfig) *DeprovisioningClustersManager {
	return &DeprovisioningClustersManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: deprovisioningClustersWorkerType,
			Reconciler: reconciler,
		},
		clusterService:         clusterService,
		dataplaneClusterConfig: dataplaneClusterConfig,
	}
}

// Start initializes the cluster manager to reconcile data plane clusters in deprovisioning state.
func (m *DeprovisioningClustersManager) Start() {
	m.StartWorker(m)
}

// Stop causes the process for reconciling data plane clusters in deprovisioning state to stop.
func (m *DeprovisioningClustersManager) Stop() {
	m.StopWorker(m)
}

func (m *DeprovisioningClustersManager) Reconcile() []error {
	glog.Infoln("reconciling clusters")

	var errList fleeterrors.ErrorList
	err := m.processDeprovisioningClusters()
	if err != nil {
		errList.AddErrors(err)
	}

	return errList.ToErrorSlice()
}

func (m *DeprovisioningClustersManager) processDeprovisioningClusters() error {
	var errList fleeterrors.ErrorList

	deprovisioningClusters, serviceErr := m.clusterService.ListByStatus(api.ClusterDeprovisioning)

	if serviceErr != nil {
		errList.AddErrors(serviceErr)
		return errList
	}

	glog.Infof("deprovisioning clusters count = %d", len(deprovisioningClusters))

	for i := range deprovisioningClusters {
		cluster := deprovisioningClusters[i]
		glog.V(10).Infof("deprovision cluster ClusterID = %s", cluster.ClusterID)
		metrics.UpdateClusterStatusSinceCreatedMetric(cluster, api.ClusterDeprovisioning)
		if err := m.reconcileDeprovisioningCluster(&cluster); err != nil {
			errList.AddErrors(errors.Wrapf(err, "failed to reconcile deprovisioning cluster %s", cluster.ClusterID))
		}
	}

	if errList.IsEmpty() {
		return nil
	}

	return errList
}

func (m *DeprovisioningClustersManager) reconcileDeprovisioningCluster(cluster *api.Cluster) error {
	if m.dataplaneClusterConfig.IsDataPlaneAutoScalingEnabled() {
		siblingCluster, findClusterErr := m.clusterService.FindCluster(services.FindClusterCriteria{
			Region:   cluster.Region,
			Provider: cluster.CloudProvider,
			MultiAZ:  cluster.MultiAZ,
			Status:   api.ClusterReady,
		})

		if findClusterErr != nil {
			return findClusterErr
		}

		//if it is the only cluster left in that region, set it back to ready.
		if siblingCluster == nil {
			return m.clusterService.UpdateStatus(*cluster, api.ClusterReady)
		}
	}

	deleted, deleteClusterErr := m.clusterService.Delete(cluster)
	if deleteClusterErr != nil {
		return deleteClusterErr
	}

	if !deleted {
		return nil
	}

	// cluster has been removed from cluster service. Mark it for cleanup.
	glog.Infof("Cluster %s  has been removed from cluster service.", cluster.ClusterID)
	updateStatusErr := m.clusterService.UpdateStatus(*cluster, api.ClusterCleanup)
	if updateStatusErr != nil {
		return errors.Wrapf(updateStatusErr, "Failed to update deprovisioning cluster %s status to 'cleanup'", cluster.ClusterID)
	}

	return nil
}
