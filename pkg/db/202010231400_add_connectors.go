package db

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addConnectors() *gormigrate.Migration {
	type Connectors struct {
		Model
		ConnectorTypeId string `json:"connector_type_id"`
		ConnectorSpec   string `json:"connector_spec"`
		Region          string `json:"region"`
		ClusterID       string `json:"cluster_id"`
		CloudProvider   string `json:"cloud_provider"`
		MultiAZ         bool   `json:"multi_az"`
		Name            string `json:"name"`
		Status          string `json:"status"`
		Owner           string `json:"owner"`
		KafkaID         string `json:"kafka_id"`
	}

	return &gormigrate.Migration{
		ID: "202011231400",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&Connectors{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&Connectors{})
		},
	}
}
