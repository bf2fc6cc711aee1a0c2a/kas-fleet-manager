package workers

import (
	"context"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/golang/glog"
	"github.com/google/uuid"
)

var _ workers.Worker = &NamespaceManager{}

type NamespaceManager struct {
	workers.BaseWorker
	namespaceService services.ConnectorNamespaceService
	db               *db.ConnectionFactory
	ctx              context.Context
}

func (m *NamespaceManager) Start() {
	m.StartWorker(m)
}

func (m *NamespaceManager) Stop() {
	m.StopWorker(m)
}

func NewNamespaceManager(namespaceService services.ConnectorNamespaceService, db *db.ConnectionFactory,
	reconciler workers.Reconciler) *NamespaceManager {
	return &NamespaceManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "connector_namespace",
			Reconciler: reconciler,
		},
		namespaceService: namespaceService,
		db:               db,
	}
}

func (m *NamespaceManager) Reconcile() []error {

	var errs []error
	if m.ctx == nil {
		ctx, err := m.db.NewContext(context.Background())
		if err != nil {
			return []error{err}
		}
		m.ctx = ctx
	}

	// reconcile expired namespaces
	m.reconcile(&errs, "expired", m.namespaceService.ReconcileExpiredNamespaces)

	// reconcile unused "deleting" namespaces that never had connectors
	m.reconcile(&errs, "unused deleting", m.namespaceService.ReconcileUnusedDeletingNamespaces)

	// reconcile used "deleting" namespaces that have connectors
	m.reconcile(&errs, "used deleting", m.namespaceService.ReconcileUsedDeletingNamespaces)

	// delete "deleted" namespaces with no connectors
	m.reconcile(&errs, "empty deleted", m.namespaceService.ReconcileDeletedNamespaces)

	return errs
}

func (m *NamespaceManager) reconcile(errs *[]error, nsType string, reconcileFunc func(ctx context.Context) (int64, *errors.ServiceError)) {
	glog.V(5).Infof("Reconciling %s namespaces...", nsType)
	var count int64
	err := InDBTransaction(m.ctx, func(ctx context.Context) error {
		var serr *errors.ServiceError
		count, serr = reconcileFunc(ctx)
		if serr != nil {
			return serr
		}
		return nil
	})
	if err != nil {
		*errs = append(*errs, err)
	}
	if count == 0 {
		glog.V(5).Infof("No %s namespaces", nsType)
	} else {
		nerr := 0
		if err != nil {
			nerr++
		}
		glog.V(5).Infof("Processed %d %s namespaces with %d errors", count, nsType, nerr)
	}
}
