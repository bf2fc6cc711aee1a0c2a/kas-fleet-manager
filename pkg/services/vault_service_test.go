package services_test

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	. "github.com/onsi/gomega"
	"testing"
)

func TestNewVaultService(t *testing.T) {
	RegisterTestingT(t)
	tests := []struct {
		config  *config.VaultConfig
		wantErr bool
	}{
		{
			config: &config.VaultConfig{Kind: "tmp"},
		},
		//{
		//	config: &config.VaultConfig{Kind: "aws"},
		//},
		{
			config:  &config.VaultConfig{Kind: "wrong"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.config.Kind, func(t *testing.T) {
			svc, err := services.NewVaultService(tt.config)
			Expect(err != nil).Should(Equal(tt.wantErr), "NewVaultService() error = %v, wantErr %v", err, tt.wantErr)
			if err == nil {
				happyPath(svc)
			}
		})
	}
}

func happyPath(vault services.VaultService) {

	counter := 0
	err := vault.ForEachSecret(func(name string, owningResource string) bool {
		counter += 1
		return true
	})
	Expect(err).Should(BeNil())
	Expect(counter).Should(Equal(0))

	err = vault.SetSecretString("test", "hello", "thistest")
	Expect(err).Should(BeNil())

	value, err := vault.GetSecretString("test")
	Expect(err).Should(BeNil())
	Expect(value).Should(Equal("hello"))

	err = vault.DeleteSecretString("test")
	Expect(err).Should(BeNil())

	value, err = vault.GetSecretString("test")
	Expect(err).Should(Not(BeNil()))
	Expect(value).Should(Equal(""))

}
