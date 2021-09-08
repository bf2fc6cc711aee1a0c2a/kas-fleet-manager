package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addClusters() *gormigrate.Migration {
	// store enough information about the cluster that we can re-create it if needed
	type Cluster struct {
		db.Model
		CloudProvider string
		ClusterID     string `gorm:"index"`
		ExternalID    string
		MultiAZ       bool
		Region        string
		BYOC          bool
		Managed       bool
	}

	return &gormigrate.Migration{
		ID: "202009101015",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&Cluster{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&Cluster{})
		},
	}
}
