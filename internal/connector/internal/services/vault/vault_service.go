package vault

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/metrics"
)

const (
	KindTmp = "tmp"
	KindAws = "aws"

	DefaultRegion = "us-east-1"
)

type VaultService interface {
	SetSecretString(name string, value string, owningResource string) error
	GetSecretString(name string) (string, error)
	DeleteSecretString(name string) error
	ForEachSecret(f func(name string, owningResource string) bool) error
	Kind() string
}

func NewVaultService(vaultConfig *Config) (VaultService, error) {
	metrics.ResetMetricsForVaultService()
	switch vaultConfig.Kind {
	case KindAws:
		return NewAwsVaultService(vaultConfig)
	case KindTmp:
		return NewTmpVaultService()
	default:
		return nil, fmt.Errorf("invalid vault kind: %s", vaultConfig.Kind)
	}
}
