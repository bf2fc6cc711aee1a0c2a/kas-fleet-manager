package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/go-gormigrate/gormigrate/v2"
)

func addConnectorNamespaceStatus(migrationId string) *gormigrate.Migration {

	type ConnectorNamespaceStatus struct {
		Phase string `gorm:"not null;index"`
		// the version of the agent
		Version            string
		ConnectorsDeployed int16 `gorm:"not null"`
		Conditions         string
	}

	type ConnectorNamespace struct {
		Status ConnectorNamespaceStatus `gorm:"embedded;embeddedPrefix:status_"`
	}

	// drop unused version column
	type ConnectorNamespaceWithVersion struct {
		Version int64 `gorm:"type:bigserial;index"`
	}

	return db.CreateMigrationFromActions(migrationId,
		// add namespace status
		db.AddTableColumnsAction(&ConnectorNamespace{}),

		// remove unused namespace version, trigger and function
		db.ExecAction(`
			DROP TRIGGER IF EXISTS connector_namespaces_version_trigger ON connector_namespaces
		`, `
			CREATE TRIGGER connector_namespaces_version_trigger BEFORE INSERT OR UPDATE ON connector_namespaces
			FOR EACH ROW EXECUTE PROCEDURE connector_namespaces_version_trigger();
		`),
		db.ExecAction(`
			DROP FUNCTION IF EXISTS connector_namespaces_version_trigger
		`, `
			CREATE OR REPLACE FUNCTION connector_namespaces_version_trigger() RETURNS TRIGGER LANGUAGE plpgsql AS '
			BEGIN
			NEW.version := nextval(''connector_namespaces_version_seq'');
			RETURN NEW;
			END;'
		`),
		db.DropTableColumnsAction(&ConnectorNamespaceWithVersion{}, "connector_namespaces"),
	)
}
