package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/go-gormigrate/gormigrate/v2"
)

func addConnectorClusterPlatform(migrationId string) *gormigrate.Migration {
	type ConnectorClusterPlatform struct {
		ID      string
		Type    string
		Version string
	}

	type ConnectorClusterStatus struct {
		Platform ConnectorClusterPlatform `gorm:"embedded;embeddedPrefix:platform_"`
	}

	type ConnectorCluster struct {
		Status ConnectorClusterStatus `gorm:"embedded;embeddedPrefix:status_"`
	}

	return db.CreateMigrationFromActions(migrationId,
		// add platform status
		db.AddTableColumnsAction(&ConnectorCluster{}),
	)
}
