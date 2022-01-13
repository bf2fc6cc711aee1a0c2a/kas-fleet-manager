package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// This make sure that a dinosaur created during the deployment of the "20210809140000_migrate_old_dinosaur_namespace.go" with id "20210809140000"
// are also properly migrated to avoid empty namespace
func migrateOldDinosaurNamespaceCreatedDuringDeployment() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20210810100000",
		Migrate: func(tx *gorm.DB) error {
			migrationScript := migrateOldDinosaurNamespace()
			return migrationScript.Migrate(tx)
		},
		Rollback: func(tx *gorm.DB) error {
			return nil
		},
	}
}
