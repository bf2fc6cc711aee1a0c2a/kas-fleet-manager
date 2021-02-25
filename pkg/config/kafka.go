package config

import (
	"github.com/spf13/pflag"
)

type KafkaConfig struct {
	KafkaTLSCert                   string `json:"kafka_tls_cert"`
	KafkaTLSCertFile               string `json:"kafka_tls_cert_file"`
	KafkaTLSKey                    string `json:"kafka_tls_key"`
	KafkaTLSKeyFile                string `json:"kafka_tls_key_file"`
	EnableKafkaExternalCertificate bool   `json:"enable_kafka_external_certificate"`
	NumOfBrokers                   int    `json:"num_of_brokers"`
	KafkaDomainName                string `json:"kafka_domain_name"`
	KafkaCanaryImage               string `json:"kafka_canary_image"`
	KafkaAdminServerImage          string `json:"kafka_admin_server_image"`
	EnableManagedKafkaCR           bool   `json:"enable_managedkafka_cr"`
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
	}
}

func (c *KafkaConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.KafkaTLSCertFile, "kafka-tls-cert-file", c.KafkaTLSCertFile, "File containing kafka certificate")
	fs.StringVar(&c.KafkaTLSKeyFile, "kafka-tls-key-file", c.KafkaTLSKeyFile, "File containing kafka certificate private key")
	fs.BoolVar(&c.EnableKafkaExternalCertificate, "enable-kafka-external-certificate", c.EnableKafkaExternalCertificate, "Enable custom certificate for Kafka TLS")
	fs.StringVar(&c.KafkaCanaryImage, "kafka-canary-image", c.KafkaCanaryImage, "Specifies a canary image")
	fs.StringVar(&c.KafkaAdminServerImage, "kafka-admin-server-image", c.KafkaAdminServerImage, "Specifies an admin server image")
	fs.BoolVar(&c.EnableManagedKafkaCR, "enable-managed-kafka-cr", c.EnableManagedKafkaCR, "Enable usage of the ManagedKafkaCR instead of the KafkaCR")
}

func (c *KafkaConfig) ReadFiles() error {
	err := readFileValueString(c.KafkaTLSCertFile, &c.KafkaTLSCert)
	if err != nil {
		return err
	}
	err = readFileValueString(c.KafkaTLSKeyFile, &c.KafkaTLSKey)
	return err
}
