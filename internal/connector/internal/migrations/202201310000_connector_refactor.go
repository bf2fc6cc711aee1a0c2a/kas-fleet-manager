package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func connectorRefactor(migrationId string) *gormigrate.Migration {
	// Meta is base model definition, embedded in all kinds
	type Meta struct {
		ID        string `json:"id"`
		CreatedAt time.Time
		UpdatedAt time.Time
		// needed for soft delete. See https://gorm.io/docs/delete.html#Soft-Delete
		DeletedAt gorm.DeletedAt
	}

	// Connector Model
	type TargetKind = string

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
		Meta
		ClusterID string
		Phase     string
	}

	type Connector struct {
		Meta

		TargetKind     TargetKind
		AddonClusterId string
		CloudProvider  string
		Region         string
		MultiAZ        bool

		Name           string
		Owner          string
		OrganisationId string
		Version        int64 `gorm:"type:bigserial;index:"`

		ConnectorTypeId string
		ConnectorSpec   api.JSON `gorm:"type:jsonb"`
		DesiredState    string
		Channel         string
		Kafka           KafkaConnectionSettings          `gorm:"embedded;embeddedPrefix:kafka_"`
		SchemaRegistry  SchemaRegistryConnectionSettings `gorm:"embedded;embeddedPrefix:schema_registry_"`
		ServiceAccount  ServiceAccount                   `gorm:"embedded;embeddedPrefix:service_account_"`

		Status ConnectorStatus `gorm:"foreignKey:ID"`
	}

	return db.CreateMigrationFromActions(migrationId,
		db.RenameTableColumnAction(&Connector{}, "kafka_client_id", "service_account_client_id"),
		db.RenameTableColumnAction(&Connector{}, "kafka_client_secret", "service_account_client_secret"),
		db.ExecAction(`ALTER TABLE connectors DROP COLUMN kafka_client_secret_ref`,
			`ALTER TABLE connectors ADD kafka_client_secret_ref text`),
		db.AddTableColumnAction(&Connector{}, "SchemaRegistryID"),
		db.AddTableColumnAction(&Connector{}, "Url"),
	)
}
