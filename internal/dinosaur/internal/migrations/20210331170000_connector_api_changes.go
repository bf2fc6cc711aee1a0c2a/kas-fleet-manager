package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/db"
	"github.com/go-gormigrate/gormigrate/v2"
)

func connectorApiChanges() *gormigrate.Migration {
	// this was a connector migration which was refactor into the
	// internal/connector/internal/migrations package.  This becomes an empty
	// migration to stay backward compatible.
	return db.CreateMigrationFromActions("202103171200")
}
