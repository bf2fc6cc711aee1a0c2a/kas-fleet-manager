package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addMissingIndexes() *gormigrate.Migration {
	type Cluster struct {
		ClusterID string `json:"cluster_id" gorm:"uniqueIndex:uix_clusters_cluster_id"`
	}

	type DinosaurRequest struct {
		Status         string `gorm:"index:idx_dinosaur_requests_status"`
		Owner          string `gorm:"index:idx_dinosaur_requests_owner"`
		ClusterID      string `gorm:"index:idx_dinosaur_requests_cluster_id"`
		OrganisationId string `gorm:"index:idx_dinosaur_requests_organisation_id"`
	}

	return &gormigrate.Migration{
		ID: "20210419164730",
		Migrate: func(tx *gorm.DB) error {
			migrator := tx.Migrator()
			dinosaur := &DinosaurRequest{}
			err := migrator.CreateIndex(dinosaur, "Status")
			if err != nil {
				return err
			}
			err = migrator.CreateIndex(dinosaur, "ClusterID")
			if err != nil {
				return err
			}
			err = migrator.CreateIndex(dinosaur, "Owner")
			if err != nil {
				return err
			}
			err = migrator.CreateIndex(dinosaur, "OrganisationId")
			if err != nil {
				return err
			}
			// in case there are duplicated entries in the database that will prevent the unique index from being created
			if err := tx.Table("clusters").Unscoped().Exec("DELETE from clusters WHERE deleted_at IS NOT NULL").Error; err != nil {
				return err
			}
			// add the new index
			return migrator.CreateIndex(&Cluster{}, "ClusterID")
		},
		Rollback: func(tx *gorm.DB) error {
			migrator := tx.Migrator()
			dinosaur := &DinosaurRequest{}
			err := migrator.DropIndex(dinosaur, "Status")
			if err != nil {
				return err
			}
			err = migrator.DropIndex(dinosaur, "ClusterID")
			if err != nil {
				return err
			}
			err = migrator.DropIndex(dinosaur, "Owner")
			if err != nil {
				return err
			}
			err = migrator.DropIndex(dinosaur, "OrganisationId")
			if err != nil {
				return err
			}
			return migrator.DropIndex(&Cluster{}, "ClusterID")
		},
	}
}
