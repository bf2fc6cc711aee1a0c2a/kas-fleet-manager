package migrations

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addDinosaurDNSWorkerLease() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20210628150600",
		Migrate: func(tx *gorm.DB) error {
			return tx.Create(&api.LeaderLease{Expires: &db.DinosaurAdditionalLeasesExpireTime, LeaseType: "dinosaur_dns", Leader: api.NewID()}).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Unscoped().Where("lease_type = ?", "dinosaur_dns").Delete(&api.LeaderLease{}).Error
		},
	}
}
