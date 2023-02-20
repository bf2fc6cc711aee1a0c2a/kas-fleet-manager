package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

const (
	kafkaRoutesBaseDomainNameColumn       = "kafkas_routes_base_domain_name"
	kafkasRoutesBaseDomainTLSCrtRefColumn = "kafkas_routes_base_domain_tls_crt_ref"
	kafkasRoutesBaseDomainTLSKeyRefColumn = "kafkas_routes_base_domain_tls_key_ref"
)

func addKafkaDomainCertificateManagementInfoInKafkaRequestsTable() *gormigrate.Migration {
	type KafkaRequest struct {
		KafkasRoutesBaseDomainName      string
		KafkasRoutesBaseDomainTLSKeyRef string
		KafkasRoutesBaseDomainTLSCrtRef string
	}

	return &gormigrate.Migration{
		ID: "20230201140000",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&KafkaRequest{})
		},
		Rollback: func(tx *gorm.DB) error {
			if tx.Migrator().HasColumn(&KafkaRequest{}, kafkaRoutesBaseDomainNameColumn) {
				err := tx.Migrator().DropColumn(&KafkaRequest{}, kafkaRoutesBaseDomainNameColumn)
				if err != nil {
					return err
				}
			}

			if tx.Migrator().HasColumn(&KafkaRequest{}, kafkasRoutesBaseDomainTLSCrtRefColumn) {
				err := tx.Migrator().DropColumn(&KafkaRequest{}, kafkasRoutesBaseDomainTLSCrtRefColumn)
				if err != nil {
					return err
				}
			}

			if tx.Migrator().HasColumn(&KafkaRequest{}, kafkasRoutesBaseDomainTLSKeyRefColumn) {
				err := tx.Migrator().DropColumn(&KafkaRequest{}, kafkasRoutesBaseDomainTLSKeyRefColumn)
				if err != nil {
					return err
				}
			}

			return nil
		},
	}
}
