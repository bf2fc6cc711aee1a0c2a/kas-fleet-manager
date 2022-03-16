package workers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/golang/glog"
	"github.com/google/uuid"
)

var _ workers.Worker = &ClusterManager{}

type ClusterManager struct {
	workers.BaseWorker
	clusterService services.ConnectorClusterService
}

func (m *ClusterManager) Start() {
	m.StartWorker(m)
}

func (m *ClusterManager) Stop() {
	m.StopWorker(m)
}

func NewClusterManager(bus signalbus.SignalBus, clusterService services.ConnectorClusterService) *ClusterManager {
	return &ClusterManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "connector_cluster",
			Reconciler: workers.Reconciler{
				SignalBus: bus,
			},
		},
		clusterService: clusterService,
	}
}

func (m *ClusterManager) Reconcile() []error {
	glog.V(5).Infof("Removing empty deleting clusters...")
	var errs []error
	count, serrs := m.clusterService.ReconcileDeletingClusters()

	if len(serrs) != 0 {
		for _, serr := range serrs {
			errs = append(errs, serr)
		}
	}
	glog.V(5).Infof("Removed %d empty deleting clusters", count)
	return errs
}
