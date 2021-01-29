package db

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
	"time"
)

type LeaderLease struct {
	Model
	Leader    string
	LeaseType string
	Expires   *time.Time
}

func addLeaderLease() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202012031820",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(
				&LeaderLease{},
			).Error; err != nil {
				return err
			}
			// pre-seed a single empty leader lease for each type of workers that leader election mgr can begin attempting to claim
			now := time.Now().Add(-time.Minute) //set to a expired time
			if err := tx.Create(&api.LeaderLease{
				Expires:   &now,
				LeaseType: "cluster",
			}).Error; err != nil {
				return err
			}
			if err := tx.Create(&api.LeaderLease{
				Expires:   &now,
				LeaseType: "kafka",
			}).Error; err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.DropTable(
				&LeaderLease{},
			).Error; err != nil {
				return err
			}
			return nil
		},
	}
}
