package migrations

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/go-gormigrate/gormigrate/v2"
)

// gormigrate is a wrapper for gorm's migration functions that adds schema versioning and rollback capabilities.
// For help writing migration steps, see the gorm documentation on migrations: https://gorm.io/docs/migration.html

// Migration rules:
//
// 1. IDs are numerical timestamps that must sort ascending.
//    Use YYYYMMDDHHMM w/ 24 hour time for format
//    Example: August 21 2018 at 2:54pm would be 201808211454.
//
// 2. Include models inline with migrations to see the evolution of the object over time.
//    Using our internal type models directly in the first migration would fail in future clean installs.
//
// 3. Migrations must be backwards compatible. There are no new required fields allowed.
//    See $project_home/db/README.md
//
// 4. Create one function in a separate file that returns your Migration. Add that single function call to this list.
var migrations = []*gormigrate.Migration{
	addConnectorTables("202106170000"),
	addConnectorTypeTables("202108020000"),
	addDeploymentOperatorId("202110140000"),
	connectorRefactor("202201310000"),
	addConnectorTypeCapabilitiesTable("202202040000"),
	addClientId("202202030000"),
	addConnectorNamespaceTables("202202070000"),
	addConnectorNamespaceDeployment("202202220000"),
	addDeletedAtIndex("202202280000"),
	deleteConnectorTargetKind("202203020000"),
	addConnectorNamespaceLease("202203030000"),
}

func New(dbConfig *db.DatabaseConfig) (*db.Migration, func(), error) {
	return db.NewMigration(dbConfig, &gormigrate.Options{
		TableName:      "connector_migrations",
		IDColumnName:   "id",
		IDColumnSize:   255,
		UseTransaction: false,
	}, migrations)
}
