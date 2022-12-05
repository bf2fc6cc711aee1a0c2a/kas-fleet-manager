package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/go-gormigrate/gormigrate/v2"
)

func renameNamespaceProfileAnnotations(migrationID string) *gormigrate.Migration {

	return db.CreateMigrationFromActions(migrationID,
		// rename namespace profile annotation keys to comply with k8s label regex
		db.ExecAction(
			"UPDATE connector_namespace_annotations"+
				" SET key = 'cos.bf2.org/profile' WHERE key = 'connector_mgmt.bf2.org/profile'",
			"UPDATE connector_namespace_annotations"+
				" SET key = 'connector_mgmt.bf2.org/profile' WHERE key = 'cos.bf2.org/profile'"),
	)
}
