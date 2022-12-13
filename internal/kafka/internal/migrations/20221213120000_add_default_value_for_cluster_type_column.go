package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addDefaultValueForClusterTypeColumn() *gormigrate.Migration {

	return &gormigrate.Migration{
		ID: "20221213120000",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.Exec("ALTER TABLE clusters ALTER cluster_type SET DEFAULT 'managed'").Error; err != nil {
				return err
			}

			if err := tx.Unscoped().Exec("UPDATE clusters set cluster_type='managed'").Error; err != nil {
				return err
			}

			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Exec("ALTER TABLE clusters ALTER cluster_type DROP DEFAULT").Error
		},
	}
}
