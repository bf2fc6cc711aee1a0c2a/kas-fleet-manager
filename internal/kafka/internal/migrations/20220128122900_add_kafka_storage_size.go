package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addKafkaStorageSize() *gormigrate.Migration {
	type KafkaRequest struct {
		KafkaStorageSize string `json:"kafka_storage_size" gorm:"default:'60Gi'"`
		// `json:"supported_instance_type" gorm:"default:'standard,eval';"` // by default support both instance types
	}

	return &gormigrate.Migration{
		ID: "20220128122900",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&KafkaRequest{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropColumn(&KafkaRequest{}, "kafka_storage_size")
		},
	}
}
