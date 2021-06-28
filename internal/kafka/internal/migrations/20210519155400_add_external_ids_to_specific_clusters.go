package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// addExternalIDsToSpecificClusters adds the external_id of the
// staging and production OCP clusters existing at the moment
// this migration was written
func addExternalIDsToSpecificClusters() *gormigrate.Migration {
	stagingClusterClusterID := "1ieq35mpf26sq2r246sbffuvg9s380sp"
	stagingClusterExternalID := "3e57a1d5-fc3e-4cca-b867-3008f2569f63"
	productionClusterClusterID := "1k5ot6e86c6itcqqsrs019bngqc7cjin"
	productionClusterExternalID := "28506554-159c-42ff-9d43-26315f6459dd"
	return &gormigrate.Migration{
		ID: "20210519155400",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.Table("clusters").Where("cluster_id = ?", stagingClusterClusterID).Update("external_id", stagingClusterExternalID).Error; err != nil {
				return err
			}

			if err := tx.Table("clusters").Where("cluster_id = ?", productionClusterClusterID).Update("external_id", productionClusterExternalID).Error; err != nil {
				return err
			}

			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.Table("clusters").Where("cluster_id = ?", stagingClusterClusterID).Update("external_id", "").Error; err != nil {
				return err
			}
			if err := tx.Table("clusters").Where("cluster_id = ?", productionClusterClusterID).Update("external_id", "").Error; err != nil {
				return err
			}

			return nil
		},
	}
}
