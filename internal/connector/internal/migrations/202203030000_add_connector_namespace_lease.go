package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
	"time"
)

func addConnectorNamespaceLease(migrationId string) *gormigrate.Migration {

	type LeaderLease struct {
		db.Model
		Leader    string
		LeaseType string
		Expires   *time.Time
	}

	// add missing deleted_at index caused by incorrectly using api.Meta instead of db.Model in dbapi.* structs
	return db.CreateMigrationFromActions(migrationId,
		db.FuncAction(func(tx *gorm.DB) error {
			// We don't want to delete the leader lease table on rollback because it's shared with the kas-fleet-manager
			// so we just create it here if it does not exist yet.. but we don't drop it on rollback.
			err := tx.Migrator().AutoMigrate(&LeaderLease{})
			if err != nil {
				return err
			}
			now := time.Now().Add(-time.Minute) //set to a expired time
			return tx.Create(&api.LeaderLease{
				Expires:   &now,
				LeaseType: "connector_namespace",
			}).Error
		}, func(tx *gorm.DB) error {
			// The leader lease table may have already been dropped, by the kafka migration rollback, ignore error
			_ = tx.Where("lease_type = ?", "connector_namespace").Delete(&api.LeaderLease{})
			return nil
		}),
	)
}
