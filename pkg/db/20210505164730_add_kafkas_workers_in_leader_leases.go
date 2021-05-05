package db

import (
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// Set new additional leases expire time to a minute later from now so that the old "kafka" leases finishes
// its execution before the new jobs kicks in.
var KafkaAdditionalLeasesExpireTime = time.Now().Add(1 * time.Minute)
var additionalLeaderLeaseTypes = []string{"accepted_kafka", "preparing_kafka", "provisioning_kafka", "deleting_kaka", "ready_kafka", "general_kafka_worker"}

const oldKafkaLeaseType string = "kafka"

func addKafkaWorkersInLeaderLeases() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20210505164730",
		Migrate: func(tx *gorm.DB) error {
			for _, leaderLeaseType := range additionalLeaderLeaseTypes {
				if err := tx.Create(&api.LeaderLease{Expires: &KafkaAdditionalLeasesExpireTime, LeaseType: leaderLeaseType, Leader: api.NewID()}).Error; err != nil {
					return err
				}
			}

			return tx.Unscoped().Where("lease_type = ?", oldKafkaLeaseType).Delete(&api.LeaderLease{}).Error
		},
		Rollback: func(tx *gorm.DB) error {
			leaseExpireTime := time.Now().Add(-time.Minute) // set to an expired time
			if err := tx.Create(&api.LeaderLease{Expires: &leaseExpireTime, LeaseType: oldKafkaLeaseType}).Error; err != nil {
				return err
			}

			return tx.Unscoped().Where("lease_type IN (?)", additionalLeaderLeaseTypes).Delete(&api.LeaderLease{}).Error
		},
	}
}
