package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addKafkaBillingModel() *gormigrate.Migration {
	type KafkaRequest struct {
		BillingModel string `json:"billing_model"`
	}

	return &gormigrate.Migration{
		ID: "20220621100000",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&KafkaRequest{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropColumn(&KafkaRequest{}, "billing_model")
		},
	}
}
