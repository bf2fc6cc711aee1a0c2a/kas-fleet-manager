package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func renameDinosaurVersionField() *gormigrate.Migration {
	const oldDinosaurVersionFieldName string = "version"
	const newDinosaurActualVersionFieldName string = "actual_dinosaur_version"

	type DinosaurRequest struct {
		Version            string `json:"version"`
		ActualDinosaurVersion string `json:"actual_dinosaur_version"`
	}

	return &gormigrate.Migration{
		ID: "20210713135700",
		Migrate: func(tx *gorm.DB) error {
			if !tx.Migrator().HasColumn(DinosaurRequest{}, newDinosaurActualVersionFieldName) {
				return tx.Migrator().RenameColumn(DinosaurRequest{}, oldDinosaurVersionFieldName, newDinosaurActualVersionFieldName)
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			if !tx.Migrator().HasColumn(DinosaurRequest{}, oldDinosaurVersionFieldName) {
				return tx.Migrator().RenameColumn(DinosaurRequest{}, newDinosaurActualVersionFieldName, oldDinosaurVersionFieldName)
			}
			return nil
		},
	}
}
