package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func renameKafkaBillingModelColumn() *gormigrate.Migration {
	const oldKafkaBillingModelFieldName string = "billing_model"
	const newKafkaBillingModelFieldName string = "actual_kafka_billing_model"

	type KafkaRequest struct {
		BillingModel            string `json:"billing_model"`
		ActualKafkaBillingModel string `json:"actual_kafka_billing_model"`
	}

	return &gormigrate.Migration{
		ID: "20221109150000",
		Migrate: func(tx *gorm.DB) error {
			if !tx.Migrator().HasColumn(KafkaRequest{}, newKafkaBillingModelFieldName) {
				return tx.Migrator().RenameColumn(KafkaRequest{}, oldKafkaBillingModelFieldName, newKafkaBillingModelFieldName)
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			if !tx.Migrator().HasColumn(KafkaRequest{}, oldKafkaBillingModelFieldName) {
				return tx.Migrator().RenameColumn(KafkaRequest{}, newKafkaBillingModelFieldName, oldKafkaBillingModelFieldName)
			}
			return nil
		},
	}
}
