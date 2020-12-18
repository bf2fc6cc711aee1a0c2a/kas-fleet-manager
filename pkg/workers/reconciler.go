package workers

import (
	"fmt"
	"time"

	"github.com/golang/glog"
)

const (
	RepeatInterval = 30 * time.Second
)

type Reconciler struct {
}

func (r *Reconciler) Start(worker Worker) {
	*worker.GetStopChan() = make(chan struct{})
	worker.GetSyncGroup().Add(1)
	worker.SetIsRunning(true)

	glog.V(1).Infoln(fmt.Sprintf("Starting reconciliation loop for %T [%s]", worker, worker.GetID()))
	//starts reconcile immediately and then on every repeat interval
	worker.reconcile()
	ticker := time.NewTicker(RepeatInterval)
	go func() {
		for {
			select {
			case <-ticker.C: //time out
				glog.V(1).Infoln(fmt.Sprintf("Starting reconciliation loop for %T [%s]", worker, worker.GetID()))
				worker.reconcile()
			case <-*worker.GetStopChan():
				ticker.Stop()
				defer worker.GetSyncGroup().Done()
				glog.V(1).Infoln(fmt.Sprintf("Stopping reconciliation loop for %T [%s]", worker, worker.GetID()))
				return
			}
		}
	}()
}

func (r *Reconciler) Stop(worker Worker) {
	defer worker.SetIsRunning(false)
	select {
	case <-*worker.GetStopChan(): //already closed
		return
	default:
		close(*worker.GetStopChan()) //explicit close
		worker.GetSyncGroup().Wait() //wait for in-flight job to finish
	}
}
