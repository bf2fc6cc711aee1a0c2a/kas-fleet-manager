package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addClusterOrgIdClusterTypeColumns() *gormigrate.Migration {
	type Cluster struct {
		ClusterType    string `json:"cluster_type"`
		OrganizationID string `json:"organization_id"`
	}

	return &gormigrate.Migration{
		ID: "20221202120000",
		Migrate: func(tx *gorm.DB) error {
			err := tx.AutoMigrate(&Cluster{})
			if err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			err := tx.Migrator().DropColumn(&Cluster{}, "cluster_type")
			if err != nil {
				return err
			}

			err = tx.Migrator().DropColumn(&Cluster{}, "organization_id")
			if err != nil {
				return err
			}
			return nil
		},
	}
}
