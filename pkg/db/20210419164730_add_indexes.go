package db

import (
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
)

func addMissingIndexes() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20210419164730",
		Migrate: func(tx *gorm.DB) error {
			db := tx.Table("kafka_requests")
			db = db.AddIndex("idx_kafka_requests_status", "status")
			db = db.AddIndex("idx_kafka_requests_cluster_id", "cluster_id")
			db = db.AddIndex("idx_kafka_requests_owner", "owner")
			db = db.AddIndex("idx_kafka_requests_organisation_id", "organisation_id")
			if err := db.Error; err != nil {
				return err
			}
			// in case there are duplicated entries in the database that will prevent the unique index from being created
			if err := tx.Table("clusters").Unscoped().Exec("DELETE from clusters WHERE deleted_at IS NOT NULL").Error; err != nil {
				return err
			}
			// add the new index
			if err := tx.Table("clusters").AddUniqueIndex("uix_clusters_cluster_id", "cluster_id").Error; err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			db := tx.Table("kafka_requests")
			db = db.RemoveIndex("idx_kafka_requests_status")
			db = db.RemoveIndex("idx_kafka_requests_cluster_id")
			db = db.RemoveIndex("idx_kafka_requests_owner")
			db = db.RemoveIndex("idx_kafka_requests_organisation_id")
			if err := db.Error; err != nil {
				return err
			}
			if err := tx.Table("clusters").RemoveIndex("uix_clusters_cluster_id").Error; err != nil {
				return err
			}
			return nil
		},
	}
}
