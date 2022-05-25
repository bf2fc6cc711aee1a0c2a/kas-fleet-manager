package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

func addAdminApiServerURL() *gormigrate.Migration {
	type KafkaRequest struct {
		AdminApiServerURL string `json:"admin_api_server_url" gorm:"default:''"`
	}

	return &gormigrate.Migration{
		ID: "202205061532",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&KafkaRequest{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropColumn(&KafkaRequest{}, "admin_api_server_url")
		},
	}
}
