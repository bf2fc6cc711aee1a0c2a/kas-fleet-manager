package services_test

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"testing"
)

func TestNewVaultService(t *testing.T) {
	env := environments.Environment()

	// Enable testing against aws if the access keys are configured..
	if content, err := ioutil.ReadFile(config.BuildFullFilePath(env.Config.Vault.AccessKeyFile)); err == nil && len(content) > 0 {
		env.Config.Vault.Kind = "aws"
	}
	_ = env.Initialize()

	tests := []struct {
		config       *config.VaultConfig
		wantErrOnNew bool
		skip         bool
	}{
		{
			config: &config.VaultConfig{Kind: "tmp"},
		},
		{
			config: &config.VaultConfig{
				Kind:            "aws",
				AccessKey:       env.Config.Vault.AccessKey,
				SecretAccessKey: env.Config.Vault.SecretAccessKey,
				Region:          env.Config.Vault.Region,
			},
			skip: env.Config.Vault.Kind != "aws",
		},
		{
			config:       &config.VaultConfig{Kind: "wrong"},
			wantErrOnNew: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.config.Kind, func(t *testing.T) {
			RegisterTestingT(t)
			svc, err := services.NewVaultService(tt.config)
			Expect(err != nil).Should(Equal(tt.wantErrOnNew), "NewVaultService() error = %v, wantErr %v", err, tt.wantErrOnNew)
			if err == nil {
				if tt.skip {
					t.SkipNow()
				}
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

	keyName := api.NewID()
	err = vault.SetSecretString(keyName, "hello", "thistest")
	Expect(err).Should(BeNil())

	value, err := vault.GetSecretString(keyName)
	Expect(err).Should(BeNil())
	Expect(value).Should(Equal("hello"))

	err = vault.DeleteSecretString(keyName)
	Expect(err).Should(BeNil())

}
