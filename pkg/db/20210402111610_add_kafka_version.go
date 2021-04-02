package db

import (
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
)

func addKafkaVersion() *gormigrate.Migration {
	type KafkaRequest struct {
		Version string `json:"version" gorm:"default:''"`
	}
	return &gormigrate.Migration{
		ID: "20210402111610",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&KafkaRequest{}).Error; err != nil {
				return err
			}
			if err := tx.Table("kafka_requests").Where("version = ?", "").Update("version", "2.6.0").Error; err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.Table("kafka_requests").DropColumn("version").Error; err != nil {
				return err
			}
			return nil
		},
	}
}
