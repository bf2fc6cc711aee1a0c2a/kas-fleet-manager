package migrations

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addDeprovisioningClusterWorkerToLeaderLeases() *gormigrate.Migration {
	const deprovisioningClustersWorkerType = "deprovisioning_clusters"

	return &gormigrate.Migration{
		ID: "20220829190000",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.Create(&api.LeaderLease{Expires: &db.KafkaAdditionalLeasesExpireTime, LeaseType: deprovisioningClustersWorkerType, Leader: api.NewID()}).Error; err != nil {
				return err
			}

			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			err := tx.Unscoped().Where("lease_type = ?", deprovisioningClustersWorkerType).Delete(&api.LeaderLease{}).Error
			if err != nil {
				return err
			}
			return nil
		},
	}
}
