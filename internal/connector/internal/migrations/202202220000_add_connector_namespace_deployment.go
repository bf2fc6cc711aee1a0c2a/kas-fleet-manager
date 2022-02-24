package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/go-gormigrate/gormigrate/v2"
)

func addConnectorNamespaceDeployment(migrationId string) *gormigrate.Migration {

	return db.CreateMigrationFromActions(migrationId,
		// replace column cluster_id with namespace_id in connectors
		db.ExecAction(`ALTER TABLE connectors DROP COLUMN addon_cluster_id`,
		`ALTER TABLE connectors ADD addon_cluster_id text`),
		db.ExecAction(`ALTER TABLE connectors ADD namespace_id text`,
			`ALTER TABLE connectors DROP COLUMN namespace_id`),
		db.ExecAction(
			"ALTER TABLE connectors ADD CONSTRAINT fk_connectors_namespace_id "+
				"FOREIGN KEY (namespace_id) REFERENCES connector_namespaces(id)",
			"ALTER TABLE connectors DROP CONSTRAINT IF EXISTS fk_connectors_namespace_id"),
		db.ExecAction(
			"CREATE INDEX ON connectors(namespace_id)",
			"DROP INDEX IF EXISTS connectors_namespace_id_idx"),

		// replace column cluster_id with namespace_id in connector_statuses
		db.ExecAction(`ALTER TABLE connector_statuses DROP COLUMN cluster_id`,
			`ALTER TABLE connector_statuses ADD cluster_id text`),
		db.ExecAction(`ALTER TABLE connector_statuses ADD namespace_id text`,
			`ALTER TABLE connector_statuses DROP COLUMN namespace_id`),
		db.ExecAction(
			"ALTER TABLE connector_statuses ADD CONSTRAINT fk_connector_statuses_namespace_id "+
				"FOREIGN KEY (namespace_id) REFERENCES connector_namespaces(id)",
			"ALTER TABLE connector_statuses DROP CONSTRAINT IF EXISTS fk_connector_statuses_namespace_id"),
		db.ExecAction(
			"CREATE INDEX ON connector_statuses(namespace_id)",
			"DROP INDEX IF EXISTS connector_statuses_namespace_id_idx"),

		// add column namespace_id in connector_deployments, keeping cluster_id for agent API
		db.ExecAction(`ALTER TABLE connector_deployments ADD namespace_id text not null`,
			`ALTER TABLE connector_deployments DROP COLUMN namespace_id`),
		db.ExecAction(
			"ALTER TABLE connector_deployments DROP CONSTRAINT IF EXISTS fk_connector_deployments_namespace_id",
			""),
		db.ExecAction(
			"ALTER TABLE connector_deployments ADD CONSTRAINT fk_connector_deployments_namespace_id "+
				"FOREIGN KEY (namespace_id) REFERENCES connector_namespaces(id)",
			"ALTER TABLE connector_deployments DROP CONSTRAINT IF EXISTS fk_connector_deployments_namespace_id"),
		db.ExecAction(
			"CREATE INDEX ON connector_deployments(namespace_id)",
			"DROP INDEX IF EXISTS connector_deployments_namespace_id_idx"),
	)
}
