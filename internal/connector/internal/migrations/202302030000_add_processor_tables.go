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

func addProcessorTables(migrationId string) *gormigrate.Migration {
	type ProcessorStatusPhase string

	type ProcessorDesiredState string

	type ServiceAccount struct {
		ClientId        string
		ClientSecret    string `gorm:"-"`
		ClientSecretRef string `gorm:"column:client_secret"`
	}

	type ProcessorStatus struct {
		db.Model
		NamespaceID *string
		Phase       ProcessorStatusPhase
	}

	type ProcessorAnnotation struct {
		ProcessorID string `gorm:"primaryKey;index"`
		Key         string `gorm:"primaryKey;not null"`
		Value       string `gorm:"not null"`
	}

	type Processor struct {
		db.Model

		NamespaceId   *string
		CloudProvider string
		Region        string
		MultiAZ       bool

		Name           string
		Owner          string
		OrganisationId string
		Version        int64                 `gorm:"type:bigserial;index:"`
		Annotations    []ProcessorAnnotation `gorm:"foreignKey:ProcessorID;references:ID"`

		ProcessorSpec  api.JSON `gorm:"type:jsonb"`
		DesiredState   ProcessorDesiredState
		ServiceAccount ServiceAccount `gorm:"embedded;embeddedPrefix:service_account_"`

		Status ProcessorStatus `gorm:"foreignKey:ID"`
	}

	type ProcessorShardMetadata struct {
		ID             int64 `gorm:"primaryKey:autoIncrement"`
		Revision       int64 `gorm:"index:idx_processor_shard_revision;default:0"`
		LatestRevision *int64
		ShardMetadata  api.JSON `gorm:"type:jsonb"`
	}

	type ProcessorDeploymentStatus struct {
		db.Model
		Phase            ProcessorStatusPhase
		Version          int64
		Conditions       api.JSON `gorm:"type:jsonb"`
		Operators        api.JSON `gorm:"type:jsonb"`
		UpgradeAvailable bool
	}

	type ProcessorDeployment struct {
		db.Model
		Version                  int64
		ProcessorID              string
		Processor                Processor
		OperatorID               string
		ProcessorVersion         int64
		ProcessorShardMetadataID int64
		ProcessorShardMetadata   ProcessorShardMetadata
		ClusterID                string
		NamespaceID              string
		AllowUpgrade             bool
		Status                   ProcessorDeploymentStatus `gorm:"foreignKey:ID;references:ID"`
		Annotations              []ProcessorAnnotation     `gorm:"foreignKey:ProcessorID;references:ProcessorID"`
	}

	return db.CreateMigrationFromActions(migrationId,
		db.CreateTableAction(&ProcessorStatus{}),
		db.CreateTableAction(&ProcessorAnnotation{}),
		db.CreateTableAction(&Processor{}),
		db.ExecAction(`                
			CREATE OR REPLACE FUNCTION processors_version_trigger() RETURNS TRIGGER LANGUAGE plpgsql AS '
			BEGIN
			NEW.version := nextval(''processors_version_seq'');
			RETURN NEW;
			END;'
		`, `
			DROP FUNCTION processors_version_trigger
		`),
		db.ExecAction(`DROP TRIGGER IF EXISTS processors_version_trigger ON processors`, ``),
		db.ExecAction(`
			CREATE TRIGGER processors_version_trigger BEFORE INSERT OR UPDATE ON processors
			FOR EACH ROW EXECUTE PROCEDURE processors_version_trigger();
		`, `
			DROP TRIGGER processors_version_trigger ON processors
		`),
		db.CreateTableAction(&ProcessorDeploymentStatus{}),
		db.CreateTableAction(&ProcessorDeployment{}),
		db.ExecAction(`
			CREATE OR REPLACE FUNCTION processor_deployments_version_trigger() RETURNS TRIGGER LANGUAGE plpgsql AS '
			BEGIN
			NEW.version := nextval(''processor_deployments_version_seq'');
			RETURN NEW;
			END;'
		`, `
			DROP FUNCTION processor_deployments_version_trigger
		`),
		db.ExecAction(`DROP TRIGGER IF EXISTS processor_deployments_version_trigger ON processor_deployments`, ``),
		db.ExecAction(`
			CREATE TRIGGER processor_deployments_version_trigger BEFORE INSERT OR UPDATE ON processor_deployments
			FOR EACH ROW EXECUTE PROCEDURE processor_deployments_version_trigger();
		`, `
			DROP TRIGGER processor_deployments_version_trigger ON processor_deployments
		`),
		db.CreateTableAction(&ProcessorShardMetadata{}),
	)
}
