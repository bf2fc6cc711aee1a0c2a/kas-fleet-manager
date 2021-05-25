package db

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addKafkaConnectionSettingsToConnectors() *gormigrate.Migration {

	actions := []func(tx *gorm.DB, do bool) error{

		func(tx *gorm.DB, do bool) error {
			type KafkaConnectionSettings struct {
				BootstrapServer string `json:"bootstrap_server,omitempty"`
				ClientId        string `json:"client_id,omitempty"`
				ClientSecret    string `json:"client_secret,omitempty"`
			}
			type Connector struct {
				Kafka KafkaConnectionSettings `gorm:"embedded;embeddedPrefix:kafka_"`
			}

			if do {
				err := tx.AutoMigrate(&Connector{})
				if err != nil {
					return err
				}
				return nil
			} else {
				err := tx.Migrator().DropColumn(&Connector{}, "kafka_bootstrap_server")
				if err != nil {
					return err
				}
				err = tx.Migrator().DropColumn(&Connector{}, "kafka_client_id")
				if err != nil {
					return err
				}
				err = tx.Migrator().DropColumn(&Connector{}, "kafka_client_secret")
				if err != nil {
					return err
				}
				return nil
			}
		},
	}

	return &gormigrate.Migration{
		ID: "20210524000000",
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
