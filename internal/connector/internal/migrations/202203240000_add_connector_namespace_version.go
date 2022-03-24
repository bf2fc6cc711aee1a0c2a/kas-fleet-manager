package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/go-gormigrate/gormigrate/v2"
)

func addConnectorNamespaceVersion(migrationId string) *gormigrate.Migration {

	// drop unused version column
	type ConnectorNamespace struct {
		Version int64 `gorm:"type:bigserial;index"`
	}

	return db.CreateMigrationFromActions(migrationId,
		// add namespace version
		db.AddTableColumnsAction(&ConnectorNamespace{}),

		// connector namespace version
		db.ExecAction(`
			CREATE OR REPLACE FUNCTION connector_namespaces_version_trigger() RETURNS TRIGGER LANGUAGE plpgsql AS '
			BEGIN
			NEW.version := nextval(''connector_namespaces_version_seq'');
			RETURN NEW;
			END;'
		`, `
			DROP FUNCTION IF EXISTS connector_namespaces_version_trigger
		`),
		db.ExecAction(`DROP TRIGGER IF EXISTS connector_namespaces_version_trigger ON connector_namespaces`, ``),
		db.ExecAction(`
			CREATE TRIGGER connector_namespaces_version_trigger BEFORE INSERT OR UPDATE ON connector_namespaces
			FOR EACH ROW EXECUTE PROCEDURE connector_namespaces_version_trigger();
		`, `
			DROP TRIGGER IF EXISTS connector_namespaces_version_trigger ON connector_namespaces
		`),
	)
}
