package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/go-gormigrate/gormigrate/v2"
)

func addDeletedAtIndex(migrationId string) *gormigrate.Migration {

	// add missing deleted_at index caused by incorrectly using api.Meta instead of db.Model in dbapi.* structs
	return db.CreateMigrationFromActions(migrationId,
		db.ExecAction(`CREATE INDEX on connectors(deleted_at)`,
			`DROP INDEX IF EXISTS connectors_deleted_at_idx`),
		db.ExecAction(`CREATE INDEX on connector_statuses(deleted_at)`,
			`DROP INDEX IF EXISTS connector_statuses_deleted_at_idx`),
		db.ExecAction(`CREATE INDEX on connector_channels(deleted_at)`,
			`DROP INDEX IF EXISTS connector_channels_deleted_at_idx`),
		db.ExecAction(`CREATE INDEX on connector_types(deleted_at)`,
			`DROP INDEX IF EXISTS connector_types_deleted_at_idx`),
		db.ExecAction(`CREATE INDEX on connector_clusters(deleted_at)`,
			`DROP INDEX IF EXISTS connector_clusters_deleted_at_idx`),
	)
}
