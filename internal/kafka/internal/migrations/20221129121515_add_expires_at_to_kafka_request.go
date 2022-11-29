package migrations

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addExpiresAtToKafkaRequest() *gormigrate.Migration {
	type KafkaRequest struct {
		ExpiresAt time.Time
	}

	return &gormigrate.Migration{
		ID: "20221129121515",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&KafkaRequest{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropColumn(&KafkaRequest{}, "expires_at")
		},
	}
}
