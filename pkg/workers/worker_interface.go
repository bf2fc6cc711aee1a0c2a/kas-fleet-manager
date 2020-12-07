package workers

import "sync"

type Worker interface {
	GetID() string
	GetWorkerType() string
	Start()
	Stop()
	reconcile()
	GetStopChan() *chan struct{}
	GetSyncGroup() *sync.WaitGroup
	IsRunning() bool
	SetIsRunning(val bool)
}
