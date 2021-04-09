package config

import (
	"github.com/spf13/pflag"
)

type VaultConfig struct {
	// Used for OSD Cluster creation with OCM
	Kind                string `json:"kind"`
	AccessKey           string `json:"access_key"`
	AccessKeyFile       string `json:"access_key_file"`
	SecretAccessKey     string `json:"secret_access_key"`
	SecretAccessKeyFile string `json:"secret_access_key_file"`
	Region              string `json:"region"`
}

func NewVaultConfig() *VaultConfig {
	return &VaultConfig{
		Kind:                "tmp",
		AccessKeyFile:       "secrets/vault.accesskey",
		SecretAccessKeyFile: "secrets/vault.secretaccesskey",
		Region:              "us-east-1",
	}
}

func (c *VaultConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.Kind, "vault-kind", c.Kind, "The kind of vault to use: aws|tmp")
	fs.StringVar(&c.AccessKeyFile, "vault-access-key-file", c.AccessKeyFile, "File containing vault access key")
	fs.StringVar(&c.SecretAccessKeyFile, "vault-secret-access-key-file", c.SecretAccessKeyFile, "File containing vault secret access key")
	fs.StringVar(&c.Region, "vault-region", c.Region, "The region of the vault")
}

func (c *VaultConfig) ReadFiles() error {
	if c.Kind == "aws" {
		err := readFileValueString(c.AccessKeyFile, &c.AccessKey)
		if err != nil {
			return err
		}
		err = readFileValueString(c.SecretAccessKeyFile, &c.SecretAccessKey)
		if err != nil {
			return err
		}
	}
	return nil
}
