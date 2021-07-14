package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addKafkaUpgradeFunctionalityRelatedFields() *gormigrate.Migration {
	type KafkaRequest struct {
		DesiredKafkaVersion   string `json:"desired_kafka_version"`
		DesiredStrimziVersion string `json:"desired_strimzi_version"`
		ActualStrimziVersion  string `json:"actual_strimzi_version"`
		KafkaUpgrading        bool   `json:"kafka_upgrading" gorm:"default:false"`
		StrimziUpgrading      bool   `json:"strimzi_upgrading" gorm:"default:false"`
	}

	return &gormigrate.Migration{
		ID: "20210713135600",
		Migrate: func(tx *gorm.DB) error {
			err := tx.AutoMigrate(&KafkaRequest{})
			if err != nil {
				return err
			}

			err = tx.Exec(`UPDATE kafka_requests SET desired_kafka_version = version WHERE desired_kafka_version IS NULL`).Error
			if err != nil {
				return err
			}

			strimziVersionToSetOnMigration := "strimzi-cluster-operator.v0.23.0-0"

			err = tx.Table("kafka_requests").Where("desired_strimzi_version IS NULL").Update("desired_strimzi_version", strimziVersionToSetOnMigration).Error
			if err != nil {
				return err
			}

			err = tx.Table("kafka_requests").Where("actual_strimzi_version IS NULL").Update("actual_strimzi_version", strimziVersionToSetOnMigration).Error
			if err != nil {
				return err
			}

			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			err := tx.Migrator().DropColumn(&KafkaRequest{}, "desired_kafka_version")
			if err != nil {
				return err
			}

			err = tx.Migrator().DropColumn(&KafkaRequest{}, "desired_strimzi_version")
			if err != nil {
				return err
			}

			err = tx.Migrator().DropColumn(&KafkaRequest{}, "actual_strimzi_version")
			if err != nil {
				return err
			}

			err = tx.Migrator().DropColumn(&KafkaRequest{}, "kafka_upgrading")
			if err != nil {
				return err
			}

			err = tx.Migrator().DropColumn(&KafkaRequest{}, "strimzi_upgrading")
			if err != nil {
				return err
			}

			return nil
		},
	}
}
