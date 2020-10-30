package workers

import (
	"fmt"
	"github.com/golang/glog"
	"time"
)

const (
	repeatInterval = 30 * time.Second
)

type Reconciler struct {
}

func (r *Reconciler) Start(worker Worker) {
	*worker.GetStopChan() = make(chan struct{})
	worker.GetSyncGroup().Add(1)

	glog.V(1).Infoln(fmt.Sprintf("Starting %T", worker))
	// start reconcile immediately and then on every repeat interval
	worker.reconcile()
	ticker := time.NewTicker(repeatInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				glog.V(1).Infoln("Reconciling OCM clusters")
				worker.reconcile()
			case <-*worker.GetStopChan():
				ticker.Stop()
				defer worker.GetSyncGroup().Done()
				glog.V(1).Infoln("Stopping reconcile loop")
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
