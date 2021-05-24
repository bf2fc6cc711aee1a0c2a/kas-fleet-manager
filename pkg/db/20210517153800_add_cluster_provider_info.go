package db

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addClusterProviderInfo() *gormigrate.Migration {
	type Cluster struct {
		ProviderType string
		ProviderSpec string
		ClusterSpec  string
	}
	return &gormigrate.Migration{
		ID: "20210517153800",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&Cluster{}); err != nil {
				return err
			}
			if err := tx.Table("clusters").Where("provider_type = ?", "").Update("provider_type", "ocm").Error; err != nil {
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
			if err := tx.Migrator().DropColumn(&Cluster{}, "provider_type"); err != nil {
				return err
			}
			if err := tx.Migrator().DropColumn(&Cluster{}, "cluster_spec"); err != nil {
				return err
			}
			if err := tx.Migrator().DropColumn(&Cluster{}, "provider_spec"); err != nil {
				return err
			}
			type OriginalCluster struct {
				BYOC    bool
				Managed bool
			}
			if err := tx.AutoMigrate(&OriginalCluster{}); err != nil {
				return err
			}
			return nil
		},
	}
}
