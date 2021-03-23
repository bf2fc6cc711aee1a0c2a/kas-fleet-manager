package db

import (
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
)

func addKafkaSsoClientIdAndSecret() *gormigrate.Migration {
	type KafkaRequest struct {
		SsoClientID     string `json:"sso_client_id"`
		SsoClientSecret string `json:"sso_client_secret"`
	}
	return &gormigrate.Migration{
		ID: "20210322131730",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&KafkaRequest{}).Error; err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.Table("kafka_requests").DropColumn("sso_client_id").Error; err != nil {
				return err
			}
			if err := tx.Table("kafka_requests").DropColumn("sso_client_secret").Error; err != nil {
				return err
			}
			return nil
		},
	}
}
