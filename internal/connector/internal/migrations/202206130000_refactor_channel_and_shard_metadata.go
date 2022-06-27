package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/go-gormigrate/gormigrate/v2"
)

func refactorChannelAndShardMetadata(migrationId string) *gormigrate.Migration {
	type ConnectorShardMetadata struct {
		ID              int64  `gorm:"primaryKey:autoIncrement"`
		ConnectorTypeId string `gorm:"index:idx_typeid_channel_revision;index:idx_typeid_channel"`
		Channel         string `gorm:"index:idx_typeid_channel_revision;index:idx_typeid_channel"`
		Revision        int64  `gorm:"index:idx_typeid_channel_revision;default:0"`
		LatestRevision  *int64
		ShardMetadata   api.JSON `gorm:"type:jsonb"`
	}

	type ConnectorStatusPhase string

	type ConnectorDeploymentStatus struct {
		db.Model
		Phase            ConnectorStatusPhase
		Version          int64
		Conditions       api.JSON `gorm:"type:jsonb"`
		Operators        api.JSON `gorm:"type:jsonb"`
		UpgradeAvailable bool
	}

	type ConnectorDesiredState string

	type KafkaConnectionSettings struct {
		KafkaID         string `gorm:"column:id"`
		BootstrapServer string
	}

	type SchemaRegistryConnectionSettings struct {
		SchemaRegistryID string `gorm:"column:id"`
		Url              string
	}

	type ServiceAccount struct {
		ClientId        string
		ClientSecret    string `gorm:"-"`
		ClientSecretRef string `gorm:"column:client_secret"`
	}

	type ConnectorStatus struct {
		db.Model
		NamespaceID *string
		Phase       ConnectorStatusPhase
	}

	type Connector struct {
		db.Model

		NamespaceId   *string
		CloudProvider string
		Region        string
		MultiAZ       bool

		Name           string
		Owner          string
		OrganisationId string
		Version        int64 `gorm:"type:bigserial;index:"`

		ConnectorTypeId string
		ConnectorSpec   api.JSON `gorm:"type:jsonb"`
		DesiredState    ConnectorDesiredState
		Channel         string
		Kafka           KafkaConnectionSettings          `gorm:"embedded;embeddedPrefix:kafka_"`
		SchemaRegistry  SchemaRegistryConnectionSettings `gorm:"embedded;embeddedPrefix:schema_registry_"`
		ServiceAccount  ServiceAccount                   `gorm:"embedded;embeddedPrefix:service_account_"`

		Status ConnectorStatus `gorm:"foreignKey:ID"`
	}

	type ConnectorDeployment struct {
		db.Model
		Version                  int64
		ConnectorID              string
		Connector                Connector
		OperatorID               string
		ConnectorVersion         int64
		ConnectorShardMetadataID int64
		ConnectorShardMetadata   ConnectorShardMetadata
		ClusterID                string
		NamespaceID              string
		AllowUpgrade             bool
		Status                   ConnectorDeploymentStatus `gorm:"foreignKey:ID;references:ID"`
	}

	return db.CreateMigrationFromActions(migrationId,
		db.ExecAction("ALTER TABLE connector_shard_metadata DROP CONSTRAINT connector_shard_metadata_pkey", "ALTER TABLE connector_shard_metadata ADD PRIMARY KEY (id,connector_type_id,channel)"),
		db.ExecAction("ALTER TABLE connector_shard_metadata ADD PRIMARY KEY (id)", "ALTER TABLE connector_shard_metadata DROP CONSTRAINT connector_shard_metadata_pkey"),
		db.DropTableColumnAction(&ConnectorShardMetadata202206130000{}, "latest_id"),
		db.AddTableColumnAction(&ConnectorShardMetadata{}, "revision"),
		db.AddTableColumnAction(&ConnectorShardMetadata{}, "latest_revision"),
		db.ExecAction("CREATE INDEX idx_typeid_channel_revision ON connector_shard_metadata(connector_type_id, channel, revision)", "DROP INDEX idx_typeid_channel_revision"),
		db.ExecAction("CREATE INDEX idx_typeid_channel ON connector_shard_metadata(connector_type_id, channel)", "DROP INDEX idx_typeid_channel"),
		db.RenameTableColumnAction(&ConnectorDeployment{}, "connector_type_channel_id", "connector_shard_metadata_id"),
	)
}

type ConnectorShardMetadata202206130000 struct {
	ID              int64    `gorm:"primaryKey:autoIncrement"`
	ConnectorTypeId string   `gorm:"primaryKey"`
	Channel         string   `gorm:"primaryKey"`
	ShardMetadata   api.JSON `gorm:"type:jsonb"`
	LatestId        *int64
}

func (ConnectorShardMetadata202206130000) TableName() string {
	return "connector_shard_metadata"
}
