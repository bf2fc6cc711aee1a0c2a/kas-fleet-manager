package workers

import "sync"

//go:generate moq -out woker_interface_moq.go . Worker
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
