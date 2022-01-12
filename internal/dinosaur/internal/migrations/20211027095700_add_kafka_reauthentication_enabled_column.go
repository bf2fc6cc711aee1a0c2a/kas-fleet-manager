package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addDinosaurReauthenticationEnabledColumn() *gormigrate.Migration {
	type DinosaurRequest struct {
		ReauthenticationEnabled bool `json:"reauthentication_enabled" gorm:"default:true"`
	}

	return &gormigrate.Migration{
		ID: "20211027095700",
		Migrate: func(tx *gorm.DB) error {
			err := tx.AutoMigrate(&DinosaurRequest{})
			if err != nil {
				return err
			}

			return tx.Exec("update dinosaur_requests set reauthentication_enabled = true;").Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropColumn(&DinosaurRequest{}, "reauthentication_enabled")
		},
	}
}
