package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// NOTE: This migration is postgres specific as GORM could not handle the migration of types to boolean.
func addKafkabootstrapServerHostType() *gormigrate.Migration {
	type KafkaRequest struct {
		BootstrapServerHost string
	}

	return &gormigrate.Migration{
		ID: "202009221550",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&KafkaRequest{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropColumn(&KafkaRequest{}, "bootstrap_server_host")
		},
	}

}
