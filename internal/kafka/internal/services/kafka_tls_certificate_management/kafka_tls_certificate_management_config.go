package kafka_tls_certificate_management

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
	"github.com/caddyserver/certmagic"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

const (
	inMemoryStorageType            = "in-memory"
	vaultStorageType               = "vault"
	fileStorageType                = "file"
	manualCertificateManagement    = "manual"
	automaticCertificateManagement = "automatic"
)

var validStorageTypes = []string{inMemoryStorageType, fileStorageType, vaultStorageType}
var validCertificateManagementStrategies = []string{manualCertificateManagement, automaticCertificateManagement}

type KafkaTLSCertificateManagementConfig struct {
	CertificateAuthorityEndpoint         string
	CertificateManagementStrategy        string
	StorageType                          string
	EnableKafkaExternalCertificate       bool
	ManualCertificateManagementConfig    ManualCertificateManagementConfig
	AutomaticCertificateManagementConfig AutomaticCertificateManagementConfig
}

type ManualCertificateManagementConfig struct {
	KafkaTLSCert     string
	KafkaTLSCertFile string `validate:"required"`
	KafkaTLSKey      string
	KafkaTLSKeyFile  string `validate:"required"`
}

type AutomaticCertificateManagementConfig struct {
	RenewalWindowRatio        float64 `validate:"gte=0,lte=1"`
	EmailToSendNotificationTo string  `validate:"required,email"`
	AcmeIssuerAccountKeyFile  string  `validate:"required"`
	AcmeIssuerAccountKey      string
	MustStaple                bool
}

func NewCertificateManagementConfig() *KafkaTLSCertificateManagementConfig {
	return &KafkaTLSCertificateManagementConfig{
		CertificateAuthorityEndpoint:   certmagic.LetsEncryptProductionCA,
		StorageType:                    inMemoryStorageType,
		EnableKafkaExternalCertificate: false,
		ManualCertificateManagementConfig: ManualCertificateManagementConfig{
			KafkaTLSCertFile: "secrets/kafka-tls.crt",
			KafkaTLSKeyFile:  "secrets/kafka-tls.key",
		},
		CertificateManagementStrategy: manualCertificateManagement,
		AutomaticCertificateManagementConfig: AutomaticCertificateManagementConfig{
			EmailToSendNotificationTo: "",
			RenewalWindowRatio:        certmagic.Default.RenewalWindowRatio,
			AcmeIssuerAccountKeyFile:  "secrets/kafka-tls-certificate-management-acme-issuer-account-key.pem",
			MustStaple:                false,
		},
	}
}

func (c *KafkaTLSCertificateManagementConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.CertificateManagementStrategy, "kafka-tls-certificate-management-strategy", c.CertificateManagementStrategy, "The strategy used to manage tls certificates: Supported values are 'manual', 'automatic'. The default value is 'manual'")
	fs.StringVar(&c.StorageType, "kafka-tls-certificate-management-storage-type", c.StorageType, "The storage type of the tls certificates: Supported values are 'in-memory', 'vault', 'file'. The default value is 'in-memory'")
	fs.StringVar(&c.ManualCertificateManagementConfig.KafkaTLSCertFile, "kafka-tls-cert-file", c.ManualCertificateManagementConfig.KafkaTLSCertFile, "File containing kafka certificate")
	fs.StringVar(&c.ManualCertificateManagementConfig.KafkaTLSKeyFile, "kafka-tls-key-file", c.ManualCertificateManagementConfig.KafkaTLSKeyFile, "File containing kafka certificate private key")
	fs.BoolVar(&c.EnableKafkaExternalCertificate, "enable-kafka-external-certificate", c.EnableKafkaExternalCertificate, "Enable custom certificate for Kafka TLS")
	fs.BoolVar(&c.AutomaticCertificateManagementConfig.MustStaple, "kafka-tls-certificate-management-must-staple", c.AutomaticCertificateManagementConfig.MustStaple, "Adds the must staple TLS extension to the certificate signing request.")
	fs.StringVar(&c.AutomaticCertificateManagementConfig.EmailToSendNotificationTo, "kafka-tls-certificate-management-email", c.AutomaticCertificateManagementConfig.EmailToSendNotificationTo, "The email address that will receive certificate notification. The field is required")
	fs.StringVar(&c.AutomaticCertificateManagementConfig.AcmeIssuerAccountKeyFile, "kafka-tls-certificate-management-acme-issuer-account-key-file", c.AutomaticCertificateManagementConfig.AcmeIssuerAccountKeyFile, "The file containing the ACME Issuer account key. This is required")
	fs.Float64Var(&c.AutomaticCertificateManagementConfig.RenewalWindowRatio, "kafka-tls-certificate-management-renewal-window-ratio", c.AutomaticCertificateManagementConfig.RenewalWindowRatio, "How much of a certificate's lifetime becomes the renewal window")
}

func (c *KafkaTLSCertificateManagementConfig) ReadFiles() error {
	if c.CertificateManagementStrategy == automaticCertificateManagement {
		err := shared.ReadFileValueString(c.AutomaticCertificateManagementConfig.AcmeIssuerAccountKeyFile, &c.AutomaticCertificateManagementConfig.AcmeIssuerAccountKey)
		if err != nil {
			return err
		}
	}

	// We always read the manual tls certificates irrespective of strategy used.
	// This is done so that we can default to using the manually provided certificate during the transition from manual to automatic certificate handling.
	// That is, we want the transition to be a smooth one, without breaking the reconciliation loop between fleet manager and fleetshard sync
	// for Kafkas that whose certificates information has not been reconcilied internally (populated in the database) by the fleet manager.
	if c.EnableKafkaExternalCertificate {
		err := shared.ReadFileValueString(c.ManualCertificateManagementConfig.KafkaTLSCertFile, &c.ManualCertificateManagementConfig.KafkaTLSCert)
		if err != nil {
			return err
		}
		err = shared.ReadFileValueString(c.ManualCertificateManagementConfig.KafkaTLSKeyFile, &c.ManualCertificateManagementConfig.KafkaTLSKey)
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

	if c.CertificateManagementStrategy == automaticCertificateManagement && c.EnableKafkaExternalCertificate {
		err := validator.New().Struct(c.AutomaticCertificateManagementConfig)
		if err != nil {
			return errors.Wrap(err, "error validating the automatic kafka tls  certificate management configuration")
		}
	}

	if c.CertificateManagementStrategy == manualCertificateManagement && c.EnableKafkaExternalCertificate {
		err := validator.New().Struct(c.ManualCertificateManagementConfig)
		if err != nil {
			return errors.Wrap(err, "error validating the manual kafka tls certificate management configuration")
		}
	}

	return nil
}
