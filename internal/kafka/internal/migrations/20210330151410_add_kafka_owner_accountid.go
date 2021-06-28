package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addKafkaOwnerAccountId() *gormigrate.Migration {
	type KafkaRequest struct {
		OwnerAccountId string `json:"owner_account_id"`
	}
	return &gormigrate.Migration{
		ID: "20210330151410",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&KafkaRequest{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropColumn(&KafkaRequest{}, "owner_account_id")
		},
	}
}
