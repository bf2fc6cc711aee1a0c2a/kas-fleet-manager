package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addClusterDynamicCapacityInfo() *gormigrate.Migration {
	type Cluster struct {
		DynamicCapacityInfo string `json:"dynamic_capacity_info" gorm:"type:jsonb"`
	}

	return &gormigrate.Migration{
		ID: "20220622140000",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&Cluster{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropColumn(&Cluster{}, "dynamic_capacity_info")
		},
	}
}
