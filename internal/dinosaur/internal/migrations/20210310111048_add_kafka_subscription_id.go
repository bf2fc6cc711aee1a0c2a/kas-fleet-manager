package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addDinosaurSubscriptionId() *gormigrate.Migration {
	type DinosaurRequest struct {
		SubscriptionId string
	}
	return &gormigrate.Migration{
		ID: "20210310111048",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&DinosaurRequest{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropColumn(&DinosaurRequest{}, "subscription_id")
		},
	}
}
