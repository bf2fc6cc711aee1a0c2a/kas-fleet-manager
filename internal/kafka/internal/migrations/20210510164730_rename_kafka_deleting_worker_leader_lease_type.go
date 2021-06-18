package migrations

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

const oldDeletingKafkaLeaseType string = "deleting_kaka"
const newDeletingKafkaLeaseType string = "deleting_kafka"

func renameDeletingKafkaLeaseType() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20210510164730",
		Migrate: func(tx *gorm.DB) error {
			// expire time set to a a minute later
			if err := tx.Create(&api.LeaderLease{Expires: &db.KafkaAdditionalLeasesExpireTime, LeaseType: newDeletingKafkaLeaseType, Leader: api.NewID()}).Error; err != nil {
				return err
			}

			return tx.Unscoped().Where("lease_type = ?", oldDeletingKafkaLeaseType).Delete(&api.LeaderLease{}).Error
		},
		Rollback: func(tx *gorm.DB) error {
			leaseExpireTime := time.Now().Add(-time.Minute) // set to an expired time
			if err := tx.Create(&api.LeaderLease{Expires: &leaseExpireTime, LeaseType: oldDeletingKafkaLeaseType}).Error; err != nil {
				return err
			}

			return tx.Unscoped().Where("lease_type = ?", newDeletingKafkaLeaseType).Delete(&api.LeaderLease{}).Error
		},
	}
}
