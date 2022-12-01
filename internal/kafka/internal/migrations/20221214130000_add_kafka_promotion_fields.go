package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addKafkaPromotionFields() *gormigrate.Migration {
	type KafkaPromotionStatus string

	type KafkaRequest struct {
		PromotionStatus  KafkaPromotionStatus `json:"promotion_status"`
		PromotionDetails string               `json:"promotion_details"`
	}

	return &gormigrate.Migration{
		ID: "20221214130000",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&KafkaRequest{})
		},
		Rollback: func(tx *gorm.DB) error {
			err := tx.Migrator().DropColumn(&KafkaRequest{}, "promotion_status")
			if err != nil {
				return err
			}

			err = tx.Migrator().DropColumn(&KafkaRequest{}, "promotion_details")
			if err != nil {
				return err
			}

			return nil
		},
	}
}
