package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
	"time"
)

func addProcessorTables(migrationId string) *gormigrate.Migration {
	type ProcessorStatusPhase string

	type ProcessorDesiredState string

	type KafkaConnectionSettings struct {
		KafkaID         string `gorm:"column:id"`
		BootstrapServer string
	}

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
		Key         string `gorm:"primaryKey;index;not null"`
		Value       string `gorm:"not null"`
	}

	type Processor struct {
		db.Model
		Name            string
		NamespaceId     string
		ProcessorTypeId string
		Channel         string
		DesiredState    ProcessorDesiredState
		Owner           string
		CloudProvider   string
		Region          string
		MultiAZ         bool
		OrganisationId  string
		Version         int64                   `gorm:"type:bigserial;index"`
		Annotations     []ProcessorAnnotation   `gorm:"foreignKey:ProcessorID;references:ID"`
		Kafka           KafkaConnectionSettings `gorm:"embedded;embeddedPrefix:kafka_"`
		ServiceAccount  ServiceAccount          `gorm:"embedded;embeddedPrefix:service_account_"`
		Definition      api.JSON                `gorm:"definition"`
		ErrorHandler    api.JSON                `gorm:"error_handler"`
		Status          ProcessorStatus         `gorm:"foreignKey:ID"`
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
		Version                  int64  `gorm:"type:bigserial;index"`
		ProcessorID              string `gorm:"index:idx_processor_deployments_processor_id_idx;unique"`
		Processor                Processor
		ProcessorVersion         int64
		ProcessorShardMetadataID int64
		ProcessorShardMetadata   ProcessorShardMetadata
		ClusterID                string
		NamespaceID              string
		OperatorID               string
		AllowUpgrade             bool
		Status                   ProcessorDeploymentStatus `gorm:"foreignKey:ID;references:ID"`
	}

	type LeaderLease struct {
		db.Model
		Leader    string
		LeaseType string
		Expires   *time.Time
	}

	return db.CreateMigrationFromActions(migrationId,
		db.FuncAction(func(tx *gorm.DB) error {
			// We don't want to delete the leader lease table on rollback because it's shared with the kas-fleet-manager
			// so, we just create it here if it does not exist yet... but we don't drop it on rollback.
			err := tx.Migrator().AutoMigrate(&LeaderLease{})
			if err != nil {
				return err
			}
			now := time.Now().Add(-time.Minute) //set to an expired time
			return tx.Create(&api.LeaderLease{
				Expires:   &now,
				LeaseType: "processor",
			}).Error
		}, func(tx *gorm.DB) error {
			// The leader lease table may have already been dropped, by the kafka migration rollback, ignore error
			_ = tx.Where("lease_type = ?", "processor").Delete(&api.LeaderLease{})
			return nil
		}),
		db.CreateTableAction(&ProcessorStatus{}),
		db.CreateTableAction(&ProcessorAnnotation{}),
		db.CreateTableAction(&Processor{}),
		db.ExecAction(`DROP TRIGGER IF EXISTS processors_version_trigger ON processors`, ``),
		db.ExecAction(`DROP FUNCTION IF EXISTS processors_version_trigger`, ``),
		db.ExecAction(`                
			CREATE OR REPLACE FUNCTION processors_version_trigger() RETURNS TRIGGER LANGUAGE plpgsql AS '
			BEGIN
			NEW.version := nextval(''processors_version_seq'');
			RETURN NEW;
			END;'
		`, `
			DROP FUNCTION IF EXISTS processors_version_trigger
		`),
		db.ExecAction(`
			CREATE TRIGGER processors_version_trigger BEFORE INSERT OR UPDATE ON processors
			FOR EACH ROW EXECUTE PROCEDURE processors_version_trigger();
		`, `
			DROP TRIGGER IF EXISTS processors_version_trigger ON processors
		`),
		db.CreateTableAction(&ProcessorDeploymentStatus{}),
		db.CreateTableAction(&ProcessorDeployment{}),
		db.ExecAction(`DROP TRIGGER IF EXISTS processor_deployments_version_trigger ON processor_deployments`, ``),
		db.ExecAction(`DROP FUNCTION IF EXISTS processor_deployments_version_trigger`, ``),
		db.ExecAction(`
			CREATE OR REPLACE FUNCTION processor_deployments_version_trigger() RETURNS TRIGGER LANGUAGE plpgsql AS '
			BEGIN
			NEW.version := nextval(''processor_deployments_version_seq'');
			RETURN NEW;
			END;'
		`, `
			DROP FUNCTION IF EXISTS processor_deployments_version_trigger
		`),
		db.ExecAction(`
			CREATE TRIGGER processor_deployments_version_trigger BEFORE INSERT OR UPDATE ON processor_deployments
			FOR EACH ROW EXECUTE PROCEDURE processor_deployments_version_trigger();
		`, `
			DROP TRIGGER IF EXISTS processor_deployments_version_trigger ON processor_deployments
		`),
		db.CreateTableAction(&ProcessorShardMetadata{}),
		db.FuncAction(func(tx *gorm.DB) error {
			// Processors don't really support different metadata but include a default until we decide how we want to manage
			return tx.Create(&ProcessorShardMetadata{
				ShardMetadata: nil,
			}).Error
		}, func(tx *gorm.DB) error {
			// No need to delete the row as the whole table will be dropped.
			return nil
		}),
	)
}
