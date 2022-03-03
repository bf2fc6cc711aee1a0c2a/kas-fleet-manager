package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addKafkaProfileIdSizeId() *gormigrate.Migration {
	type KafkaRequest struct {
		ProfileId string `json:"profile_id"`
		SizeId    string `json:"size_id" gorm:"default:x1"`
	}

	return &gormigrate.Migration{
		ID: "20220301120000",
		Migrate: func(tx *gorm.DB) error {
			err := tx.AutoMigrate(&KafkaRequest{})
			if err != nil {
				return err
			}

			if err := tx.Exec(`UPDATE kafka_requests SET profile_id = 'eval' WHERE profile_id IS NULL AND instance_type='eval'`).Error; err != nil {
				return err
			}

			if err := tx.Exec(`UPDATE kafka_requests SET profile_id = 'standard' WHERE profile_id IS NULL AND instance_type='standard'`).Error; err != nil {
				return err
			}

			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			err := tx.Migrator().DropColumn(&KafkaRequest{}, "profile_id")
			if err != nil {
				return err
			}

			err = tx.Migrator().DropColumn(&KafkaRequest{}, "size_id")
			if err != nil {
				return err
			}

			return nil
		},
	}
}
