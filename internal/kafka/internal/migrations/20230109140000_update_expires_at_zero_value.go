package migrations

import (
	"database/sql"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func updateExpiresAtZeroValueFromKafkaRequests() *gormigrate.Migration {
	nullTime := sql.NullTime{Time: time.Time{}, Valid: false}
	zeroTime := time.Time{}

	return &gormigrate.Migration{
		ID: "20230109140000",
		Migrate: func(tx *gorm.DB) error {
			err := tx.Table("kafka_requests").Where("expires_at = ?", zeroTime).Where("deleted_at IS NULL").Update("expires_at", nullTime).Error
			if err != nil {
				return err
			}

			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			err := tx.Table("kafka_requests").Where("expires_at = ?", nullTime).Where("deleted_at IS NULL").Update("expires_at", zeroTime).Error
			if err != nil {
				return err
			}

			return nil
		},
	}
}
