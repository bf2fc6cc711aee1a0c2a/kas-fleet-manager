package migrations

import (
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/db"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

var additionalLeaderLeaseTypes = []string{"accepted_dinosaur", "preparing_dinosaur", "provisioning_dinosaur", "deleting_kaka", "ready_dinosaur", "general_dinosaur_worker"}

const oldDinosaurLeaseType string = "dinosaur"

func addDinosaurWorkersInLeaderLeases() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20210505164730",
		Migrate: func(tx *gorm.DB) error {
			for _, leaderLeaseType := range additionalLeaderLeaseTypes {
				if err := tx.Create(&api.LeaderLease{Expires: &db.DinosaurAdditionalLeasesExpireTime, LeaseType: leaderLeaseType, Leader: api.NewID()}).Error; err != nil {
					return err
				}
			}

			return tx.Unscoped().Where("lease_type = ?", oldDinosaurLeaseType).Delete(&api.LeaderLease{}).Error
		},
		Rollback: func(tx *gorm.DB) error {
			leaseExpireTime := time.Now().Add(-time.Minute) // set to an expired time
			if err := tx.Create(&api.LeaderLease{Expires: &leaseExpireTime, LeaseType: oldDinosaurLeaseType}).Error; err != nil {
				return err
			}

			return tx.Unscoped().Where("lease_type IN (?)", additionalLeaderLeaseTypes).Delete(&api.LeaderLease{}).Error
		},
	}
}
