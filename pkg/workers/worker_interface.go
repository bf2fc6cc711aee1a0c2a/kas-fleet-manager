package workers

import "sync"

type Worker interface {
	Start()
	Stop()
	reconcile()
	GetStopChan() *chan struct{}
	GetSyncGroup() *sync.WaitGroup
}
