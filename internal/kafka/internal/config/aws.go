package config

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/spf13/pflag"
)

type AWSConfig struct {
	// Used for OSD Cluster creation with OCM
	AccountID           string
	AccountIDFile       string
	AccessKey           string
	AccessKeyFile       string
	SecretAccessKey     string
	SecretAccessKeyFile string

	// Used for domain modifications in Route 53
	Route53AccessKey           string
	Route53AccessKeyFile       string
	Route53SecretAccessKey     string
	Route53SecretAccessKeyFile string
}

func NewAWSConfig() *AWSConfig {
	return &AWSConfig{
		AccountIDFile:              "secrets/aws.accountid",
		AccessKeyFile:              "secrets/aws.accesskey",
		SecretAccessKeyFile:        "secrets/aws.secretaccesskey",
		Route53AccessKeyFile:       "secrets/aws.route53accesskey",
		Route53SecretAccessKeyFile: "secrets/aws.route53secretaccesskey",
	}
}

func (c *AWSConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.AccountIDFile, "aws-account-id-file", c.AccountIDFile, "File containing AWS account id")
	fs.StringVar(&c.AccessKeyFile, "aws-access-key-file", c.AccessKeyFile, "File containing AWS access key")
	fs.StringVar(&c.SecretAccessKeyFile, "aws-secret-access-key-file", c.SecretAccessKeyFile, "File containing AWS secret access key")
	fs.StringVar(&c.Route53AccessKeyFile, "aws-route53-access-key-file", c.Route53AccessKeyFile, "File containing AWS access key for route53")
	fs.StringVar(&c.Route53SecretAccessKeyFile, "aws-route53-secret-access-key-file", c.Route53SecretAccessKeyFile, "File containing AWS secret access key for route53")
}

func (c *AWSConfig) ReadFiles() error {
	err := shared.ReadFileValueString(c.AccountIDFile, &c.AccountID)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(c.AccessKeyFile, &c.AccessKey)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(c.SecretAccessKeyFile, &c.SecretAccessKey)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(c.Route53AccessKeyFile, &c.Route53AccessKey)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(c.Route53SecretAccessKeyFile, &c.Route53SecretAccessKey)
	return err
}
