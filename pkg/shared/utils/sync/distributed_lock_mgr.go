package sync

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
	"gorm.io/gorm"
	"sync"
)

var mgrLock sync.Mutex

var _ DistributedLockMgr = &distributedLockMgr{}

func NewDistributedLockMgr(db *gorm.DB) DistributedLockMgr {
	return &distributedLockMgr{
		db:           db,
		locks:        map[string][]DistributedLock{},
		createLockFn: NewDistributedLock,
	}
}

// DistributedLockMgr manages a list of DistributedLock objects. The DistributedLock object is automatically removed
// from the list when the Unlock method is called.
type DistributedLockMgr interface {
	// The Lock function is designed to activate the lock identified by the given ID.
	// If the lock is already activated, this method will block until the lock is available
	Lock(lockID string) error
	// The Unlock function is used to release the lock associated with the given ID. Once the lock is released,
	// it will be removed from the list of managed locks.
	Unlock(lockID string) error
}

type distributedLockMgr struct {
	locks map[string][]DistributedLock
	db    *gorm.DB
	// createLockFn is the function that will create DistributedLock instances. Its main usage is to simplify testing by returning mock DistributedLock instances.
	createLockFn func(*gorm.DB, string) DistributedLock
}

func (dlm *distributedLockMgr) Lock(lockID string) error {
	mgrLock.Lock()

	lock := dlm.createLockFn(dlm.db, lockID)
	dlm.locks[lockID] = append(dlm.locks[lockID], lock)
	mgrLock.Unlock()
	err := lock.Lock()
	return err
}

func (dlm *distributedLockMgr) Unlock(lockID string) error {
	mgrLock.Lock()
	defer mgrLock.Unlock()

	locks := dlm.locks[lockID]

	idx, lock := arrays.FindFirst(locks, func(x DistributedLock) bool { return x.isLocked() })

	if idx == -1 {
		return fmt.Errorf("lock %q does not exists or is not locked in this manager", lockID)
	}

	err := lock.Unlock()

	if err != nil {
		return err
	}
	dlm.locks[lockID] = append(dlm.locks[lockID][:idx], dlm.locks[lockID][idx+1:]...)
	return nil
}
