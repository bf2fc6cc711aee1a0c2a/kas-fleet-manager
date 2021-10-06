package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"fmt"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func resetCanaryServiceAccountForAffectedInstances() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20211006140000",
		Migrate: func(tx *gorm.DB) error {
			affectedInstancesServiceAccountClientIds := "('canary-c4to1usg3rsm76lahvag', 'canary-c52ajhf2ek06ri44ife0', 'canary-1vfa8vm83xoodssdgafno09lrxy', 'canary-c4n333s82f35bbnmm4n0', 'canary-1vtfoyh6pbl2llnnyycqfdbh1bh')"
			query := fmt.Sprintf("update kafka_requests set canary_service_account_client_id='', canary_service_account_client_secret = '' where canary_service_account_client_id in %s;", affectedInstancesServiceAccountClientIds)
			return tx.Exec(query).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return nil
		},
	}
}
