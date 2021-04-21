package db

import (
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
)

func addClusterStatusIndex() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20210420164730",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.Table("clusters").AddIndex("idx_clusters_status", "status").Error; err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.Table("clusters").RemoveIndex("idx_clusters_status").Error; err != nil {
				return err
			}
			return nil
		},
	}
}
