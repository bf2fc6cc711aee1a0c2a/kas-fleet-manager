package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addDinosaurUpgradeFunctionalityRelatedFields() *gormigrate.Migration {
	type DinosaurRequest struct {
		DesiredDinosaurVersion   string `json:"desired_dinosaur_version"`
		DesiredStrimziVersion string `json:"desired_strimzi_version"`
		ActualStrimziVersion  string `json:"actual_strimzi_version"`
		DinosaurUpgrading        bool   `json:"dinosaur_upgrading" gorm:"default:false"`
		StrimziUpgrading      bool   `json:"strimzi_upgrading" gorm:"default:false"`
	}

	return &gormigrate.Migration{
		ID: "20210713135600",
		Migrate: func(tx *gorm.DB) error {
			err := tx.AutoMigrate(&DinosaurRequest{})
			if err != nil {
				return err
			}

			err = tx.Exec(`UPDATE dinosaur_requests SET desired_dinosaur_version = version WHERE desired_dinosaur_version IS NULL`).Error
			if err != nil {
				return err
			}

			strimziVersionToSetOnMigration := "strimzi-cluster-operator.v0.23.0-0"

			err = tx.Table("dinosaur_requests").Where("desired_strimzi_version IS NULL").Update("desired_strimzi_version", strimziVersionToSetOnMigration).Error
			if err != nil {
				return err
			}

			err = tx.Table("dinosaur_requests").Where("actual_strimzi_version IS NULL").Update("actual_strimzi_version", strimziVersionToSetOnMigration).Error
			if err != nil {
				return err
			}

			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			err := tx.Migrator().DropColumn(&DinosaurRequest{}, "desired_dinosaur_version")
			if err != nil {
				return err
			}

			err = tx.Migrator().DropColumn(&DinosaurRequest{}, "desired_strimzi_version")
			if err != nil {
				return err
			}

			err = tx.Migrator().DropColumn(&DinosaurRequest{}, "actual_strimzi_version")
			if err != nil {
				return err
			}

			err = tx.Migrator().DropColumn(&DinosaurRequest{}, "dinosaur_upgrading")
			if err != nil {
				return err
			}

			err = tx.Migrator().DropColumn(&DinosaurRequest{}, "strimzi_upgrading")
			if err != nil {
				return err
			}

			return nil
		},
	}
}
