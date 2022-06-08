package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/go-gormigrate/gormigrate/v2"
)

func fixConnectorNamespaceVersionTrigger(migrationId string) *gormigrate.Migration {

	return db.CreateMigrationFromActions(migrationId,
		db.ExecAction(`DROP TRIGGER IF EXISTS connector_namespaces_version_trigger ON connector_namespaces`, ``),
		db.ExecAction(`
			CREATE TRIGGER connector_namespaces_version_insert_trigger BEFORE INSERT ON connector_namespaces
			FOR EACH ROW EXECUTE PROCEDURE connector_namespaces_version_trigger();
		`, `
			DROP TRIGGER IF EXISTS connector_namespaces_version_insert_trigger ON connector_namespaces
		`),
		db.ExecAction(`
			CREATE TRIGGER connector_namespaces_version_update_trigger BEFORE UPDATE OF name, status_phase ON connector_namespaces
			FOR EACH ROW EXECUTE PROCEDURE connector_namespaces_version_trigger();
		`, `
			DROP TRIGGER IF EXISTS connector_namespaces_version_update_trigger ON connector_namespaces
		`),
	)
}
