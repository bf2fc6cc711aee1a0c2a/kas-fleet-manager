package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func removeTheWronglyAutoCreatedClusterInStageEnvironmentWithID_cdhunvd8igjhbi0nmtt0() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20221104090000",
		Migrate: func(tx *gorm.DB) error {
			// remove the cluster with internal id=cdhunvd8igjhbi0nmtt0 which was wrongly created by dynamic scaling in a Stage environment
			// which has a strong requirement of limiting the number of clusters.
			if err := tx.Exec("DELETE FROM clusters where status='cluster_accepted' AND id='cdhunvd8igjhbi0nmtt0' AND (cluster_id='' OR cluster_id IS NULL);").Error; err != nil {
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
