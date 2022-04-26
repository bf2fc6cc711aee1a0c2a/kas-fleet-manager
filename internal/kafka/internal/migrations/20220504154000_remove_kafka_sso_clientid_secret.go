package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func dropKafkaSsoClientIdAndSecret() *gormigrate.Migration {
	type KafkaRequest struct {
		SsoClientID     string `json:"sso_client_id"`
		SsoClientSecret string `json:"sso_client_secret"`
	}
	return &gormigrate.Migration{
		ID: "20220504154000",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.Migrator().DropColumn(&KafkaRequest{}, "sso_client_id"); err != nil {
				return err
			}
			if err := tx.Migrator().DropColumn(&KafkaRequest{}, "sso_client_secret"); err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&KafkaRequest{})
		},
	}
}
