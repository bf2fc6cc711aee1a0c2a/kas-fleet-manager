package workers

import (
	"context"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/golang/glog"
	"github.com/google/uuid"
)

var _ workers.Worker = &ClusterManager{}

type ClusterManager struct {
	workers.BaseWorker
	clusterService services.ConnectorClusterService
	db             *db.ConnectionFactory
	ctx            context.Context
}

func (m *ClusterManager) Start() {
	m.StartWorker(m)
}

func (m *ClusterManager) Stop() {
	m.StopWorker(m)
}

func NewClusterManager(clusterService services.ConnectorClusterService, db *db.ConnectionFactory, reconciler workers.Reconciler) *ClusterManager {
	return &ClusterManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "connector_cluster",
			Reconciler: reconciler,
		},
		clusterService: clusterService,
		db:             db,
	}
}

func (m *ClusterManager) Reconcile() []error {

	var errs []error
	if m.ctx == nil {
		ctx, err := m.db.NewContext(context.Background())
		if err != nil {
			return []error{err}
		}
		m.ctx = ctx
	}

	// reconcile empty deleting clusters
	m.doReconcile(&errs, "empty deleting", m.clusterService.ReconcileEmptyDeletingClusters,
		"connector_clusters.status_phase = ? AND "+
			"connector_clusters.deleted_at IS NULL AND cluster_id IS NULL", dbapi.ConnectorClusterPhaseDeleting)

	// reconcile non-empty deleting clusters, marking their non-deleting namespaces for deletion
	m.doReconcile(&errs, "non-empty deleting", m.clusterService.ReconcileNonEmptyDeletingClusters,
		"connector_clusters.status_phase = ? AND "+
			"connector_clusters.deleted_at IS NULL AND cluster_id IS NOT NULL", dbapi.ConnectorClusterPhaseDeleting)

	return errs
}

func (m *ClusterManager) doReconcile(errs *[]error, kind string,
	reconcileFunc func(context.Context, []string) (int, []*errors.ServiceError),
	query string, args ...interface{}) {

	glog.V(5).Infof("Reconciling %s clusters...", kind)

	clusterIds, err := m.clusterService.GetClusterIds(query, args)
	if err != nil {
		glog.Errorf("Error retrieving %s clusters: %s", kind, err)
		*errs = append(*errs, err)
	}
	if len(clusterIds) == 0 {
		glog.V(5).Infof("No %s clusters", kind)
		return
	}

	if derr := InDBTransaction(m.ctx, func(ctx context.Context) error {
		count, serrs := reconcileFunc(ctx, clusterIds)

		for _, serr := range serrs {
			*errs = append(*errs, serr)
		}
		if count == 0 {
			glog.V(5).Infof("No %s clusters", kind)
		} else {
			glog.V(5).Infof("Reconciled %d %s clusters with %d errors", count, kind, len(serrs))
		}

		if len(serrs) != 0 {
			return errors.GeneralError("Error reconciling %s clusters: %s", kind, serrs)
		}
		return nil
	}); derr != nil {
		glog.Errorf("Error reconciling %s clusters: %s", kind, err)
	}
}
