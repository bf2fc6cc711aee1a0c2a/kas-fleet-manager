package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addDinosaurQuotaTypeColumn() *gormigrate.Migration {
	type DinosaurRequest struct {
		QuotaType string
	}
	return &gormigrate.Migration{
		ID: "20210601170000",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&DinosaurRequest{}); err != nil {
				return err
			}
			return tx.Unscoped().Exec("UPDATE dinosaur_requests SET quota_type='allow-list';").Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropColumn(&DinosaurRequest{}, "quota_type")
		},
	}
}
