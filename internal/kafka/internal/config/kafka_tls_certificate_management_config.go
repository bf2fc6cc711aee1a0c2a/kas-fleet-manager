package config

import (
	"fmt"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
	"github.com/caddyserver/certmagic"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

const (
	InMemoryTLSCertStorageType     = "in-memory"
	SecureTLSCertStorageType       = "secure-storage"
	FileTLSCertStorageType         = "file"
	ManualCertificateManagement    = "manual"
	AutomaticCertificateManagement = "automatic"
)

var validStorageTypes = []string{InMemoryTLSCertStorageType, FileTLSCertStorageType, SecureTLSCertStorageType}
var validCertificateManagementStrategies = []string{ManualCertificateManagement, AutomaticCertificateManagement}

type KafkaTLSCertificateManagementConfig struct {
	CertificateAuthorityEndpoint         string
	CertificateManagementStrategy        string
	StorageType                          string
	EnableKafkaExternalCertificate       bool
	ManualCertificateManagementConfig    ManualCertificateManagementConfig
	AutomaticCertificateManagementConfig AutomaticCertificateManagementConfig
}

type ManualCertificateManagementConfig struct {
	KafkaTLSCert         string
	KafkaTLSKey          string
	KafkaTLSCertFilePath string `validate:"required"`
	KafkaTLSKeyFilePath  string `validate:"required"`
}

type AutomaticCertificateManagementConfig struct {
	RenewalWindowRatio           float64 `validate:"gte=0,lte=1"`
	EmailToSendNotificationTo    string  `validate:"required,email"`
	AcmeIssuerAccountKeyFilePath string  `validate:"required"`
	CertificateCacheTTL          time.Duration
	AcmeIssuerAccountKey         string
	MustStaple                   bool
}

func NewCertificateManagementConfig() *KafkaTLSCertificateManagementConfig {
	return &KafkaTLSCertificateManagementConfig{
		CertificateAuthorityEndpoint:   certmagic.LetsEncryptProductionCA,
		StorageType:                    InMemoryTLSCertStorageType,
		EnableKafkaExternalCertificate: false,
		ManualCertificateManagementConfig: ManualCertificateManagementConfig{
			KafkaTLSCertFilePath: "secrets/kafka-tls.crt",
			KafkaTLSKeyFilePath:  "secrets/kafka-tls.key",
		},
		CertificateManagementStrategy: ManualCertificateManagement,
		AutomaticCertificateManagementConfig: AutomaticCertificateManagementConfig{
			EmailToSendNotificationTo:    "",
			CertificateCacheTTL:          10 * time.Minute,
			RenewalWindowRatio:           certmagic.Default.RenewalWindowRatio,
			AcmeIssuerAccountKeyFilePath: "secrets/kafka-tls-certificate-management-acme-issuer-account-key.pem",
			MustStaple:                   false,
		},
	}
}

func (c *KafkaTLSCertificateManagementConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.CertificateManagementStrategy, "kafka-tls-certificate-management-strategy", c.CertificateManagementStrategy, "The strategy used to manage tls certificates: Supported values are 'manual', 'automatic'. The default value is 'manual'")
	fs.StringVar(&c.StorageType, "kafka-tls-certificate-management-storage-type", c.StorageType, "The storage type of the tls certificates: Supported values are 'in-memory', 'secure-storage', 'file'. The default value is 'in-memory'")
	fs.StringVar(&c.ManualCertificateManagementConfig.KafkaTLSCertFilePath, "kafka-tls-cert-file", c.ManualCertificateManagementConfig.KafkaTLSCertFilePath, "File containing kafka certificate")
	fs.StringVar(&c.ManualCertificateManagementConfig.KafkaTLSKeyFilePath, "kafka-tls-key-file", c.ManualCertificateManagementConfig.KafkaTLSKeyFilePath, "File containing kafka certificate private key")
	fs.BoolVar(&c.EnableKafkaExternalCertificate, "enable-kafka-external-certificate", c.EnableKafkaExternalCertificate, "Enable custom certificate for Kafka TLS")
	fs.BoolVar(&c.AutomaticCertificateManagementConfig.MustStaple, "kafka-tls-certificate-management-must-staple", c.AutomaticCertificateManagementConfig.MustStaple, "Adds the must staple TLS extension to the certificate signing request.")
	fs.StringVar(&c.AutomaticCertificateManagementConfig.EmailToSendNotificationTo, "kafka-tls-certificate-management-email", c.AutomaticCertificateManagementConfig.EmailToSendNotificationTo, "The email address that will receive certificate notification. The field is required")
	fs.StringVar(&c.AutomaticCertificateManagementConfig.AcmeIssuerAccountKeyFilePath, "kafka-tls-certificate-management-acme-issuer-account-key-file-path", c.AutomaticCertificateManagementConfig.AcmeIssuerAccountKeyFilePath, "The file containing the ACME Issuer account key. This is required")
	fs.Float64Var(&c.AutomaticCertificateManagementConfig.RenewalWindowRatio, "kafka-tls-certificate-management-renewal-window-ratio", c.AutomaticCertificateManagementConfig.RenewalWindowRatio, "How much of a certificate's lifetime becomes the renewal window")
	fs.DurationVar(&c.AutomaticCertificateManagementConfig.CertificateCacheTTL, "kafka-tls-certificate-management-secure-storage-cache-ttl", c.AutomaticCertificateManagementConfig.CertificateCacheTTL, "The cache duration of the certificate when secure-storage is used. Past this duration, the cached certificate will be refreshed from the secure storage on its retrieval")
}

func (c *KafkaTLSCertificateManagementConfig) ReadFiles() error {
	if c.CertificateManagementStrategy == AutomaticCertificateManagement {
		err := shared.ReadFileValueString(c.AutomaticCertificateManagementConfig.AcmeIssuerAccountKeyFilePath, &c.AutomaticCertificateManagementConfig.AcmeIssuerAccountKey)
		if err != nil {
			return err
		}
	}

	// We always read the manual tls certificates irrespective of strategy used.
	// This is done so that we can default to using the manually provided certificate during the transition from manual to automatic certificate handling.
	// That is, we want the transition to be a smooth one, without breaking the reconciliation loop between fleet manager and fleetshard sync
	// for Kafkas that whose certificates information has not been reconcilied internally (populated in the database) by the fleet manager.
	if c.EnableKafkaExternalCertificate {
		err := shared.ReadFileValueString(c.ManualCertificateManagementConfig.KafkaTLSCertFilePath, &c.ManualCertificateManagementConfig.KafkaTLSCert)
		if err != nil {
			return err
		}
		err = shared.ReadFileValueString(c.ManualCertificateManagementConfig.KafkaTLSKeyFilePath, &c.ManualCertificateManagementConfig.KafkaTLSKey)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *KafkaTLSCertificateManagementConfig) Validate(env *environments.Env) error {
	if !arrays.Contains(validStorageTypes, c.StorageType) {
		return fmt.Errorf("invalid storage type %q supplied. Valid storage types are %v", c.StorageType, validStorageTypes)
	}

	if !arrays.Contains(validCertificateManagementStrategies, c.CertificateManagementStrategy) {
		return fmt.Errorf("invalid certificate management strategy %q supplied. Valid strategies are %v", c.CertificateManagementStrategy, validCertificateManagementStrategies)
	}

	if c.CertificateManagementStrategy == AutomaticCertificateManagement && c.EnableKafkaExternalCertificate {
		err := validator.New().Struct(c.AutomaticCertificateManagementConfig)
		if err != nil {
			return errors.Wrap(err, "error validating the automatic kafka tls certificate management configuration")
		}
	}

	if c.CertificateManagementStrategy == ManualCertificateManagement && c.EnableKafkaExternalCertificate {
		err := validator.New().Struct(c.ManualCertificateManagementConfig)
		if err != nil {
			return errors.Wrap(err, "error validating the manual kafka tls certificate management configuration")
		}
	}

	return nil
}
