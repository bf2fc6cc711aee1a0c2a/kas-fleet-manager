package cluster_mgrs

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/golang/glog"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/google/uuid"
)

const (
	DynamicScaleUpWorkerType = "dynamic_scale_up"
)

type DynamicScaleUpManager struct {
	workers.BaseWorker

	DataplaneClusterConfig *config.DataplaneClusterConfig
}

var _ workers.Worker = &DynamicScaleUpManager{}

func NewDynamicScaleUpManager(
	reconciler workers.Reconciler,
	dataplaneClusterConfig *config.DataplaneClusterConfig,
) *DynamicScaleUpManager {

	return &DynamicScaleUpManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: DynamicScaleUpWorkerType,
			Reconciler: reconciler,
		},
		DataplaneClusterConfig: dataplaneClusterConfig,
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

	return nil
}

func (m *DynamicScaleUpManager) logReconcileEventEnd() {
	glog.Infoln("dynamic scale up reconcile event finished")
}
