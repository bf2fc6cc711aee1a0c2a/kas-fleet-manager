package config

import (
	"github.com/spf13/pflag"
)

type KafkaConfig struct {
	KafkaTLSCert           string `json:"kafka_tls_cert"`
	KafkaTLSCertFile       string `json:"kafka_tls_cert_file"`
	KafkaTLSKey            string `json:"kafka_tls_key"`
	KafkaTLSKeyFile        string `json:"kafka_tls_key_file"`
	EnableKafkaTLS         bool   `json:"enable_kafka_tls"`
	NumOfBrokers           int    `json:"num_of_brokers"`
	KafkaDomainName        string `json:"kafka_domain_name"`
	EnableDedicatedIngress bool   `json:"enable_dedicated_ingress"`
	KafkaStorageClass      string  `json:"kafka_storage_class"`
	KafkaCanaryImage       string  `json:"kafka_canary_image"`
	KafkaAdminServerImage  string  `json:"kafka_admin_server_image"`
}

func NewKafkaConfig() *KafkaConfig {
	return &KafkaConfig{
		KafkaTLSCertFile:       "secrets/kafka-tls.crt",
		KafkaTLSKeyFile:        "secrets/kafka-tls.key",
		EnableKafkaTLS:         false,
		KafkaDomainName:        "kafka.devshift.org",
		EnableDedicatedIngress: false,
		NumOfBrokers:           3,
		KafkaStorageClass:      "",
		KafkaCanaryImage:       "quay.io/ppatierno/strimzi-canary:0.0.1",
		KafkaAdminServerImage:  "quay.io/sknot/strimzi-admin:0.0.2",
	}
}

func (c *KafkaConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.KafkaTLSCertFile, "kafka-tls-cert-file", c.KafkaTLSCertFile, "File containing kafka certificate")
	fs.StringVar(&c.KafkaTLSKeyFile, "kafka-tls-key-file", c.KafkaTLSKeyFile, "File containing kafka certificate private key")
	fs.BoolVar(&c.EnableKafkaTLS, "enable-kafka-tls", c.EnableKafkaTLS, "Enable custom certificate for Kafka TLS")
	fs.BoolVar(&c.EnableDedicatedIngress, "enable-dedicated-ingress", c.EnableDedicatedIngress, "Enable a dedicated ingress for Kafka")
	fs.StringVar(&c.KafkaStorageClass, "kafka-storage-class", c.KafkaStorageClass, "Specifies a storage class for Kafka")
	fs.StringVar(&c.KafkaCanaryImage, "kafka-canary-image", c.KafkaCanaryImage, "Specifies a canary image")
	fs.StringVar(&c.KafkaAdminServerImage, "kafka-admin-server-image", c.KafkaAdminServerImage, "Specifies an admin server image")
}

func (c *KafkaConfig) ReadFiles() error {
	err := readFileValueString(c.KafkaTLSCertFile, &c.KafkaTLSCert)
	if err != nil {
		return err
	}
	err = readFileValueString(c.KafkaTLSKeyFile, &c.KafkaTLSKey)
	return err
}
