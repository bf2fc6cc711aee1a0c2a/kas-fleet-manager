package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func renameEvalKafkaInstanceType() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20220325111000",
		Migrate: func(tx *gorm.DB) error {
			return tx.Unscoped().Exec("UPDATE kafka_requests SET instance_type='developer' where instance_type='eval';").Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Unscoped().Exec("UPDATE kafka_requests SET instance_type='eval' where instance_type='developer';").Error
		},
	}
}
