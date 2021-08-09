package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addKafkaNamespaceColumn() *gormigrate.Migration {
	type KafkaRequest struct {
		Namespace string `json:"namespace" gorm:"default:''"`
	}

	return &gormigrate.Migration{
		ID: "20210809135700",
		Migrate: func(tx *gorm.DB) error {
			err := tx.AutoMigrate(&KafkaRequest{})
			if err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			err := tx.Migrator().DropColumn(&KafkaRequest{}, "namespace")
			if err != nil {
				return err
			}

			return nil
		},
	}
}
