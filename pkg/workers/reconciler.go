package workers

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/getsentry/sentry-go"
	"sync"
	"time"

	"github.com/golang/glog"
)

var RepeatInterval time.Duration = 30 * time.Second

type Reconciler struct {
	wakeup chan *sync.WaitGroup
}

// Wakeup causes the worker reconcile to be performed as soon as possible.  If wait is true, the this
// function blocks until the reconcile is completed, otherwise this function does not block.
func (r *Reconciler) Wakeup(wait bool) {
	if wait {
		wg := &sync.WaitGroup{}
		wg.Add(1)
		r.wakeup <- wg
		wg.Wait()
	} else {
		select {
		case r.wakeup <- nil:
			// wakeup channel accepted the message
		default:
			// wakeup channel was full..
		}
	}
}

func (r *Reconciler) Start(worker Worker) {
	r.wakeup = make(chan *sync.WaitGroup, 1)
	*worker.GetStopChan() = make(chan struct{})
	worker.GetSyncGroup().Add(1)
	worker.SetIsRunning(true)

	bus := environments.Environment().Services.SignalBus
	sub := bus.Subscribe("reconcile:" + worker.GetWorkerType())

	glog.V(1).Infoln(fmt.Sprintf("Starting reconciliation loop for %T [%s]", worker, worker.GetID()))
	//starts reconcile immediately and then on every repeat interval
	r.runReconcile(worker)
	ticker := time.NewTicker(RepeatInterval)
	go func() {
		defer sub.Close()
		for {
			select {
			case wg := <-r.wakeup: //we were asked to wake up...
				glog.V(1).Infoln(fmt.Sprintf("Starting reconciliation loop for %T [%s]", worker, worker.GetID()))
				r.runReconcile(worker)
				if wg != nil {
					wg.Done()
				}
			case <-ticker.C: //time out
				glog.V(1).Infoln(fmt.Sprintf("Starting reconciliation loop for %T [%s]", worker, worker.GetID()))
				r.runReconcile(worker)
			case <-sub.Signal():
				glog.V(1).Infoln(fmt.Sprintf("Starting reconciliation loop for %T [%s]", worker, worker.GetID()))
				r.runReconcile(worker)
			case <-*worker.GetStopChan():
				ticker.Stop()
				defer worker.GetSyncGroup().Done()
				glog.V(1).Infoln(fmt.Sprintf("Stopping reconciliation loop for %T [%s]", worker, worker.GetID()))
				return
			}
		}
	}()
}

func (r *Reconciler) runReconcile(worker Worker) {
	start := time.Now()
	errors := worker.reconcile()
	if len(errors) == 0 {
		metrics.IncreaseReconcilerSuccessCount(worker.GetWorkerType())
	} else {
		metrics.IncreaseReconcilerFailureCount(worker.GetWorkerType())
		metrics.IncreaseReconcilerErrorsCount(worker.GetWorkerType(), len(errors))
	}
	metrics.UpdateReconcilerDurationMetric(worker.GetWorkerType(), time.Since(start))
	for _, e := range errors {
		// TODO: we can inspect the error and decide if they need to be send to sentry or not
		sentry.CaptureException(e)
	}
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
