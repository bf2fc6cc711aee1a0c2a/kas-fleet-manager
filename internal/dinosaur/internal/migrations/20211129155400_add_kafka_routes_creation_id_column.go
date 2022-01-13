package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addDinosaurRoutesCreationIdColumn() *gormigrate.Migration {
	type DinosaurRequest struct {
		RoutesCreationId string `json:"routes_creation_id" gorm:"default:''"`
	}

	return &gormigrate.Migration{
		ID: "20211129155400",
		Migrate: func(tx *gorm.DB) error {
			err := tx.AutoMigrate(&DinosaurRequest{})
			if err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			err := tx.Migrator().DropColumn(&DinosaurRequest{}, "routes_creation_id")
			if err != nil {
				return err
			}

			return nil
		},
	}
}
