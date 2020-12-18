package db

import (
	"github.com/jinzhu/gorm"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
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
