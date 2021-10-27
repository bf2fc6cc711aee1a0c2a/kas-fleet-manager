package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addKafkaReauthenticationEnabledColumn() *gormigrate.Migration {
	type KafkaRequest struct {
		ReauthenticationEnabled bool `json:"reauthentication_enabled" gorm:"default:true"`
	}

	return &gormigrate.Migration{
		ID: "20211027095700",
		Migrate: func(tx *gorm.DB) error {
			err := tx.AutoMigrate(&KafkaRequest{})
			if err != nil {
				return err
			}

			return tx.Exec("update kafka_requests set reauthentication_enabled = true;").Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropColumn(&KafkaRequest{}, "reauthentication_enabled")
		},
	}
}
