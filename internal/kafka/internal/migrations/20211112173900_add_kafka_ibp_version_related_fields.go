package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addKafkaIBPVersionRelatedFields() *gormigrate.Migration {
	type KafkaRequest struct {
		DesiredKafkaIBPVersion string `json:"desired_kafka_ibp_version"`
		ActualKafkaIBPVersion  string `json:"actual_kafka_ibp_version"`
		KafkaIBPUpgrading      bool   `json:"kafka_ibp_upgrading" gorm:"default:false"` // TODO is this needed
	}

	return &gormigrate.Migration{
		ID: "20211112173900",
		Migrate: func(tx *gorm.DB) error {
			err := tx.AutoMigrate(&KafkaRequest{})
			if err != nil {
				return err
			}

			err = tx.Exec(`UPDATE kafka_requests SET desired_kafka_ibp_version = version WHERE desired_kafka_ibp_version IS NULL`).Error
			if err != nil {
				return err
			}

			kafkaIBPVersionToSetOnMigration := "TODO"

			err = tx.Table("kafka_requests").Where("desired_kafka_ibp_version IS NULL").Update("desired_kafka_ibp_version", kafkaIBPVersionToSetOnMigration).Error
			if err != nil {
				return err
			}

			err = tx.Table("kafka_requests").Where("actual_kafka_ibp_version IS NULL").Update("actual_kafka_ibp_version", kafkaIBPVersionToSetOnMigration).Error
			if err != nil {
				return err
			}

			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			err := tx.Migrator().DropColumn(&KafkaRequest{}, "desired_kafka_ibp_version")
			if err != nil {
				return err
			}

			err = tx.Migrator().DropColumn(&KafkaRequest{}, "actual_kafka_ibp_version")
			if err != nil {
				return err
			}

			err = tx.Migrator().DropColumn(&KafkaRequest{}, "kafka_ibp_upgrading")
			if err != nil {
				return err
			}

			return nil
		},
	}
}
