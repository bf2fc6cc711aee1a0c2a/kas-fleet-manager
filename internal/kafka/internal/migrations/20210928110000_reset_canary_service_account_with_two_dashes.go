package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func resetCanaryServiceAccountWithTwoDashes() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20210928110000",
		Migrate: func(tx *gorm.DB) error {
			var kafkas []dbapi.KafkaRequest
			canaryServiceAccountUpdates := map[string]interface{}{
				"canary_service_account_client_id":     "",
				"canary_service_account_client_secret": "",
			}
			err := tx.Model(&dbapi.KafkaRequest{}).
				Select("id", "canary_service_account_client_id", "canary_service_account_client_secret").
				Where("status = ?", "ready").
				Where("canary_service_account_client_id like ?", "%--%").
				Scan(&kafkas).
				Error

			if err != nil {
				return err
			}

			for _, kafka := range kafkas {
				err = tx.Model(&kafka).Updates(canaryServiceAccountUpdates).Error
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
