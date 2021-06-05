package db

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/go-gormigrate/gormigrate/v2"
)

func addConnectorTypeChannel() *gormigrate.Migration {
	type ConnectorShardMetadata struct {
		ID              int64  `gorm:"primaryKey;autoIncrement"`
		ConnectorTypeId string `gorm:"primaryKey"`
		Channel         string `gorm:"primaryKey"`
		ShardMetadata   string `gorm:"type:jsonb"`
		LatestId        *int64
	}

	type ConnectorDeployment struct {
		ConnectorTypeChannelId int64
	}

	type ConnectorDeploymentStatus struct {
		Operators        string `gorm:"type:jsonb"`
		UpgradeAvailable bool
	}

	return CreateMigrationFromActions("20210604000000",
		CreateTableAction(&ConnectorShardMetadata{}),
		AddTableColumnsAction(&ConnectorDeployment{}),
		AddTableColumnsAction(&ConnectorDeploymentStatus{}),
		DropTableColumnsAction(&struct {
			AvailableUpgrades string
		}{}, "connector_deployment_statuses"),
	)
}
