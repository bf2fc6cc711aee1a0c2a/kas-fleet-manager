package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

const accessKafkasViaPrivateNetworkColumnName = "access_kafkas_via_private_network"

func addAccessKafkasViaPrivateNetworkColumnInClustersTable() *gormigrate.Migration {
	type Cluster struct {
		AccessKafkasViaPrivateNetwork bool `gorm:"default:false"`
	}

	return &gormigrate.Migration{
		ID: "20230111120000",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&Cluster{})
		},
		Rollback: func(tx *gorm.DB) error {
			if !tx.Migrator().HasColumn(&Cluster{}, accessKafkasViaPrivateNetworkColumnName) {
				return nil
			}

			return tx.Migrator().DropColumn(&Cluster{}, accessKafkasViaPrivateNetworkColumnName)
		},
	}
}
