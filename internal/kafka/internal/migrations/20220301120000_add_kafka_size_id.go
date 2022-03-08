package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addKafkaSizeId() *gormigrate.Migration {
	type KafkaRequest struct {
		SizeId string `json:"size_id" gorm:"default:x1"`
	}

	return &gormigrate.Migration{
		ID: "20220301120000",
		Migrate: func(tx *gorm.DB) error {
			err := tx.AutoMigrate(&KafkaRequest{})
			if err != nil {
				return err
			}

			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			err := tx.Migrator().DropColumn(&KafkaRequest{}, "size_id")
			if err != nil {
				return err
			}

			return nil
		},
	}
}
