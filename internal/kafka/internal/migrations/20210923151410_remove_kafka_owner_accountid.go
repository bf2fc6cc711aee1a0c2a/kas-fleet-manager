package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func removeKafkaOwnerAccountId() *gormigrate.Migration {
	type KafkaRequest struct {
		OwnerAccountId string `json:"owner_account_id"`
	}
	return &gormigrate.Migration{
		ID: "20210923151410",
		Migrate: func(tx *gorm.DB) error {
			return tx.Migrator().DropColumn(&KafkaRequest{}, "owner_account_id")
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&KafkaRequest{})
		},
	}
}
