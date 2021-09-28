package migrations

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addKafkaFailedWorkerLease() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20210928130000",
		Migrate: func(tx *gorm.DB) error {
			return tx.Create(&api.LeaderLease{Expires: &db.KafkaAdditionalLeasesExpireTime, LeaseType: "failed_kafka", Leader: api.NewID()}).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Unscoped().Where("lease_type = ?", "failed_kafka").Delete(&api.LeaderLease{}).Error
		},
	}
}
