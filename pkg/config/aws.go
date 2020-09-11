package config

import (
	"github.com/spf13/pflag"
)

type AWSConfig struct {
	AccountID           string `json:"account_id"`
	AccountIDFile       string `json:"account_id_file"`
	AccessKey           string `json:"access_key"`
	AccessKeyFile       string `json:"access_key_file"`
	SecretAccessKey     string `json:"secret_access_key"`
	SecretAccessKeyFile string `json:"secret_access_key_file"`
}

func NewAWSConfig() *AWSConfig {
	return &AWSConfig{
		AccountIDFile:       "secrets/aws.accountid",
		AccessKeyFile:       "secrets/aws.accesskey",
		SecretAccessKeyFile: "secrets/aws.secretaccesskey",
	}
}

func (c *AWSConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.AccountIDFile, "aws-account-id-file", c.AccountIDFile, "File containing AWS account id")
	fs.StringVar(&c.AccessKeyFile, "aws-access-key-file", c.AccessKeyFile, "File containing AWS access key")
	fs.StringVar(&c.SecretAccessKeyFile, "aws-secret-access-key-file", c.SecretAccessKeyFile, "File containing AWS secret access key")
}

func (c *AWSConfig) ReadFiles() error {
	err := readFileValueString(c.AccountIDFile, &c.AccountID)
	if err != nil {
		return err
	}
	err = readFileValueString(c.AccessKeyFile, &c.AccessKey)
	if err != nil {
		return err
	}
	err = readFileValueString(c.SecretAccessKeyFile, &c.SecretAccessKey)
	return err
}
