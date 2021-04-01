package config

import (
	"github.com/ghodss/yaml"
	"github.com/spf13/pflag"
)

type KafkaCapacityConfig struct {
	IngressEgressThroughputPerSec string `json:"ingressEgressThroughputPerSec"`
	TotalMaxConnections           int    `json:"totalMaxConnections"`
	MaxDataRetentionSize          string `json:"maxDataRetentionSize"`
	MaxPartitions                 int    `json:"maxPartitions"`
	MaxDataRetentionPeriod        string `json:"maxDataRetentionPeriod"`
	MaxCapacity                   int    `json:"maxCapacity"`
}

type KafkaConfig struct {
	KafkaTLSCert                   string              `json:"kafka_tls_cert"`
	KafkaTLSCertFile               string              `json:"kafka_tls_cert_file"`
	KafkaTLSKey                    string              `json:"kafka_tls_key"`
	KafkaTLSKeyFile                string              `json:"kafka_tls_key_file"`
	EnableKafkaExternalCertificate bool                `json:"enable_kafka_external_certificate"`
	NumOfBrokers                   int                 `json:"num_of_brokers"`
	KafkaDomainName                string              `json:"kafka_domain_name"`
	KafkaCanaryImage               string              `json:"kafka_canary_image"`
	KafkaAdminServerImage          string              `json:"kafka_admin_server_image"`
	EnableManagedKafkaCR           bool                `json:"enable_managedkafka_cr"`
	KafkaCapacity                  KafkaCapacityConfig `json:"kafka_capacity_config"`
	KafkaCapacityConfigFile        string              `json:"kafka_capacity_config_file"`
	EnableKasFleetshardSync        bool                `json:"enable_kas_fleetshard_sync"`
	EnableQuotaService             bool                `json:"enable_quota_service"`
}

func NewKafkaConfig() *KafkaConfig {
	return &KafkaConfig{
		KafkaTLSCertFile:               "secrets/kafka-tls.crt",
		KafkaTLSKeyFile:                "secrets/kafka-tls.key",
		EnableKafkaExternalCertificate: false,
		KafkaDomainName:                "kafka.devshift.org",
		NumOfBrokers:                   3,
		KafkaCanaryImage:               "quay.io/ppatierno/strimzi-canary:0.0.1-1",
		KafkaAdminServerImage:          "quay.io/sknot/kafka-admin-api:0.0.4",
		KafkaCapacityConfigFile:        "config/kafka-capacity-config.yaml",
		EnableKasFleetshardSync:        false,
		EnableQuotaService:             false,
	}
}

func (c *KafkaConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.KafkaTLSCertFile, "kafka-tls-cert-file", c.KafkaTLSCertFile, "File containing kafka certificate")
	fs.StringVar(&c.KafkaTLSKeyFile, "kafka-tls-key-file", c.KafkaTLSKeyFile, "File containing kafka certificate private key")
	fs.BoolVar(&c.EnableKafkaExternalCertificate, "enable-kafka-external-certificate", c.EnableKafkaExternalCertificate, "Enable custom certificate for Kafka TLS")
	fs.StringVar(&c.KafkaCanaryImage, "kafka-canary-image", c.KafkaCanaryImage, "Specifies a canary image")
	fs.StringVar(&c.KafkaAdminServerImage, "kafka-admin-server-image", c.KafkaAdminServerImage, "Specifies an admin server image")
	fs.BoolVar(&c.EnableManagedKafkaCR, "enable-managed-kafka-cr", c.EnableManagedKafkaCR, "Enable usage of the ManagedKafkaCR instead of the KafkaCR")
	fs.StringVar(&c.KafkaCapacityConfigFile, "kafka-capacity-config-file", c.KafkaCapacityConfigFile, "File containing kafka capacity configurations")
	fs.BoolVar(&c.EnableKasFleetshardSync, "enable-kas-fleetshard-sync", c.EnableKasFleetshardSync, "Enable direct data synchronisation with kas-fleetshard-operator")
	fs.BoolVar(&c.EnableQuotaService, "enable-quota-service", c.EnableQuotaService, "Enable quota service")
}

func (c *KafkaConfig) ReadFiles() error {
	err := readFileValueString(c.KafkaTLSCertFile, &c.KafkaTLSCert)
	if err != nil {
		return err
	}
	err = readFileValueString(c.KafkaTLSKeyFile, &c.KafkaTLSKey)
	if err != nil {
		return err
	}
	content, err := readFile(c.KafkaCapacityConfigFile)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal([]byte(content), &c.KafkaCapacity)
	if err != nil {
		return err
	}
	return nil
}
