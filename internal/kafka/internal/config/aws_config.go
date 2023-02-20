package config

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/spf13/pflag"
)

type AWSConfig struct {
	Route53                     awsRoute53Config
	SecretManager               awsSecretManagerConfig
	ConfigForOSDClusterCreation awsConfigForOSDClusterCreation
}

type awsSecretManagerConfig struct {
	Region                  string
	AccessKey               string
	SecretPrefix            string
	SecretAccessKey         string
	accessKeyFilePath       string
	secretAccessKeyFilePath string
}

type awsConfigForOSDClusterCreation struct {
	accountIDFilePath       string
	accessKeyFilePath       string
	secretAccessKeyFilePath string
	AccountID               string
	AccessKey               string
	SecretAccessKey         string
}

type awsRoute53Config struct {
	AccessKey               string
	SecretAccessKey         string
	accessKeyFilePath       string
	secretAccessKeyFilePath string
}

func NewAWSConfig() *AWSConfig {
	return &AWSConfig{
		ConfigForOSDClusterCreation: awsConfigForOSDClusterCreation{
			accountIDFilePath:       "secrets/aws.accountid",
			accessKeyFilePath:       "secrets/aws.accesskey",
			secretAccessKeyFilePath: "secrets/aws.secretaccesskey",
		},
		Route53: awsRoute53Config{
			accessKeyFilePath:       "secrets/aws.route53accesskey",
			secretAccessKeyFilePath: "secrets/aws.route53secretaccesskey",
		},
		SecretManager: awsSecretManagerConfig{
			accessKeyFilePath:       "secrets/aws-secret-manager/aws_access_key_id",
			secretAccessKeyFilePath: "secrets/aws-secret-manager/aws_secret_access_key",
			Region:                  "us-east-1",
			SecretPrefix:            "kas-fleet-manager",
		},
	}
}

func (c *AWSConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.ConfigForOSDClusterCreation.accountIDFilePath, "aws-account-id-file", c.ConfigForOSDClusterCreation.accountIDFilePath, "File containing AWS account id")
	fs.StringVar(&c.ConfigForOSDClusterCreation.accessKeyFilePath, "aws-access-key-file", c.ConfigForOSDClusterCreation.accessKeyFilePath, "File containing AWS access key")
	fs.StringVar(&c.ConfigForOSDClusterCreation.secretAccessKeyFilePath, "aws-secret-access-key-file", c.ConfigForOSDClusterCreation.secretAccessKeyFilePath, "File containing AWS secret access key")
	fs.StringVar(&c.Route53.accessKeyFilePath, "aws-route53-access-key-file", c.Route53.accessKeyFilePath, "File containing AWS access key for route53")
	fs.StringVar(&c.Route53.secretAccessKeyFilePath, "aws-route53-secret-access-key-file", c.Route53.secretAccessKeyFilePath, "File containing AWS secret access key for route53")
	fs.StringVar(&c.SecretManager.accessKeyFilePath, "aws-secret-manager-access-key-file", c.SecretManager.accessKeyFilePath, "File containing AWS secret manager access key")
	fs.StringVar(&c.SecretManager.secretAccessKeyFilePath, "aws-secret-manager-secret-access-key-file", c.SecretManager.secretAccessKeyFilePath, "File containing AWS secret manager secret access key")
	fs.StringVar(&c.SecretManager.SecretPrefix, "aws-secret-manager-secret-prefix", c.SecretManager.SecretPrefix, "Prefix to use for all secret names in AWS secret manager")
	fs.StringVar(&c.SecretManager.Region, "aws-secret-manager-region", c.SecretManager.Region, "The region of the AWS secret manager")
}

func (c *AWSConfig) ReadFiles() error {
	err := shared.ReadFileValueString(c.ConfigForOSDClusterCreation.accountIDFilePath, &c.ConfigForOSDClusterCreation.AccountID)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(c.ConfigForOSDClusterCreation.accessKeyFilePath, &c.ConfigForOSDClusterCreation.AccessKey)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(c.ConfigForOSDClusterCreation.secretAccessKeyFilePath, &c.ConfigForOSDClusterCreation.SecretAccessKey)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(c.Route53.accessKeyFilePath, &c.Route53.AccessKey)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(c.Route53.secretAccessKeyFilePath, &c.Route53.SecretAccessKey)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(c.SecretManager.accessKeyFilePath, &c.SecretManager.AccessKey)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(c.SecretManager.secretAccessKeyFilePath, &c.SecretManager.SecretAccessKey)
	if err != nil {
		return err
	}
	return nil
}
