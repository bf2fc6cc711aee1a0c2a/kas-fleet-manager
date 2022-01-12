package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addDinosaurIBPVersionRelatedFields() *gormigrate.Migration {
	type DinosaurRequest struct {
		DesiredDinosaurIBPVersion string `json:"desired_dinosaur_ibp_version"`
		ActualDinosaurIBPVersion  string `json:"actual_dinosaur_ibp_version"`
		DinosaurIBPUpgrading      bool   `json:"dinosaur_ibp_upgrading" gorm:"default:false"`
	}

	return &gormigrate.Migration{
		ID: "20211112173900",
		Migrate: func(tx *gorm.DB) error {
			err := tx.AutoMigrate(&DinosaurRequest{})
			if err != nil {
				return err
			}

			dinosaurIBPVersionToSetOnMigration := "2.7"

			err = tx.Table("dinosaur_requests").Where("desired_dinosaur_ibp_version IS NULL").Update("desired_dinosaur_ibp_version", dinosaurIBPVersionToSetOnMigration).Error
			if err != nil {
				return err
			}

			err = tx.Table("dinosaur_requests").Where("actual_dinosaur_ibp_version IS NULL").Update("actual_dinosaur_ibp_version", dinosaurIBPVersionToSetOnMigration).Error
			if err != nil {
				return err
			}

			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			err := tx.Migrator().DropColumn(&DinosaurRequest{}, "desired_dinosaur_ibp_version")
			if err != nil {
				return err
			}

			err = tx.Migrator().DropColumn(&DinosaurRequest{}, "actual_dinosaur_ibp_version")
			if err != nil {
				return err
			}

			err = tx.Migrator().DropColumn(&DinosaurRequest{}, "dinosaur_ibp_upgrading")
			if err != nil {
				return err
			}

			return nil
		},
	}
}
