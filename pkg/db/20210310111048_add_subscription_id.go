package db

import (
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
)

func addSubscriptionId() *gormigrate.Migration {
	type KafkaRequest struct {
		SubscriptionId string
	}
	return &gormigrate.Migration{
		ID: "20210310111048",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&KafkaRequest{}).Error; err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.Table("kafka_requests").DropColumn("subscription_id").Error; err != nil {
				return err
			}
			return nil
		},
	}
}
