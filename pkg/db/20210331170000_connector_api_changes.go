package db

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func connectorApiChanges() *gormigrate.Migration {

	actions := []func(tx *gorm.DB, do bool) error{

		func(tx *gorm.DB, do bool) error {
			if do {
				return tx.Exec("ALTER TABLE connector_clusters RENAME COLUMN status TO status_phase").Error
			} else {
				return tx.Exec("ALTER TABLE connector_clusters RENAME COLUMN status_phase TO status").Error
			}
		},

		func(tx *gorm.DB, do bool) error {
			type ConnectorClusterStatus struct {
				Version    string `json:"version,omitempty"`
				Conditions string `json:"conditions,omitempty"`
				Operators  string `json:"operators,omitempty"`
			}
			type ConnectorCluster struct {
				Status ConnectorClusterStatus `json:"status" gorm:"embedded;embeddedPrefix:status_"`
			}
			if do {
				return tx.AutoMigrate(&ConnectorCluster{})
			} else {
				err := tx.Migrator().DropColumn(&ConnectorCluster{}, "status_version")
				if err != nil {
					return err
				}
				err = tx.Migrator().DropColumn(&ConnectorCluster{}, "status_conditions")
				if err != nil {
					return err
				}
				err = tx.Migrator().DropColumn(&ConnectorCluster{}, "status_operators")
				if err != nil {
					return err
				}
				return nil
			}
		},

		func(tx *gorm.DB, do bool) error {
			if do {
				return tx.Exec("ALTER TABLE connectors ALTER COLUMN connector_spec TYPE json USING to_json(connector_spec)").Error
			} else {
				return tx.Exec("ALTER TABLE connectors ALTER COLUMN connector_spec TYPE text").Error
			}
		},

		func(tx *gorm.DB, do bool) error {
			if do {
				return tx.Exec("ALTER TABLE connectors RENAME COLUMN addon_group TO addon_cluster_id").Error
			} else {
				return tx.Exec("ALTER TABLE connectors RENAME COLUMN addon_cluster_id TO addon_group").Error
			}
		},

		func(tx *gorm.DB, do bool) error {
			type ConnectorStatus struct {
				Model
				ClusterID string `json:"cluster_id"`
				Phase     string
			}
			type Connector struct {
				Model
				Status ConnectorStatus `gorm:"foreignKey:ID"`
			}
			if do {
				err := tx.AutoMigrate(&ConnectorStatus{})
				if err != nil {
					return err
				}
				err = tx.AutoMigrate(&Connector{})
				if err != nil {
					return err
				}
			} else {
				return tx.Migrator().DropTable(&ConnectorStatus{})
			}
			return nil
		},

		func(tx *gorm.DB, do bool) error {
			type Connector struct {
				Model
				ClusterID string
				Status    string
			}
			if do {
				err := tx.Migrator().DropColumn(&Connector{}, "status")
				if err != nil {
					return err
				}
				err = tx.Migrator().DropColumn(&Connector{}, "cluster_id")
				if err != nil {
					return err
				}
			} else {
				return tx.AutoMigrate(&Connector{})
			}
			return nil
		},

		func(tx *gorm.DB, do bool) error {
			type Connector struct {
				DesiredState string `json:"desired_state"`
			}
			if do {
				return tx.AutoMigrate(&Connector{})
			} else {
				return tx.Migrator().DropColumn(&Connector{}, "desired_state")
			}
		},

		func(tx *gorm.DB, do bool) error {
			type ConnectorClusters struct {
				AddonGroup string
			}
			if do {
				return tx.Migrator().DropColumn(&ConnectorClusters{}, "addon_group")
			} else {
				return tx.AutoMigrate(&ConnectorClusters{})
			}
		},

		func(tx *gorm.DB, do bool) error {

			type ConnectorDeploymentStatus struct {
				Model
				Phase        string
				SpecChecksum string
				Conditions   string `gorm:"type:jsonb;index:"`
			}

			type ConnectorDeployment struct {
				Model
				Version              int64                     `gorm:"type:bigserial;index:"`
				ConnectorID          string                    `json:"connector_id"`
				ConnectorVersion     int64                     `json:"connector_resource_version"`
				Status               ConnectorDeploymentStatus `gorm:"foreignKey:ID"`
				ConnectorTypeService string
				ClusterID            string `gorm:"index:"`
				SpecChecksum         string `json:"spec_checksum,omitempty"`
			}

			if do {
				err := tx.AutoMigrate(&ConnectorDeploymentStatus{})
				if err != nil {
					return err
				}
				err = tx.AutoMigrate(&ConnectorDeployment{})
				if err != nil {
					return err
				}
			} else {
				err := tx.Migrator().DropTable(&ConnectorDeployment{})
				if err != nil {
					return err
				}
				err = tx.Migrator().DropTable(&ConnectorDeploymentStatus{})
				if err != nil {
					return err
				}
			}
			return nil
		},
		func(tx *gorm.DB, do bool) error {
			if do {
				return tx.Exec(`
                CREATE FUNCTION connector_deployments_version_trigger() RETURNS TRIGGER LANGUAGE plpgsql AS '
					BEGIN
					NEW.version := nextval(''connector_deployments_version_seq'');
					RETURN NEW;
					END;
				'
			`).Error
			} else {
				return tx.Exec(`DROP FUNCTION connector_deployments_version_trigger`).Error
			}
		},
		func(tx *gorm.DB, do bool) error {
			if do {
				return tx.Exec(`
				CREATE TRIGGER connector_deployments_version_trigger BEFORE INSERT OR UPDATE ON connector_deployments
				FOR EACH ROW EXECUTE PROCEDURE connector_deployments_version_trigger();
			`).Error
			} else {
				return tx.Exec(`DROP TRIGGER connector_deployments_version_trigger ON connector_deployments`).Error
			}
		},
	}

	return &gormigrate.Migration{
		ID: "202103171200",
		Migrate: func(tx *gorm.DB) error {
			for _, action := range actions {
				err := action(tx, true)
				if err != nil {
					return err
				}
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			for i := len(actions) - 1; i >= 0; i-- {
				err := actions[i](tx, false)
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
}
