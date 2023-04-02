package sync

import (
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// DistributedLock manages locks on the database through the usage of the `distributed_lock` table.
// WARNING: a lock can only be released by the instance that initially created it.
type DistributedLock interface {
	Lock() error
	Unlock() error
	isLocked() bool
}

var _ DistributedLock = &distributedLock{}

// NewDistributedLock returns a new instance of DistributedLock
// db - the connection to the database
// lockID - the ID of the lock. When multiple threads request to use the Lock function with the same lock ID, only the first thread is granted access while the remaining threads must wait until the lock is released before proceeding.
func NewDistributedLock(db *gorm.DB, lockID string) DistributedLock {
	return &distributedLock{db: db, id: lockID}
}

type distributedLock struct {
	db     *gorm.DB
	tx     *gorm.DB
	id     string
	locked bool
}

// Lock locks dl. If the lock is already in use, the caller blocks until the lock is available.
func (dl *distributedLock) Lock() error {
	dl.tx = dl.db.Begin()
	result := dl.tx.Exec("INSERT INTO distributed_locks (ID) VALUES (?) ON CONFLICT DO NOTHING", dl.id)
	if result.Error != nil {
		dl.tx.Rollback() // close the transaction
		dl.tx = nil
		return errors.Wrapf(result.Error, "unable to acquire lock: %v", result.Error)
	}
	dl.locked = true
	return nil
}

// Unlock unlock dl.
func (dl *distributedLock) Unlock() error {
	if dl.tx == nil {
		return errors.Errorf("lock not acquired")
	}

	result := dl.tx.Rollback()
	if result.Error != nil {
		return errors.Wrapf(result.Error, "unable to release lock: %v", result.Error)
	}
	dl.locked = false
	return nil
}

func (dl *distributedLock) isLocked() bool {
	return dl.locked
}
