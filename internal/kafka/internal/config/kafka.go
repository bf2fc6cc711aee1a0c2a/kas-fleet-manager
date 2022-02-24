package config

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

type KafkaCapacityConfig struct {
	IngressEgressThroughputPerSec string `yaml:"ingressEgressThroughputPerSec"`
	TotalMaxConnections           int    `yaml:"totalMaxConnections"`
	MaxDataRetentionSize          string `yaml:"maxDataRetentionSize"`
	MaxPartitions                 int    `yaml:"maxPartitions"`
	MaxDataRetentionPeriod        string `yaml:"maxDataRetentionPeriod"`
	MaxConnectionAttemptsPerSec   int    `yaml:"maxConnectionAttemptsPerSec"`
	MaxCapacity                   int64  `yaml:"maxCapacity"`
}

type KafkaConfig struct {
	KafkaTLSCert                   string              `json:"kafka_tls_cert"`
	KafkaTLSCertFile               string              `json:"kafka_tls_cert_file"`
	KafkaTLSKey                    string              `json:"kafka_tls_key"`
	KafkaTLSKeyFile                string              `json:"kafka_tls_key_file"`
	EnableKafkaExternalCertificate bool                `json:"enable_kafka_external_certificate"`
	KafkaDomainName                string              `json:"kafka_domain_name"`
	KafkaCapacity                  KafkaCapacityConfig `json:"kafka_capacity_config"`
	KafkaCapacityConfigFile        string              `json:"kafka_capacity_config_file"`
	BrowserUrl                     string              `json:"browser_url"`

	KafkaLifespan       *KafkaLifespanConfig       `json:"kafka_lifespan"`
	Quota               *KafkaQuotaConfig          `json:"kafka_quota"`
	SupportedKafkaSizes *KafkaSupportedSizesConfig `json:"kafka_supported_sizes"`
}

func NewKafkaConfig() *KafkaConfig {
	return &KafkaConfig{
		KafkaTLSCertFile:               "secrets/kafka-tls.crt",
		KafkaTLSKeyFile:                "secrets/kafka-tls.key",
		EnableKafkaExternalCertificate: false,
		KafkaDomainName:                "kafka.bf2.dev",
		KafkaCapacityConfigFile:        "config/kafka-capacity-config.yaml",
		KafkaLifespan:                  NewKafkaLifespanConfig(),
		Quota:                          NewKafkaQuotaConfig(),
		BrowserUrl:                     "http://localhost:8080/",
		SupportedKafkaSizes:            NewKafkaSupportedSizesConfig(),
	}
}

func (c *KafkaConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.KafkaTLSCertFile, "kafka-tls-cert-file", c.KafkaTLSCertFile, "File containing kafka certificate")
	fs.StringVar(&c.KafkaTLSKeyFile, "kafka-tls-key-file", c.KafkaTLSKeyFile, "File containing kafka certificate private key")
	fs.BoolVar(&c.EnableKafkaExternalCertificate, "enable-kafka-external-certificate", c.EnableKafkaExternalCertificate, "Enable custom certificate for Kafka TLS")
	fs.StringVar(&c.KafkaCapacityConfigFile, "kafka-capacity-config-file", c.KafkaCapacityConfigFile, "File containing kafka capacity configurations")
	fs.BoolVar(&c.KafkaLifespan.EnableDeletionOfExpiredKafka, "enable-deletion-of-expired-kafka", c.KafkaLifespan.EnableDeletionOfExpiredKafka, "Enable the deletion of kafkas when its life span has expired")
	fs.IntVar(&c.KafkaLifespan.KafkaLifespanInHours, "kafka-lifespan", c.KafkaLifespan.KafkaLifespanInHours, "The desired lifespan of a Kafka instance")
	fs.StringVar(&c.KafkaDomainName, "kafka-domain-name", c.KafkaDomainName, "The domain name to use for Kafka instances")
	fs.StringVar(&c.Quota.Type, "quota-type", c.Quota.Type, "The type of the quota service to be used. The available options are: 'ams' for AMS backed implementation and 'quota-management-list' for quota list backed implementation (default).")
	fs.BoolVar(&c.Quota.AllowEvaluatorInstance, "allow-evaluator-instance", c.Quota.AllowEvaluatorInstance, "Allow the creation of kafka evaluator instances")
	fs.StringVar(&c.BrowserUrl, "browser-url", c.BrowserUrl, "Browser url to kafka admin UI")
	fs.StringVar(&c.SupportedKafkaSizes.SupportedKafkaSizesConfigFile, "supported-kafka-sizes-config-file", c.SupportedKafkaSizes.SupportedKafkaSizesConfigFile, "File containing the supported kafka sizes configuration")
}

func (c *KafkaConfig) ReadFiles() error {
	err := shared.ReadFileValueString(c.KafkaTLSCertFile, &c.KafkaTLSCert)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(c.KafkaTLSKeyFile, &c.KafkaTLSKey)
	if err != nil {
		return err
	}
	content, err := shared.ReadFile(c.KafkaCapacityConfigFile)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal([]byte(content), &c.KafkaCapacity)
	if err != nil {
		return err
	}

	supportedKafkaSizesContents, err := shared.ReadFile(c.SupportedKafkaSizes.SupportedKafkaSizesConfigFile)
	if err != nil {
		return err
	}
	return yaml.UnmarshalStrict([]byte(supportedKafkaSizesContents), &c.SupportedKafkaSizes.SupportedKafkaSizesConfig)
}

func (c *KafkaConfig) Validate(env *environments.Env) error {
	return c.SupportedKafkaSizes.SupportedKafkaSizesConfig.validate()
}
