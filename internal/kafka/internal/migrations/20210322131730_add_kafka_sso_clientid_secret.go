package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addKafkaSsoClientIdAndSecret() *gormigrate.Migration {
	type KafkaRequest struct {
		SsoClientID     string `json:"sso_client_id"`
		SsoClientSecret string `json:"sso_client_secret"`
	}
	return &gormigrate.Migration{
		ID: "20210322131730",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&KafkaRequest{})
		},
		Rollback: func(tx *gorm.DB) error {
			err := tx.Migrator().DropColumn(&KafkaRequest{}, "sso_client_id")
			if err != nil {
				return err
			}
			return tx.Migrator().DropColumn(&KafkaRequest{}, "sso_client_secret")
		},
	}
}
