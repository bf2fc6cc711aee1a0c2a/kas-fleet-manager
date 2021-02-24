package db

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
	"time"
)

func addConnectorClusters() *gormigrate.Migration {
	type ConnectorClusters struct {
		Model
		Owner  string `json:"owner"`
		Name   string `json:"name"`
		Group  string `json:"group"`
		Status string `json:"status"`
	}

	type Connectors struct {
		Version    int64  `gorm:"type:bigserial;index:"`
		TargetKind string `json:"target_kind"`
		AddonGroup string `json:"addon_group"`
	}

	return &gormigrate.Migration{
		ID: "202102190900",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&ConnectorClusters{}).Error; err != nil {
				return err
			}

			if err := tx.AutoMigrate(&Connectors{}).Error; err != nil {
				return err
			}

			if err := tx.Exec(`CREATE SEQUENCE connector_version_seq OWNED BY connectors.version`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`
                CREATE FUNCTION connector_version_trigger() RETURNS TRIGGER LANGUAGE plpgsql AS '
					BEGIN
					NEW.version := nextval(''connector_version_seq'');
					RETURN NEW;
					END;
				'
			`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`
				CREATE TRIGGER connector_version_trigger BEFORE INSERT OR UPDATE ON connectors
				FOR EACH ROW EXECUTE PROCEDURE connector_version_trigger();
			`).Error; err != nil {
				return err
			}

			now := time.Now().Add(-time.Minute) //set to a expired time
			if err := tx.Create(&api.LeaderLease{
				Expires:   &now,
				LeaseType: "connector",
			}).Error; err != nil {
				return err
			}

			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.Where("lease_type", "connector").Delete(&api.LeaderLease{}).Error; err != nil {
				return err
			}
			if err := tx.Exec(`DROP TRIGGER connector_version_trigger ON connectors`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`DROP FUNCTION connector_version_trigger`).Error; err != nil {
				return err
			}
			if err := tx.Table("connectors").DropColumn("version").Error; err != nil {
				return err
			}
			if err := tx.Table("connectors").DropColumn("target_kind").Error; err != nil {
				return err
			}
			if err := tx.Table("connectors").DropColumn("addon_group").Error; err != nil {
				return err
			}

			if err := tx.DropTable(&ConnectorClusters{}).Error; err != nil {
				return err
			}
			return nil
		},
	}
}
