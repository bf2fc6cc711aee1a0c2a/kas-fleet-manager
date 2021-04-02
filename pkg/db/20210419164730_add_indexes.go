package db

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addMissingIndexes() *gormigrate.Migration {
	type Cluster struct {
		ClusterID string `json:"cluster_id" gorm:"uniqueIndex:uix_clusters_cluster_id"`
	}

	type KafkaRequest struct {
		Status         string `gorm:"index:idx_kafka_requests_status"`
		Owner          string `gorm:"index:idx_kafka_requests_owner"`
		ClusterID      string `gorm:"index:idx_kafka_requests_cluster_id"`
		OrganisationId string `gorm:"index:idx_kafka_requests_organisation_id"`
	}

	return &gormigrate.Migration{
		ID: "20210419164730",
		Migrate: func(tx *gorm.DB) error {
			migrator := tx.Migrator()
			kafka := &KafkaRequest{}
			err := migrator.CreateIndex(kafka, "Status")
			if err != nil {
				return err
			}
			err = migrator.CreateIndex(kafka, "ClusterID")
			if err != nil {
				return err
			}
			err = migrator.CreateIndex(kafka, "Owner")
			if err != nil {
				return err
			}
			err = migrator.CreateIndex(kafka, "OrganisationId")
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
			kafka := &KafkaRequest{}
			err := migrator.DropIndex(kafka, "Status")
			if err != nil {
				return err
			}
			err = migrator.DropIndex(kafka, "ClusterID")
			if err != nil {
				return err
			}
			err = migrator.DropIndex(kafka, "Owner")
			if err != nil {
				return err
			}
			err = migrator.DropIndex(kafka, "OrganisationId")
			if err != nil {
				return err
			}
			return migrator.DropIndex(&Cluster{}, "ClusterID")
		},
	}
}
