package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func renameKafkaVersionField() *gormigrate.Migration {
	const oldKafkaVersionFieldName string = "version"
	const newKafkaActualVersionFieldName string = "actual_kafka_version"

	type KafkaRequest struct {
		Version            string `json:"version"`
		ActualKafkaVersion string `json:"actual_kafka_version"`
	}

	return &gormigrate.Migration{
		ID: "20210713135700",
		Migrate: func(tx *gorm.DB) error {
			if !tx.Migrator().HasColumn(KafkaRequest{}, newKafkaActualVersionFieldName) {
				return tx.Migrator().RenameColumn(KafkaRequest{}, oldKafkaVersionFieldName, newKafkaActualVersionFieldName)
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			if !tx.Migrator().HasColumn(KafkaRequest{}, oldKafkaVersionFieldName) {
				return tx.Migrator().RenameColumn(KafkaRequest{}, newKafkaActualVersionFieldName, oldKafkaVersionFieldName)
			}
			return nil
		},
	}
}
