package vault

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/spf13/pflag"
)

type Config struct {
	// Used for OSD Cluster creation with OCM
	Kind                string `json:"kind"`
	AccessKey           string `json:"access_key"`
	AccessKeyFile       string `json:"access_key_file"`
	SecretAccessKey     string `json:"secret_access_key"`
	SecretAccessKeyFile string `json:"secret_access_key_file"`
	SecretPrefix        string `json:"secret_prefix"`
	SecretPrefixEnable  bool   `json:"secret_prefix_enable"`
	Region              string `json:"region"`
}

func NewConfig() *Config {
	return &Config{
		Kind:                KindTmp,
		AccessKeyFile:       "secrets/vault/aws_access_key_id",
		SecretAccessKeyFile: "secrets/vault/aws_secret_access_key",
		Region:              DefaultRegion,
		SecretPrefixEnable:  false,
		SecretPrefix:        "managed-connectors",
	}
}

func (c *Config) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.Kind, "vault-kind", c.Kind, "The kind of vault to use: aws|tmp")
	fs.StringVar(&c.AccessKeyFile, "vault-access-key-file", c.AccessKeyFile, "File containing vault access key")
	fs.StringVar(&c.SecretAccessKeyFile, "vault-secret-access-key-file", c.SecretAccessKeyFile, "File containing vault secret access key")
	fs.BoolVar(&c.SecretPrefixEnable, "vault-secret-prefix-enable", c.SecretPrefixEnable, "Enable use of a prefix for all managed connectors secret names in AWS vault, default false")
	fs.StringVar(&c.SecretPrefix, "vault-secret-prefix", c.SecretPrefix, "Prefix to use for all managed connectors secret names in AWS vault")
	fs.StringVar(&c.Region, "vault-region", c.Region, "The region of the vault")
}

func (c *Config) Validate(env *environments.Env) error {
	if c.Kind == KindAws && c.SecretPrefixEnable && len(c.SecretPrefix) == 0 {
		return fmt.Errorf("error validating AWS vault config, vault-secret-prefix must be set to a non-empty value if vault-secret-prefix-enable is true")
	}
	return nil
}

func (c *Config) ReadFiles() error {
	if c.Kind == KindAws {
		err := shared.ReadFileValueString(c.AccessKeyFile, &c.AccessKey)
		if err != nil {
			return err
		}
		err = shared.ReadFileValueString(c.SecretAccessKeyFile, &c.SecretAccessKey)
		if err != nil {
			return err
		}
	}
	return nil
}
