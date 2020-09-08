package db

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
)

func addKafka() *gormigrate.Migration {
	type Kafka struct {
		Model
		Region        string
		ClusterID     string
		CloudProvider string
		Name          string `gorm:"index"`
		Status        string
		Owner         string
	}

	return &gormigrate.Migration{
		ID: "202009040909",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&Kafka{}).Error; err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.DropTable(&Kafka{}).Error; err != nil {
				return err
			}
			return nil
		},
	}
}
