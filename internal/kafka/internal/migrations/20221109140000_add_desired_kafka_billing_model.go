package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addDesiredKafkaBillingModel() *gormigrate.Migration {
	type KafkaRequest struct {
		DesiredKafkaBillingModel string `json:"desired_kafka_billing_model"`
	}

	return &gormigrate.Migration{
		ID: "20221109140000",
		Migrate: func(tx *gorm.DB) error {
			err := tx.AutoMigrate(&KafkaRequest{})
			if err != nil {
				return err
			}

			// For each row where desired_kafka_billing_model is NULL we update
			// desired_kafka_billing_model's value to the value of the billing_model
			// column
			err = tx.Table("kafka_requests").Where("desired_kafka_billing_model IS NULL").Update("desired_kafka_billing_model", gorm.Expr("billing_model")).Error
			if err != nil {
				return err
			}

			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			err := tx.Migrator().DropColumn(&KafkaRequest{}, "desired_kafka_billing_model")
			if err != nil {
				return err
			}

			return nil
		},
	}
}
