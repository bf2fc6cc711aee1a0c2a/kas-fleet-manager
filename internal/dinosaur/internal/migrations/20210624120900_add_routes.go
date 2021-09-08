package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addRoutes() *gormigrate.Migration {
	type DinosaurRequest struct {
		Routes        string `gorm:"type:jsonb"`
		RoutesCreated bool   `gorm:"default:false"`
	}

	return &gormigrate.Migration{
		ID: "20210624120900",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&DinosaurRequest{}); err != nil {
				return err
			}
			return tx.Unscoped().Exec("UPDATE dinosaur_requests SET routes_created=true WHERE status='ready';").Error
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.Migrator().DropColumn(&DinosaurRequest{}, "routes"); err != nil {
				return err
			}
			return tx.Migrator().DropColumn(&DinosaurRequest{}, "routes_created")
		},
	}
}
