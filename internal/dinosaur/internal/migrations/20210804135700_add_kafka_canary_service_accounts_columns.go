package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addDinosaurCanaryServiceAccountColumns() *gormigrate.Migration {
	type DinosaurRequest struct {
		CanaryServiceAccountClientID     string `json:"canary_service_account_client_id"`
		CanaryServiceAccountClientSecret string `json:"canary_service_account_client_secret"`
	}

	return &gormigrate.Migration{
		ID: "20210804135700",
		Migrate: func(tx *gorm.DB) error {
			err := tx.AutoMigrate(&DinosaurRequest{})
			if err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			err := tx.Migrator().DropColumn(&DinosaurRequest{}, "canary_service_account_client_id")
			if err != nil {
				return err
			}

			err = tx.Migrator().DropColumn(&DinosaurRequest{}, "canary_service_account_client_secret")
			if err != nil {
				return err
			}
			return nil
		},
	}
}
