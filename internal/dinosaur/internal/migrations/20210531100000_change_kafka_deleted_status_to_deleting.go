package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func changeDinosaurDeleteStatusToDeleting() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20210531100000",
		Migrate: func(tx *gorm.DB) error {
			return tx.Unscoped().Exec("UPDATE dinosaur_requests SET status='deleting' WHERE status='deleted';").Error
		},
		Rollback: func(tx *gorm.DB) error {
			// do nothing as this is oneshot operation
			return nil
		},
	}
}
