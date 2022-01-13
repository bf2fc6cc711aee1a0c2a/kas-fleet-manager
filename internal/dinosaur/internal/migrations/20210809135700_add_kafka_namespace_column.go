package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addDinosaurNamespaceColumn() *gormigrate.Migration {
	type DinosaurRequest struct {
		Namespace string `json:"namespace" gorm:"default:''"`
	}

	return &gormigrate.Migration{
		ID: "20210809135700",
		Migrate: func(tx *gorm.DB) error {
			err := tx.AutoMigrate(&DinosaurRequest{})
			if err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			err := tx.Migrator().DropColumn(&DinosaurRequest{}, "namespace")
			if err != nil {
				return err
			}

			return nil
		},
	}
}
