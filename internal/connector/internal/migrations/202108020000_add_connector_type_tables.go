package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/db"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addConnectorTypeTables(migrationId string) *gormigrate.Migration {
	type ConnectorChannel struct {
		Channel   string `gorm:"primaryKey"`
		CreatedAt time.Time
		UpdatedAt time.Time
		// needed for soft delete. See https://gorm.io/docs/delete.html#Soft-Delete
		DeletedAt gorm.DeletedAt
	}
	type ConnectorTypeLabel struct {
		ConnectorTypeID string `gorm:"primaryKey"`
		Label           string `gorm:"primaryKey"`
	}
	type ConnectorType struct {
		api.Meta
		Version     string
		Name        string `gorm:"index"`
		Description string
		// A json schema that can be used to validate a connector's connector_spec field.
		JsonSchema api.JSON `gorm:"type:jsonb"`

		// Type's channels
		Channels []ConnectorChannel `gorm:"many2many:connector_type_channels;"`
		// URL to an icon of the connector.
		IconHref string
		// labels used to categorize the connector
		Labels []ConnectorTypeLabel `gorm:"foreignKey:ConnectorTypeID"`
	}
	type ConnectorShardMetadata struct {
		ID              int64  `gorm:"primaryKey:autoIncrement"`
		ConnectorTypeId string `gorm:"primaryKey"`
		Channel         string `gorm:"primaryKey"`
		ShardMetadata   string `gorm:"type:jsonb"`
		LatestId        *int64
	}

	return db.CreateMigrationFromActions(migrationId,
		db.ExecAction("", "DROP TABLE connector_type_channels"),
		db.CreateTableAction(&ConnectorTypeLabel{}),
		db.CreateTableAction(&ConnectorChannel{}),
		db.CreateTableAction(&ConnectorShardMetadata{}),
		db.CreateTableAction(&ConnectorType{}),
		db.ExecAction(
			"ALTER TABLE connector_shard_metadata ADD CONSTRAINT fk_connector_shard_metadata_connector_channels FOREIGN KEY (channel) REFERENCES connector_channels(channel)",
			"ALTER TABLE connector_shard_metadata DROP CONSTRAINT IF EXISTS fk_connector_shard_metadata_connector_channels"),
	)
}
