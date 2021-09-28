package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func resetCanaryServiceAccountWithTwoDashes() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20210928110000",
		Migrate: func(tx *gorm.DB) error {
			return tx.Exec("update kafka_requests set canary_service_account_client_id='', canary_service_account_client_secret = '' where status = 'ready' and canary_service_account_client_id like '%--%';").
				Error
		},
		Rollback: func(tx *gorm.DB) error {
			return nil
		},
	}
}
