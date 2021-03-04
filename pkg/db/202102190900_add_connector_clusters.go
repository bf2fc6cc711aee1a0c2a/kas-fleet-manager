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
		Owner          string
		OrganisationId string
		Name           string
		AddonGroup     string
		Status         string
	}

	type Connectors struct {
		Version        int64 `gorm:"type:bigserial;index:"`
		TargetKind     string
		AddonGroup     string
		OrganisationId string
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
			if err := tx.Exec(`
                CREATE FUNCTION connectors_version_trigger() RETURNS TRIGGER LANGUAGE plpgsql AS '
					BEGIN
					NEW.version := nextval(''connectors_version_seq'');
					RETURN NEW;
					END;
				'
			`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`
				CREATE TRIGGER connectors_version_trigger BEFORE INSERT OR UPDATE ON connectors
				FOR EACH ROW EXECUTE PROCEDURE connectors_version_trigger();
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
			if err := tx.Where("lease_type = ?", "connector").Delete(&api.LeaderLease{}).Error; err != nil {
				return err
			}
			if err := tx.Exec(`DROP TRIGGER connectors_version_trigger ON connectors`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`DROP FUNCTION connectors_version_trigger`).Error; err != nil {
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
			if err := tx.Table("connectors").DropColumn("organisation_id").Error; err != nil {
				return err
			}

			if err := tx.DropTable(&ConnectorClusters{}).Error; err != nil {
				return err
			}
			return nil
		},
	}
}
