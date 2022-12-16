package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func removeWronglyCreatedEnterpriseClusterInProd() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20221216120000",
		Migrate: func(tx *gorm.DB) error {
			// remove the cluster with cluster id=20l15s8hsji3190maa4ut40p5561k87q which was wrongly created by a enterprise cluster registration call
			if err := tx.Exec("DELETE FROM clusters where cluster_id='20l15s8hsji3190maa4ut40p5561k87q';").Error; err != nil {
				return err
			}

			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			// no need to rollback a one shot operation
			return nil
		},
	}
}
