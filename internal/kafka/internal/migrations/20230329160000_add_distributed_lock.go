package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addDistributedLockTable() *gormigrate.Migration {
	type DistributedLock struct {
		ID string
	}

	return &gormigrate.Migration{
		ID: "20230329160000",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&DistributedLock{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&DistributedLock{})
		},
	}
}
