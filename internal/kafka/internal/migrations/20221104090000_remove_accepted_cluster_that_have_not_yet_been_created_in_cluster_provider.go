package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func removeAcceptedClusterThatHaveNotYetBeenCreatedInClusterProvider() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20221104090000",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.Exec("DELETE FROM clusters where status='cluster_accepted' AND (cluster_id='' OR cluster_id IS NULL);").Error; err != nil {
				return err
			}

			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			// no need to rollback a one shot operation
			return nil
		},
	}
}
