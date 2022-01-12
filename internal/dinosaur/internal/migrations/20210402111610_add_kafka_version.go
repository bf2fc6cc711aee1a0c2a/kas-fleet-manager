package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addDinosaurVersion() *gormigrate.Migration {
	type DinosaurRequest struct {
		Version string `json:"version" gorm:"default:''"`
	}
	return &gormigrate.Migration{
		ID: "20210402111610",
		Migrate: func(tx *gorm.DB) error {
			err := tx.AutoMigrate(&DinosaurRequest{})
			if err != nil {
				return err
			}
			if err := tx.Table("dinosaur_requests").Where("version = ?", "").Update("version", "2.6.0").Error; err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropColumn(&DinosaurRequest{}, "version")
		},
	}
}
