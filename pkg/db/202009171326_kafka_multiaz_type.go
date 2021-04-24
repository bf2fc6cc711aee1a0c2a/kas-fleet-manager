package db

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// NOTE: This migration is postgres specific as GORM could not handle the migration of types to boolean.
func updateKafkaMultiAZTypeToBoolean() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202009171326",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.Exec("ALTER TABLE kafka_requests ALTER COLUMN multi_az DROP DEFAULT").Error; err != nil {
				return err
			}
			if err := tx.Exec("ALTER TABLE kafka_requests ALTER COLUMN multi_az TYPE bool USING CASE WHEN multi_az='true' THEN TRUE ELSE FALSE END").Error; err != nil {
				return err
			}
			if err := tx.Exec("ALTER TABLE kafka_requests ALTER COLUMN multi_az SET DEFAULT FALSE").Error; err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.Exec("ALTER TABLE kafka_requests ALTER COLUMN multi_az DROP DEFAULT").Error; err != nil {
				return err
			}
			if err := tx.Exec("ALTER TABLE kafka_requests ALTER COLUMN multi_az TYPE text").Error; err != nil {
				return err
			}
			return nil
		},
	}

}
