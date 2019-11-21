package db

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
)

func addDinosaurs() *gormigrate.Migration {
	type Dinosaur struct {
		Model
		Species string `gorm:"index"`
	}

	return &gormigrate.Migration{
		ID: "201911212019",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&Dinosaur{}).Error; err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.DropTable(&Dinosaur{}).Error; err != nil {
				return err
			}
			return nil
		},
	}
}
