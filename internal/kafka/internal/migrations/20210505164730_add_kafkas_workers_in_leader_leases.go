package migrations

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

var additionalLeaderLeaseTypes = []string{"accepted_kafka", "preparing_kafka", "provisioning_kafka", "deleting_kaka", "ready_kafka", "general_kafka_worker"}

const oldKafkaLeaseType string = "kafka"

func addKafkaWorkersInLeaderLeases() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20210505164730",
		Migrate: func(tx *gorm.DB) error {
			for _, leaderLeaseType := range additionalLeaderLeaseTypes {
				if err := tx.Create(&api.LeaderLease{Expires: &db.KafkaAdditionalLeasesExpireTime, LeaseType: leaderLeaseType, Leader: api.NewID()}).Error; err != nil {
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
