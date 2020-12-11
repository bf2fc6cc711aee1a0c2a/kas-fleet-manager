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

	glog.V(1).Infoln(fmt.Sprintf("Starting %T", worker))
	// start reconcile immediately and then on every repeat interval
	worker.reconcile()
	ticker := time.NewTicker(RepeatInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				glog.V(1).Infoln(fmt.Sprintf("Starting %T", worker))
				worker.reconcile()
			case <-*worker.GetStopChan():
				ticker.Stop()
				defer worker.GetSyncGroup().Done()
				glog.V(1).Infoln(fmt.Sprintf("Stopping reconcile loop for %T", worker))
				return
			}
		}
	}()
}

func (r *Reconciler) Stop(worker Worker) {
	select {
	case <-*worker.GetStopChan():
		return
	default:
		close(*worker.GetStopChan())
		worker.GetSyncGroup().Wait()
	}
}
