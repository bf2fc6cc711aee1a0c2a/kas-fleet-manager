package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func renameKafkaStorageSizeColumn() *gormigrate.Migration {
	const KafkaStorageSize string = "kafka_storage_size"
	const MaxDataRetentionSize string = "max_data_retention_size"

	type KafkaRequest struct {
		KafkaStorageSize     string `json:"kafka_storage_size"`
		MaxDataRetentionSize string `json:"max_data_retention_size"`
	}

	return &gormigrate.Migration{
		ID: "20230214160000",
		Migrate: func(tx *gorm.DB) error {
			if !tx.Migrator().HasColumn(KafkaRequest{}, MaxDataRetentionSize) {
				return tx.Migrator().RenameColumn(KafkaRequest{}, KafkaStorageSize, MaxDataRetentionSize)
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			if !tx.Migrator().HasColumn(KafkaRequest{}, KafkaStorageSize) {
				return tx.Migrator().RenameColumn(KafkaRequest{}, MaxDataRetentionSize, KafkaStorageSize)
			}
			return nil
		},
	}
}
