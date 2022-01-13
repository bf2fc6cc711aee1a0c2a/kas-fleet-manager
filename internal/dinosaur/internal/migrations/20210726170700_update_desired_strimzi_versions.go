package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func updateDesiredStrimziVersions() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20210726170700",
		Migrate: func(tx *gorm.DB) error {

			strimziVersionToSetOnMigration := "strimzi-cluster-operator.v0.23.0-0"

			err := tx.Table("dinosaur_requests").Where("desired_strimzi_version IS NULL OR desired_strimzi_version = ?", "").Update("desired_strimzi_version", strimziVersionToSetOnMigration).Error
			if err != nil {
				return err
			}

			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return nil
		},
	}
}
