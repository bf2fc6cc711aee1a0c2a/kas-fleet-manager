package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/go-gormigrate/gormigrate/v2"
)

func addOrgIDAnnotations(migrationID string) *gormigrate.Migration {

	return db.CreateMigrationFromActions(migrationID,
		// add cluster annotations for undeleted clusters with non-null organisation_id
		db.ExecAction(
			"INSERT INTO connector_cluster_annotations (connector_cluster_id, key, value)"+
				" SELECT id, 'cos.bf2.org/organisation-id', organisation_id FROM connector_clusters"+
				" WHERE deleted_at IS NULL AND organisation_id IS NOT NULL ON CONFLICT DO NOTHING",
			""),
		// add connector annotations for undeleted connectors with non-null organisation_id
		db.ExecAction(
			"INSERT INTO connector_annotations (connector_id, key, value)"+
				" SELECT id, 'cos.bf2.org/organisation-id', organisation_id FROM connectors"+
				" WHERE deleted_at IS NULL AND organisation_id IS NOT NULL ON CONFLICT DO NOTHING",
			""),
	)
}
