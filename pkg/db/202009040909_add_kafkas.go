package db

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addKafkaRequest() *gormigrate.Migration {
	type KafkaRequest struct {
		Model
		Region        string
		ClusterID     string
		CloudProvider string
		MultiAZ       string
		Name          string `gorm:"index"`
		Status        string
		Owner         string
	}

	return &gormigrate.Migration{
		ID: "202009040909",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&KafkaRequest{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&KafkaRequest{})
		},
	}
}
