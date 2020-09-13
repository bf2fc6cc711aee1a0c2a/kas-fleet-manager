package db

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
)

func addClusters() *gormigrate.Migration {
	// store enough information about the cluster that we can re-create it if needed
	type Cluster struct {
		Model
		CloudProvider string
		ClusterID     string `gorm:"index"`
		ExternalID    string
		MultiAZ       bool
		Region        string
		State         string
		BYOC          bool
		Managed       bool
	}

	return &gormigrate.Migration{
		ID: "202009101015",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&Cluster{}).Error; err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.DropTable(&Cluster{}).Error; err != nil {
				return err
			}
			return nil
		},
	}
}
