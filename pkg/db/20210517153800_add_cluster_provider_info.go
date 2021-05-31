package db

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addClusterProviderInfo() *gormigrate.Migration {

	return &gormigrate.Migration{
		ID: "20210517153800",
		Migrate: func(tx *gorm.DB) error {
			type Cluster struct {
				ProviderType string
				ProviderSpec string
				ClusterSpec  string
			}
			if err := tx.AutoMigrate(&Cluster{}); err != nil {
				return err
			}
			if err := tx.Exec(`UPDATE clusters SET provider_type = 'ocm' WHERE provider_type IS NULL`).Error; err != nil {
				return err
			}
			if err := tx.Migrator().DropColumn(&Cluster{}, "byoc"); err != nil {
				return err
			}
			if err := tx.Migrator().DropColumn(&Cluster{}, "managed"); err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			type Cluster struct {
				BYOC    bool
				Managed bool
			}
			if err := tx.Migrator().DropColumn(&Cluster{}, "provider_type"); err != nil {
				return err
			}
			if err := tx.Migrator().DropColumn(&Cluster{}, "cluster_spec"); err != nil {
				return err
			}
			if err := tx.Migrator().DropColumn(&Cluster{}, "provider_spec"); err != nil {
				return err
			}

			if err := tx.AutoMigrate(&Cluster{}); err != nil {
				return err
			}
			return nil
		},
	}
}
