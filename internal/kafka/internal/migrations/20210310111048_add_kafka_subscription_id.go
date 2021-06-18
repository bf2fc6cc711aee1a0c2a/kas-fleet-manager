package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addKafkaSubscriptionId() *gormigrate.Migration {
	type KafkaRequest struct {
		SubscriptionId string
	}
	return &gormigrate.Migration{
		ID: "20210310111048",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&KafkaRequest{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropColumn(&KafkaRequest{}, "subscription_id")
		},
	}
}
