package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/go-gormigrate/gormigrate/v2"
)

func addConnectorResourceAnnotations(migrationID string) *gormigrate.Migration {

	type ConnectorClusterAnnotation struct {
		ConnectorClusterID string `gorm:"primaryKey;index"`
		Key                string `gorm:"primaryKey;not null"`
		Value              string `gorm:"not null"`
	}

	type ConnectorTypeAnnotation struct {
		ConnectorTypeID string `gorm:"primaryKey;index"`
		Key             string `gorm:"primaryKey;not null"`
		Value           string `gorm:"not null"`
	}

	type ConnectorAnnotation struct {
		ConnectorID string `gorm:"primaryKey;index"`
		Key         string `gorm:"primaryKey;not null"`
		Value       string `gorm:"not null"`
	}

	return db.CreateMigrationFromActions(migrationID,
		db.CreateTableAction(&ConnectorClusterAnnotation{}),
		db.CreateTableAction(&ConnectorTypeAnnotation{}),
		db.CreateTableAction(&ConnectorAnnotation{}),

		// relationship between annotations and resources
		db.ExecAction(
			"ALTER TABLE connector_cluster_annotations ADD CONSTRAINT fk_connector_clusters_annotations "+
				"FOREIGN KEY (connector_cluster_id) REFERENCES connector_clusters(id)",
			"ALTER TABLE connector_cluster_annotations DROP CONSTRAINT IF EXISTS fk_connector_clusters_annotations"),
		db.ExecAction(
			"ALTER TABLE connector_type_annotations ADD CONSTRAINT fk_connector_types_annotations "+
				"FOREIGN KEY (connector_type_id) REFERENCES connector_types(id)",
			"ALTER TABLE connector_type_annotations DROP CONSTRAINT IF EXISTS fk_connector_types_annotations"),
		db.ExecAction(
			"ALTER TABLE connector_annotations ADD CONSTRAINT fk_connectors_annotations "+
				"FOREIGN KEY (connector_id) REFERENCES connectors(id)",
			"ALTER TABLE connector_annotations DROP CONSTRAINT IF EXISTS fk_connectors_annotations"),
	)
}
