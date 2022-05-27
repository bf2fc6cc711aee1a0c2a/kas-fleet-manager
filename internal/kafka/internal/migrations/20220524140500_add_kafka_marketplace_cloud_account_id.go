package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addKafkaCloudAccountIdMarketplaceFields() *gormigrate.Migration {
	type KafkaRequest struct {
		BillingCloudAccountId string `json:"billing_cloud_account_id"`
		Marketplace           string `json:"marketplace"`
	}

	return &gormigrate.Migration{
		ID: "20220524140500",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&KafkaRequest{})
		},
		Rollback: func(tx *gorm.DB) error {
			err := tx.Migrator().DropColumn(&KafkaRequest{}, "billing_cloud_account_id")
			if err != nil {
				return err
			}

			return tx.Migrator().DropColumn(&KafkaRequest{}, "marketplace")
		},
	}
}
