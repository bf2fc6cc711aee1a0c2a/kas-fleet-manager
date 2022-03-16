package workers

import (
	"context"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/golang/glog"
	"github.com/google/uuid"
)

var _ workers.Worker = &NamespaceManager{}

type NamespaceManager struct {
	workers.BaseWorker
	namespaceService services.ConnectorNamespaceService
	db                      *db.ConnectionFactory
	ctx                     context.Context
}

func (m *NamespaceManager) Start() {
	m.StartWorker(m)
}

func (m *NamespaceManager) Stop() {
	m.StopWorker(m)
}

func NewNamespaceManager(bus signalbus.SignalBus, namespaceService services.ConnectorNamespaceService, db *db.ConnectionFactory) *NamespaceManager {
	return &NamespaceManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "connector_namespace",
			Reconciler: workers.Reconciler{
				SignalBus: bus,
			},
		},
		namespaceService: namespaceService,
		db: db,
	}
}

func (m *NamespaceManager) Reconcile() []error {

	if err := m.initContext(); err != nil {
		return []error{err}
	}

	var errs []error
	glog.V(5).Infof("Deleting expired namespaces...")
	namespaces, err := m.namespaceService.GetExpiredNamespaceIds()
	if err != nil {
		errs = append(errs, err)
	} else {
		n := len(namespaces)
		if n == 0 {
			glog.V(5).Infof("No expired namespaces")
		} else {

			glog.V(5).Infof("Deleting %d namespaces...", n)
			success := n
			if serr := InDBTransaction(m.ctx, func(ctx context.Context) error {
				for _, id := range namespaces {
					if err := m.namespaceService.Delete(ctx, id); err != nil {
						errs = append(errs, err)
						success--
					}
				}
				return nil
			}); serr != nil {
				errs = append(errs, serr)
			}
			glog.V(5).Infof("Deleted %d expired namespaces with %d errors", success, len(errs))
		}
	}

	// delete "deleting" namespaces with no connectors
	glog.V(5).Infof("Removing empty deleting namespaces...")
	count, serrs := m.namespaceService.ReconcileDeletingNamespaces()
	for _, serr := range serrs {
		errs = append(errs, serr)
	}
	if count == 0 {
		glog.V(5).Infof("No empty deleting namespaces")
	} else {
		glog.V(5).Infof("Removed %d empty deleting namespaces with %d errors", count, len(serrs))
	}

	return errs
}

func (m *NamespaceManager) initContext() error {
	if m.ctx == nil {
		ctx, err := m.db.NewContext(context.Background())
		if err != nil {
			return err
		}
		m.ctx = ctx
	}
	return nil
}
