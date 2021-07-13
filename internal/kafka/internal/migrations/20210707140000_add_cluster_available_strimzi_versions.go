package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addClusterAvailableStrimziVersions() *gormigrate.Migration {
	type Cluster struct {
		AvailableStrimziVersions string `json:"available_strimzi_versions" gorm:"type:jsonb"`
	}

	return &gormigrate.Migration{
		ID: "20210707140000",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&Cluster{}); err != nil {
				return err
			}

			// Store the field as a JSON array of strings. The reason for that is that
			// available_strimzi_versions is stored in the database as a jsonb data
			// type
			strimziVersionToSetOnMigration := `["strimzi-cluster-operator.v0.23.0-0"]`

			err := tx.Table("clusters").Where("available_strimzi_versions IS NULL").Update("available_strimzi_versions", strimziVersionToSetOnMigration).Error
			if err != nil {
				return nil
			}

			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropColumn(&Cluster{}, "available_strimzi_versions")
		},
	}
}
