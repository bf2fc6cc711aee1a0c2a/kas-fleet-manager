package migrations

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func migrateOldKafkaNamespace() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20210809140000",
		Migrate: func(tx *gorm.DB) error {
			var kafkas []dbapi.KafkaRequest
			err := tx.Unscoped().Model(&dbapi.KafkaRequest{}).Where("namespace = ?", "").Scan(&kafkas).Error
			if err != nil {
				return err
			}

			for _, kafka := range kafkas {
				namespace, err := services.BuildNamespaceName(&kafka)
				if err != nil {
					return err
				}
				err = tx.Model(&kafka).Update("namespace", namespace).Error
				if err != nil {
					return err
				}
			}

			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return nil
		},
	}
}
