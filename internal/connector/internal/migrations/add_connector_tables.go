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

func addConnectorTables(migrationId string) *gormigrate.Migration {
	type ConnectorStatus struct {
		db.Model
		ClusterID string
		Phase     string
	}

	type KafkaConnectionSettings struct {
		BootstrapServer string
		ClientId        string
		ClientSecret    string
		ClientSecretRef string
	}

	type Connector struct {
		db.Model

		TargetKind     string
		AddonClusterId string
		CloudProvider  string
		Region         string
		MultiAZ        bool

		Name           string
		Owner          string
		OrganisationId string
		KafkaID        string
		Version        int64 `gorm:"type:bigserial;index:"`

		ConnectorTypeId string
		ConnectorSpec   string `gorm:"type:jsonb"`
		DesiredState    string
		Channel         string
		Kafka           KafkaConnectionSettings `gorm:"embedded;embeddedPrefix:kafka_"`

		Status ConnectorStatus `gorm:"foreignKey:ID"`
	}

	type ConnectorDeploymentStatus struct {
		db.Model
		Phase            string
		Version          int64
		Conditions       string `gorm:"type:jsonb"`
		Operators        string `gorm:"type:jsonb"`
		UpgradeAvailable bool
	}
	// ConnectorDeployment Holds the deployment configuration of a connector
	type ConnectorDeployment struct {
		db.Model
		Version                int64 `gorm:"type:bigserial;index:"`
		ConnectorID            string
		ConnectorVersion       int64
		ConnectorTypeChannelId int64
		ClusterID              string
		AllowUpgrade           bool
		Status                 ConnectorDeploymentStatus `gorm:"foreignKey:ID"`
	}

	type ConnectorClusterStatus struct {
		Phase      string
		Version    string
		Conditions string
		Operators  string
	}
	type ConnectorCluster struct {
		api.Meta
		Owner          string
		OrganisationId string
		Name           string
		Status         ConnectorClusterStatus `gorm:"embedded;embeddedPrefix:status_"`
	}

	type ConnectorShardMetadata struct {
		ID              int64  `gorm:"primaryKey;autoIncrement"`
		ConnectorTypeId string `gorm:"primaryKey"`
		Channel         string `gorm:"primaryKey"`
		ShardMetadata   string `gorm:"type:jsonb"`
		LatestId        *int64
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
			// so we just create it here if it does not exist yet.. but we don't drop it on rollback.
			err := tx.Migrator().AutoMigrate(&LeaderLease{})
			if err != nil {
				return err
			}
			now := time.Now().Add(-time.Minute) //set to a expired time
			return tx.Create(&api.LeaderLease{
				Expires:   &now,
				LeaseType: "connector",
			}).Error
		}, func(tx *gorm.DB) error {
			// The leader lease table may have already been dropped, by the kafka migration rollback, ignore error
			_ = tx.Where("lease_type = ?", "connector").Delete(&api.LeaderLease{})
			return nil
		}),
		db.CreateTableAction(&ConnectorStatus{}),
		db.CreateTableAction(&Connector{}),
		db.ExecAction(`                
			CREATE OR REPLACE FUNCTION connectors_version_trigger() RETURNS TRIGGER LANGUAGE plpgsql AS '
			BEGIN
			NEW.version := nextval(''connectors_version_seq'');
			RETURN NEW;
			END;'
		`, `
			DROP FUNCTION connectors_version_trigger
		`),
		db.ExecAction(`DROP TRIGGER IF EXISTS connectors_version_trigger ON connectors`, ``),
		db.ExecAction(`
			CREATE TRIGGER connectors_version_trigger BEFORE INSERT OR UPDATE ON connectors
			FOR EACH ROW EXECUTE PROCEDURE connectors_version_trigger();
		`, `
			DROP TRIGGER connectors_version_trigger ON connectors
		`),

		db.CreateTableAction(&ConnectorDeploymentStatus{}),
		db.CreateTableAction(&ConnectorDeployment{}),
		db.ExecAction(`
			CREATE OR REPLACE FUNCTION connector_deployments_version_trigger() RETURNS TRIGGER LANGUAGE plpgsql AS '
			BEGIN
			NEW.version := nextval(''connector_deployments_version_seq'');
			RETURN NEW;
			END;'
		`, `
			DROP FUNCTION connector_deployments_version_trigger
		`),
		db.ExecAction(`DROP TRIGGER IF EXISTS connector_deployments_version_trigger ON connector_deployments`, ``),
		db.ExecAction(`
			CREATE TRIGGER connector_deployments_version_trigger BEFORE INSERT OR UPDATE ON connector_deployments
			FOR EACH ROW EXECUTE PROCEDURE connector_deployments_version_trigger();
		`, `
			DROP TRIGGER connector_deployments_version_trigger ON connector_deployments
		`),
		db.CreateTableAction(&ConnectorCluster{}),
		db.CreateTableAction(&ConnectorShardMetadata{}),
	)
}
