package db

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func connectorMigrations20210518() *gormigrate.Migration {

	actions := []func(tx *gorm.DB, do bool) error{

		func(tx *gorm.DB, do bool) error {
			type Connector struct {
				Channel string
			}
			if do {
				return tx.AutoMigrate(&Connector{})
			} else {
				return tx.Migrator().DropColumn(&Connector{}, "channel")
			}
		},

		func(tx *gorm.DB, do bool) error {
			type ConnectorDeployment struct {
				ConnectorTypeService string
			}
			if do {
				return tx.Migrator().DropColumn(&ConnectorDeployment{}, "connector_type_service")
			} else {
				return tx.AutoMigrate(&ConnectorDeployment{})
			}
		},

		func(tx *gorm.DB, do bool) error {
			type ConnectorDeployment struct {
				AllowUpgrade bool
			}
			if do {
				return tx.AutoMigrate(&ConnectorDeployment{})
			} else {
				return tx.Migrator().DropColumn(&ConnectorDeployment{}, "allow_upgrade")
			}
		},

		func(tx *gorm.DB, do bool) error {
			type ConnectorDeployment struct {
				SpecChecksum string
			}
			if do {
				return tx.Migrator().DropColumn(&ConnectorDeployment{}, "spec_checksum")
			} else {
				return tx.AutoMigrate(&ConnectorDeployment{})
			}
		},

		func(tx *gorm.DB, do bool) error {
			type ConnectorDeployment struct {
				ConnectorVersion int64
			}
			if do {
				return tx.AutoMigrate(&ConnectorDeployment{})
			} else {
				return tx.Migrator().DropColumn(&ConnectorDeployment{}, "connector_version")
			}
		},

		func(tx *gorm.DB, do bool) error {
			type ConnectorDeploymentStatus struct {
				SpecChecksum string
			}
			if do {
				return tx.Migrator().DropColumn(&ConnectorDeploymentStatus{}, "spec_checksum")
			} else {
				return tx.AutoMigrate(&ConnectorDeploymentStatus{})
			}
		},

		func(tx *gorm.DB, do bool) error {
			type ConnectorDeploymentStatus struct {
				Version int64
			}
			if do {
				return tx.AutoMigrate(&ConnectorDeploymentStatus{})
			} else {
				return tx.Migrator().DropColumn(&ConnectorDeploymentStatus{}, "version")
			}
		},

		func(tx *gorm.DB, do bool) error {
			type ConnectorDeploymentStatus struct {
				AvailableUpgrades string
			}
			if do {
				return tx.AutoMigrate(&ConnectorDeploymentStatus{})
			} else {
				return tx.Migrator().DropColumn(&ConnectorDeploymentStatus{}, "available_upgrades")
			}
		},
	}

	return &gormigrate.Migration{
		ID: "20210518000000",
		Migrate: func(tx *gorm.DB) error {
			for _, action := range actions {
				err := action(tx, true)
				if err != nil {
					return err
				}
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			for i := len(actions) - 1; i >= 0; i-- {
				err := actions[i](tx, false)
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
}
