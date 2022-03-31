package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/go-gormigrate/gormigrate/v2"
)

func addConnectorClusterClientSecret(migrationId string) *gormigrate.Migration {

	type ConnectorCluster struct {
		ClientSecret   string `gorm:"not null"`
	}

	return db.CreateMigrationFromActions(migrationId,
		// add client id and secret
		db.AddTableColumnsAction(&ConnectorCluster{}),

		// make client_id not null
		db.ExecAction(`ALTER TABLE connector_clusters ALTER COLUMN client_id SET NOT NULL`,
			`ALTER TABLE connector_clusters ALTER COLUMN client_id DROP NOT NULL`),
	)
}
