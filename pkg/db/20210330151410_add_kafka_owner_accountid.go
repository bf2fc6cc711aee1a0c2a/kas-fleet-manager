package db

import (
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
)

func addKafkaOwnerAccountId() *gormigrate.Migration {
	type KafkaRequest struct {
		OwnerAccountId string `json:"owner_account_id"`
	}
	return &gormigrate.Migration{
		ID: "20210330151410",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&KafkaRequest{}).Error; err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.Table("kafka_requests").DropColumn("owner_account_id").Error; err != nil {
				return err
			}
			return nil
		},
	}
}
