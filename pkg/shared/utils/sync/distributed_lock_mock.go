package sync

import (
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

var _ DistributedLock = &distributedLockMock{}

// It appears that go-mocket does not offer a straightforward method to intercept the Rollback call. To make it easily
// interceptable, this mock substitutes the rollback call in the Unlock method with a DELETE.

func newDistributedLockMock(db *gorm.DB, lockID string) DistributedLock {
	dl := NewDistributedLock(db, lockID)
	return &distributedLockMock{dl: dl, db: db, id: lockID}
}

// This object mocks the `DistributedMock` object and is to be used with a mocked database
type distributedLockMock struct {
	dl DistributedLock
	db *gorm.DB
	id string
}

func (dl *distributedLockMock) isLocked() bool {
	return dl.dl.isLocked()
}

func (dl *distributedLockMock) Lock() error {
	return dl.dl.Lock()
}

func (dl *distributedLockMock) Unlock() error {
	_ = dl.dl.Unlock()
	result := dl.db.Exec("DELETE FROM distributed_locks where ID = ?", dl.id)
	if result.Error != nil {
		return errors.Wrapf(result.Error, "unable to acquire lock: %v", result.Error)
	}
	return nil
}
