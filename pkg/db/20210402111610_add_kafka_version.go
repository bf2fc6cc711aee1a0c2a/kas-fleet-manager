package db

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addKafkaVersion() *gormigrate.Migration {
	type KafkaRequest struct {
		Version string `json:"version" gorm:"default:''"`
	}
	return &gormigrate.Migration{
		ID: "20210402111610",
		Migrate: func(tx *gorm.DB) error {
			err := tx.AutoMigrate(&KafkaRequest{})
			if err != nil {
				return err
			}
			if err := tx.Table("kafka_requests").Where("version = ?", "").Update("version", "2.6.0").Error; err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropColumn(&KafkaRequest{}, "version")
		},
	}
}
