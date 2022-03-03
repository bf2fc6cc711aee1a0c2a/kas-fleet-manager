package workers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/golang/glog"
	"github.com/google/uuid"
)

var _ workers.Worker = &NamespaceManager{}

type NamespaceManager struct {
	workers.BaseWorker
	namespaceService services.ConnectorNamespaceService
}

func (m *NamespaceManager) Start() {
	m.StartWorker(m)
}

func (m *NamespaceManager) Stop() {
	m.StopWorker(m)
}

func NewNamespaceManager(bus signalbus.SignalBus, namespaceService services.ConnectorNamespaceService) *NamespaceManager {
	return &NamespaceManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "connector_namespace",
			Reconciler: workers.Reconciler{
				SignalBus: bus,
			},
		},
		namespaceService: namespaceService,
	}
}

func (m *NamespaceManager) Reconcile() []error {
	glog.V(5).Infof("Deleting expired namespaces...")
	namespaces, err := m.namespaceService.GetExpiredNamespaces()
	if err != nil {
		return []error{err}
	}

	n := len(namespaces)
	if n == 0 {
		glog.V(5).Infof("No expired namespaces")
		return nil
	}

	glog.V(5).Infof("Deleting %d namespaces...", n)
	success := n
	var errs []error
	for _, namespace := range namespaces {
		id := namespace.ID
		if err := m.namespaceService.Delete(id); err != nil {
			errs = append(errs, errors.GeneralError("Error deleting namespace %s: %s", id, err))
			success--
		}
	}

	glog.V(5).Infof("Deleted %d expired namespaces with %d errors", success, len(errs))
	return errs
}
