package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addRoutes() *gormigrate.Migration {
	type KafkaRequest struct {
		Routes        string `gorm:"type:jsonb"`
		RoutesCreated bool   `gorm:"default:false"`
	}

	return &gormigrate.Migration{
		ID: "20210624120900",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&KafkaRequest{}); err != nil {
				return err
			}
			return tx.Unscoped().Exec("UPDATE kafka_requests SET routes_created=true WHERE status='ready';").Error
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.Migrator().DropColumn(&KafkaRequest{}, "routes"); err != nil {
				return err
			}
			return tx.Migrator().DropColumn(&KafkaRequest{}, "routes_created")
		},
	}
}
