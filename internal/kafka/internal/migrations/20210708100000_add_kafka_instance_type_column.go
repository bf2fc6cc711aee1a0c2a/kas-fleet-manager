package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addKafkaInstanceTypeColumn() *gormigrate.Migration {
	type KafkaRequest struct {
		InstanceType string
	}
	return &gormigrate.Migration{
		ID: "20210708100000",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&KafkaRequest{}); err != nil {
				return err
			}
			return tx.Unscoped().Exec("UPDATE kafka_requests SET instance_type='standard' where (instance_type is null OR instance_type='');").Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropColumn(&KafkaRequest{}, "instance_type")
		},
	}
}
