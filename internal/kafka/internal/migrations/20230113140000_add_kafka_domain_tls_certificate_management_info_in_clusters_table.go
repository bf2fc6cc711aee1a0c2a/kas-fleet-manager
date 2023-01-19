package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

const (
	baseKafkasDomainNameColumn      = "base_kafkas_domain_name"
	baseKafkasDomainTLSCrtRefColumn = "base_kafkas_domain_tls_crt_ref"
	baseKafkasDomainTLSKeyRefColumn = "base_kafkas_domain_tls_key_ref"
)

func addKafkaDomainCertificateManagementInfoInClustersTable() *gormigrate.Migration {
	type Cluster struct {
		BaseKafkasDomainName      string
		BaseKafkasDomainTLSCrtRef string
		BaseKafkasDomainTLSKeyRef string
	}

	return &gormigrate.Migration{
		ID: "20230113140000",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&Cluster{})
		},
		Rollback: func(tx *gorm.DB) error {
			if tx.Migrator().HasColumn(&Cluster{}, baseKafkasDomainNameColumn) {
				err := tx.Migrator().DropColumn(&Cluster{}, baseKafkasDomainNameColumn)
				if err != nil {
					return err
				}
			}

			if tx.Migrator().HasColumn(&Cluster{}, baseKafkasDomainTLSCrtRefColumn) {
				err := tx.Migrator().DropColumn(&Cluster{}, baseKafkasDomainTLSCrtRefColumn)
				if err != nil {
					return err
				}
			}

			if tx.Migrator().HasColumn(&Cluster{}, baseKafkasDomainTLSKeyRefColumn) {
				err := tx.Migrator().DropColumn(&Cluster{}, baseKafkasDomainTLSKeyRefColumn)
				if err != nil {
					return err
				}
			}

			return nil
		},
	}
}
