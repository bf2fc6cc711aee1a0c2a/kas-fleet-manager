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
}

func NewKafkaConfig() *KafkaConfig {
	return &KafkaConfig{
		KafkaTLSCertFile:       "secrets/kafka-tls.crt",
		KafkaTLSKeyFile:        "secrets/kafka-tls.key",
		EnableKafkaTLS:         false,
		KafkaDomainName:        "kafka.devshift.org",
		EnableDedicatedIngress: false,
		NumOfBrokers:           3,
	}
}

func (c *KafkaConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.KafkaTLSCertFile, "kafka-tls-cert-file", c.KafkaTLSCertFile, "File containing kafka certificate")
	fs.StringVar(&c.KafkaTLSKeyFile, "kafka-tls-key-file", c.KafkaTLSKeyFile, "File containing kafka certificate private key")
	fs.BoolVar(&c.EnableKafkaTLS, "enable-kafka-tls", c.EnableKafkaTLS, "Enable custom certificate for Kafka TLS")
	fs.BoolVar(&c.EnableDedicatedIngress, "enable-dedicated-ingress", c.EnableDedicatedIngress, "Enable a dedicated ingress for Kafka")
}

func (c *KafkaConfig) ReadFiles() error {
	err := readFileValueString(c.KafkaTLSCertFile, &c.KafkaTLSCert)
	if err != nil {
		return err
	}
	err = readFileValueString(c.KafkaTLSKeyFile, &c.KafkaTLSKey)
	return err
}
