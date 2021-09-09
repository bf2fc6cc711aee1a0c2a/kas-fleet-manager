package signalbus

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/db"
	"github.com/golang/glog"
	"github.com/lib/pq"
)

var _ SignalBus = &PgSignalBus{} // type check the interface is implemented.

// PgSignalBus implements a signalbus.SignalBus that is clustered using postgresql notify events.
type PgSignalBus struct {
	isRunning         int32
	stopChan          chan struct{}
	syncGroup         sync.WaitGroup
	connectionFactory *db.ConnectionFactory
	signalBus         SignalBus // typically an in memory signal bus.
}

// NewSignalBusService creates a new PgSignalBus
func NewPgSignalBus(signalBus SignalBus, connectionFactory *db.ConnectionFactory) *PgSignalBus {
	return &PgSignalBus{
		connectionFactory: connectionFactory,
		signalBus:         signalBus,
		stopChan:          make(chan struct{}),
	}
}

// Notify will notify all the subscriptions created across the cluster of the given named signal.
func (sbw *PgSignalBus) Notify(name string) {
	// instead of send the Notify to the in memory bus, first send it to the DB
	// with the pg_notify function.  The DB will send it back to us and all other processes
	// that are listening for those events.
	dbc := sbw.connectionFactory.New()
	if err := dbc.Exec("SELECT pg_notify('signalbus', ?)", name).Error; err != nil {
		glog.V(1).Info("notify failed:", err.Error())
	}
}

// Subscribe creates a subscription the named signal.
// They are performed on the in memory bus.
func (sbw *PgSignalBus) Subscribe(name string) *Subscription {
	return sbw.signalBus.Subscribe(name)
}

// Start starts the background worker that listens for the
// events that are sent from this process and all other processes publishing
// to the signalbus channel.
func (sbw *PgSignalBus) Start() {
	// protect against being called twice...
	if atomic.CompareAndSwapInt32(&sbw.isRunning, 0, 1) {
		sbw.stopChan = make(chan struct{})
		sbw.syncGroup.Add(1)

		go func() {
			defer sbw.syncGroup.Done()
			for {

				// Exit out of the loop if the worker is stopped
				select {
				case <-sbw.stopChan:
					return
				default:
				}

				exit := sbw.run()
				if exit {
					return
				}

				// sleep a little between runs, to avoid failure loops.
				time.Sleep(10 * time.Second)
			}
		}()
	}
}

// Stop causes the worker to stop.  Blocks until all background go routines complete.
func (sbw *PgSignalBus) Stop() {
	select {
	case <-sbw.stopChan:
		//already closed
	default:
		close(sbw.stopChan)  //explicit close
		sbw.syncGroup.Wait() //wait for in-flight job to finish
	}
	atomic.StoreInt32(&sbw.isRunning, 0)
}

func (sbw *PgSignalBus) run() (exit bool) {

	// use the posgresql db driver specific APIs to listen for events from the DB connection.
	listener := pq.NewListener(sbw.connectionFactory.Config.ConnectionString(), 10*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			glog.V(1).Info("pq listener error", err.Error())
		}
	})
	defer listener.Close() // clean up connections on return..

	// Listen on the "signalbus" channel.
	err := listener.Listen("signalbus")
	if err != nil {
		glog.V(1).Info("error listening to events:", err.Error())
		return false
	}
	for {
		// Now lets pull events sent to the listener
		exit, err := sbw.waitForNotification(listener)
		if err != nil {
			glog.V(1).Info("error waiting for event:", err.Error())
			return false
		}
		if exit {
			return true
		}
	}
}

func (sbw *PgSignalBus) waitForNotification(l *pq.Listener) (exit bool, err error) {
	for {
		select {
		case <-sbw.stopChan:
			// this occurs triggered when PgSignalBus.Stop() is called... let the caller we should exit..
			return true, nil
		case n := <-l.Notify:
			if n == nil {
				// notify channel was closed.. likely due to db connection failure.
				return false, fmt.Errorf("postgres listner channel closed")
			}
			glog.V(1).Infof("Received data from channel: %s, data: %s", n.Channel, n.Extra)

			// we got the signal name from the DB... lets use the in memory signalBus
			// to notify all the subscribers that registered for events.
			sbw.signalBus.Notify(n.Extra)
			return
		case <-time.After(90 * time.Second):
			// in case we have not received an event in a while... lets check to make sure the DB
			// connection is still good... if not exit with error so we can retry...
			glog.V(1).Info("Received no events for 90 seconds, checking connection")
			err = l.Ping()
			if err != nil {
				return false, err
			}
		}
	}
}
