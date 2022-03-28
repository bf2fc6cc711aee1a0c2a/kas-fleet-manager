package workers

import (
	"sync"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
)

//go:generate moq -out worker_interface_moq.go . Worker
type Worker interface {
	GetID() string
	GetWorkerType() string
	Start()
	Stop()
	Reconcile() []error
	GetStopChan() *chan struct{}
	GetSyncGroup() *sync.WaitGroup
	IsRunning() bool
	SetIsRunning(val bool)
}

type BaseWorker struct {
	Id           string
	WorkerType   string
	Reconciler   Reconciler
	isRunning    bool
	imStop       chan struct{}
	syncTeardown sync.WaitGroup
}

func (b *BaseWorker) GetID() string {
	return b.Id
}

func (b *BaseWorker) GetWorkerType() string {
	return b.WorkerType
}

func (b *BaseWorker) GetStopChan() *chan struct{} {
	return &b.imStop
}

func (b *BaseWorker) GetSyncGroup() *sync.WaitGroup {
	return &b.syncTeardown
}

func (b *BaseWorker) IsRunning() bool {
	return b.isRunning
}

func (b *BaseWorker) SetIsRunning(val bool) {
	b.isRunning = val
}

func (b *BaseWorker) StartWorker(w Worker) {
	metrics.SetLeaderWorkerMetric(b.WorkerType, true)
	b.Reconciler.Start(w)
}

func (b *BaseWorker) StopWorker(w Worker) {
	b.Reconciler.Stop(w)
	metrics.ResetMetricsForKafkaManagers()
	metrics.SetLeaderWorkerMetric(b.WorkerType, false)
}
