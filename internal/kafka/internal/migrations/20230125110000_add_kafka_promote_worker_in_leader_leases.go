package migrations

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addKafkaPromoteWorkerInLeaderLeases() *gormigrate.Migration {
	leaderLeaseType := "promoting_kafka"
	return &gormigrate.Migration{
		ID: "20230125110000",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.Create(&api.LeaderLease{Expires: &db.KafkaAdditionalLeasesExpireTime, LeaseType: leaderLeaseType, Leader: api.NewID()}).Error; err != nil {
				return err
			}

			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Unscoped().Where("lease_type = ?", leaderLeaseType).Delete(&api.LeaderLease{}).Error
		},
	}
}
