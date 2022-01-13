package migrations

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/db"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addDinosaurFailedWorkerLease() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20210928130000",
		Migrate: func(tx *gorm.DB) error {
			return tx.Create(&api.LeaderLease{Expires: &db.DinosaurAdditionalLeasesExpireTime, LeaseType: "failed_dinosaur", Leader: api.NewID()}).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Unscoped().Where("lease_type = ?", "failed_dinosaur").Delete(&api.LeaderLease{}).Error
		},
	}
}
