package workers

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/jinzhu/gorm"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/db"
	"time"
)

const (
	mgrRepeatInterval = 30 * time.Second
	leaseRenewTime    = 3 * time.Minute
)

type LeaderElectionManager struct {
	workers           []Worker
	connectionFactory *db.ConnectionFactory
	tearDown          chan struct{}
	mgrRepeatInterval time.Duration
	leaseRenewTime    time.Duration
}

// leaderLeaseAcquisition a wrapper for a lease and whether it's been acquired/is owned by another worker
type leaderLeaseAcquisition struct {
	// acquired whether the worker successfully acquired/retained the leader lease
	acquired bool
	// currentLease the current lease, it may not necessarily belong to the worker, see acquired
	currentLease *api.LeaderLease
}

func NewLeaderLeaseManager(workers []Worker, connectionFactory *db.ConnectionFactory) *LeaderElectionManager {
	return &LeaderElectionManager{
		workers:           workers,
		connectionFactory: connectionFactory,
		mgrRepeatInterval: mgrRepeatInterval,
		leaseRenewTime:    leaseRenewTime,
	}
}

func (s *LeaderElectionManager) Start() {
	s.tearDown = make(chan struct{})
	glog.V(1).Infoln("Starting LeaderElectionManager")
	// Starts once immediately
	s.startWorkers()
	ticker := time.NewTicker(s.mgrRepeatInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				s.startWorkers()
			case <-s.tearDown:
				for _, worker := range s.workers {
					if worker.IsRunning() {
						worker.Stop()
					}
				}
				return
			}
		}
	}()
}

// impl. Stoppable
func (s *LeaderElectionManager) Stop() {
	select {
	case <-s.tearDown:
		return
	default:
		close(s.tearDown)
	}
}

func (s *LeaderElectionManager) startWorkers() {
	for _, worker := range s.workers {
		isLeader := s.isWorkerLeader(worker)
		if isLeader && !worker.IsRunning() {
			glog.V(1).Infoln(fmt.Sprintf("Running as the leader and starting worker %T [%s]", worker, worker.GetID()))
			worker.Start()
		} else if !isLeader && worker.IsRunning() {
			glog.V(1).Infoln(fmt.Sprintf("No longer the leader and stopping worker %T [%s]", worker, worker.GetID()))
			worker.Stop()
		}
	}
}

func (s *LeaderElectionManager) isWorkerLeader(worker Worker) bool {
	dbConn := s.connectionFactory.New()
	leaderLeaseAcquisition, err := s.acquireLeaderLease(worker.GetID(), worker.GetWorkerType(), dbConn)
	if err != nil {
		// we don't know whether we're the leader or not, set metric to false for now
		//metrics.UpdateLeaderStatusMetric(false)
		glog.V(5).Infof("failed to acquire leader lease: %s", err)
		return false
	}

	if !leaderLeaseAcquisition.acquired {
		glog.V(5).Infof("not currently leader, skipping reconcile %T [%s]", worker, worker.GetID())
		return false
	}

	return true
}

// acquireLeaderLease attempt to claim the leader role using a provided table and return a leaderLeaseAcquisition
// containing the lease
func (s *LeaderElectionManager) acquireLeaderLease(workerId string, workerType string, dbConn *gorm.DB) (*leaderLeaseAcquisition, error) {
	// read the leader lease, to see whether the worker has an opportunity to:
	// - acquire the lease
	// - extend their lease, if they are the leader, if they are the leader and the lease expiry time is close
	// - continue with their existing lease, if they are the leader and the lease expiry time is far away
	var leaseList api.LeaderLeaseList
	if err := dbConn.Raw("SELECT * FROM leader_leases where deleted_at is null and lease_type = ?  LIMIT 1", workerType).Scan(&leaseList).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve leader leases: %w", err)
	}

	// we failed to read the current lease, we always expect a single lease to exist, create one so that worker can proceed.
	if len(leaseList) == 0 {
		return nil, fmt.Errorf("expected to find a lease entry, found none for :%s", workerType)
	}

	// the lease will be the first entry returned
	lease := leaseList[0]
	// if we get the opportunity to acquire or extend the lease, use this expiry time
	newExpiryTime := time.Now().Add(s.leaseRenewTime)
	// assume we're not the leader by default
	isLeader := false

	// determine if we have an opportunity to acquire or extend the lease (extend if the lease is going to expire in one min)
	if isExpired(lease) || (lease.Leader == workerId && lease.Expires.Before(time.Now().Add(time.Minute))) {
		// begin a new transaction
		// we must ensure we commit or rollback this transaction to avoid stale transactions being left around
		leaderTx := dbConn.Begin() //starts a new transaction

		// attempt to lock the leader lease
		if err := leaderTx.Raw("SELECT * FROM leader_leases where deleted_at is null and lease_type = ? FOR UPDATE SKIP LOCKED LIMIT 1", workerType).Scan(&leaseList).Error; err != nil {
			leaderTx.Rollback()
			return nil, fmt.Errorf("failed to retrieve leader leases: %w", err)
		}

		/* if length is 0 we missed our opportunity, another worker has taken the lease for update:
		- if we've passed the lease expiry, it could be another worker on this node or another, so fail
		  gracefully
		- if we haven't passed the lease expiry, it's another worker on this node extending the lease, so
		  continue the reconcile
		*/
		if len(leaseList) == 0 && isExpired(lease) {
			glog.V(1).Infof("failed to acquire lock on leader lease for update, skipping")
			leaderTx.Rollback()
			return &leaderLeaseAcquisition{
				acquired:     false,
				currentLease: lease,
			}, nil
		}

		// if length is non-zero then we have claimed a lock on the lease entry
		// we have the opportunity to extend or acquire the leader lease
		if len(leaseList) > 0 {
			if err := leaderTx.Model(&lease).Updates(map[string]interface{}{"leader": workerId, "expires": newExpiryTime}).Error; err != nil {
				leaderTx.Rollback()
				return nil, fmt.Errorf("failed to update leader lease: %w", err)
			}
		}

		// if we got to this point we either:
		// - have acquired an expired lease
		// - extended our existing lease
		// - are in our existing lease that doesn't need to be extended
		//
		// we are the leader at this point
		isLeader = true

		// ensure we persist our update by committing the transaction
		leaderTx.Commit()
	} else if lease.Leader == workerId {
		// we didn't have the opportunity to acquire or extend the lease, but we are marked as the leader on the
		// existing, unexpired lease, so continue assuming we're leader
		isLeader = true
	}

	return &leaderLeaseAcquisition{
		acquired:     isLeader,
		currentLease: lease,
	}, nil
}

func isExpired(lease *api.LeaderLease) bool {
	return time.Now().After(*lease.Expires)
}
