package migrations

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

type LeaderLease struct {
	db.Model
	Leader    string
	LeaseType string
	Expires   *time.Time
}

func addLeaderLease() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202012031820",
		Migrate: func(tx *gorm.DB) error {
			var err error
			err = tx.AutoMigrate(&LeaderLease{})
			if err != nil {
				return err
			}
			// pre-seed a single empty leader lease for each type of workers that leader election mgr can begin attempting to claim
			now := time.Now().Add(-time.Minute) //set to a expired time
			if err = tx.Create(&api.LeaderLease{
				Expires:   &now,
				LeaseType: "cluster",
			}).Error; err != nil {
				return err
			}
			if err = tx.Create(&api.LeaderLease{
				Expires:   &now,
				LeaseType: "dinosaur",
			}).Error; err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&LeaderLease{})
		},
	}
}
