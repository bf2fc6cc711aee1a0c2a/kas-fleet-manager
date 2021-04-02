package db

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addClusterStatusIndex() *gormigrate.Migration {
	type Cluster struct {
		Status string `gorm:"index"`
	}

	return &gormigrate.Migration{
		ID: "20210420164730",
		Migrate: func(tx *gorm.DB) error {
			return tx.Migrator().CreateIndex(&Cluster{}, "Status")
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropIndex(&Cluster{}, "Status")
		},
	}
}
