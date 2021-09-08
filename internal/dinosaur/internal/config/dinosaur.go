package config

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/shared"
	"github.com/ghodss/yaml"
	"github.com/spf13/pflag"
)

type DinosaurCapacityConfig struct {
	IngressEgressThroughputPerSec string `json:"ingressEgressThroughputPerSec"`
	TotalMaxConnections           int    `json:"totalMaxConnections"`
	MaxDataRetentionSize          string `json:"maxDataRetentionSize"`
	MaxPartitions                 int    `json:"maxPartitions"`
	MaxDataRetentionPeriod        string `json:"maxDataRetentionPeriod"`
	MaxConnectionAttemptsPerSec   int    `json:"maxConnectionAttemptsPerSec"`
	MaxCapacity                   int64  `json:"maxCapacity"`
}

type DinosaurConfig struct {
	DinosaurTLSCert                   string                 `json:"DINOSAUR_TLS_CRT"`
	DinosaurTLSCertFile               string                 `json:"DINOSAUR_TLS_CRT_file"`
	DinosaurTLSKey                    string                 `json:"DINOSAUR_TLS_KEY"`
	DinosaurTLSKeyFile                string                 `json:"DINOSAUR_TLS_KEY_file"`
	EnableDinosaurExternalCertificate bool                   `json:"enable_dinosaur_external_certificate"`
	NumOfBrokers                      int                    `json:"num_of_brokers"`
	DinosaurDomainName                string                 `json:"dinosaur_domain_name"`
	DinosaurCapacity                  DinosaurCapacityConfig `json:"dinosaur_capacity_config"`
	DinosaurCapacityConfigFile        string                 `json:"dinosaur_capacity_config_file"`

	DefaultDinosaurVersion string                  `json:"default_dinosaur_version"`
	DinosaurLifespan       *DinosaurLifespanConfig `json:"dinosaur_lifespan"`
	Quota                  *DinosaurQuotaConfig    `json:"dinosaur_quota"`
}

func NewDinosaurConfig() *DinosaurConfig {
	return &DinosaurConfig{
		DinosaurTLSCertFile:               "secrets/dinosaur-tls.crt",
		DinosaurTLSKeyFile:                "secrets/dinosaur-tls.key",
		EnableDinosaurExternalCertificate: false,
		DinosaurDomainName:                "dinosaur.devshift.org",
		NumOfBrokers:                      3,
		DinosaurCapacityConfigFile:        "config/dinosaur-capacity-config.yaml",
		DefaultDinosaurVersion:            "2.7.0",
		DinosaurLifespan:                  NewDinosaurLifespanConfig(),
		Quota:                             NewDinosaurQuotaConfig(),
	}
}

func (c *DinosaurConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.DinosaurTLSCertFile, "dinosaur-tls-cert-file", c.DinosaurTLSCertFile, "File containing dinosaur certificate")
	fs.StringVar(&c.DinosaurTLSKeyFile, "dinosaur-tls-key-file", c.DinosaurTLSKeyFile, "File containing dinosaur certificate private key")
	fs.BoolVar(&c.EnableDinosaurExternalCertificate, "enable-dinosaur-external-certificate", c.EnableDinosaurExternalCertificate, "Enable custom certificate for Dinosaur TLS")
	fs.StringVar(&c.DinosaurCapacityConfigFile, "dinosaur-capacity-config-file", c.DinosaurCapacityConfigFile, "File containing dinosaur capacity configurations")
	fs.StringVar(&c.DefaultDinosaurVersion, "default-dinosaur-version", c.DefaultDinosaurVersion, "The default version of Dinosaur when creating Dinosaur instances")
	fs.BoolVar(&c.DinosaurLifespan.EnableDeletionOfExpiredDinosaur, "enable-deletion-of-expired-dinosaur", c.DinosaurLifespan.EnableDeletionOfExpiredDinosaur, "Enable the deletion of dinosaurs when its life span has expired")
	fs.IntVar(&c.DinosaurLifespan.DinosaurLifespanInHours, "dinosaur-lifespan", c.DinosaurLifespan.DinosaurLifespanInHours, "The desired lifespan of a Dinosaur instance")
	fs.StringVar(&c.DinosaurDomainName, "dinosaur-domain-name", c.DinosaurDomainName, "The domain name to use for Dinosaur instances")
	fs.StringVar(&c.Quota.Type, "quota-type", c.Quota.Type, "The type of the quota service to be used. The available options are: 'ams' for AMS backed implementation and 'quota-management-list' for quota list backed implementation (default).")
	fs.BoolVar(&c.Quota.AllowEvaluatorInstance, "allow-evaluator-instance", c.Quota.AllowEvaluatorInstance, "Allow the creation of dinosaur evaluator instances")
}

func (c *DinosaurConfig) ReadFiles() error {
	err := shared.ReadFileValueString(c.DinosaurTLSCertFile, &c.DinosaurTLSCert)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(c.DinosaurTLSKeyFile, &c.DinosaurTLSKey)
	if err != nil {
		return err
	}
	content, err := shared.ReadFile(c.DinosaurCapacityConfigFile)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal([]byte(content), &c.DinosaurCapacity)
	if err != nil {
		return err
	}
	return nil
}
