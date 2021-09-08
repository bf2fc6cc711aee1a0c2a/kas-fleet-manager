package vault_test

import (
	"io/ioutil"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/services/vault"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/shared"
	. "github.com/onsi/gomega"
)

func TestNewVaultService(t *testing.T) {
	RegisterTestingT(t)
	vc := vault.NewConfig()

	// Enable testing against aws if the access keys are configured..
	if content, err := ioutil.ReadFile(shared.BuildFullFilePath(vc.AccessKeyFile)); err == nil && len(content) > 0 {
		vc.Kind = "aws"
	}
	Expect(vc.ReadFiles()).To(BeNil())

	tests := []struct {
		config       *vault.Config
		wantErrOnNew bool
		skip         bool
	}{
		{
			config: &vault.Config{Kind: "tmp"},
		},
		{
			config: &vault.Config{
				Kind:            "aws",
				AccessKey:       vc.AccessKey,
				SecretAccessKey: vc.SecretAccessKey,
				Region:          vc.Region,
			},
			skip: vc.Kind != "aws",
		},
		{
			config:       &vault.Config{Kind: "wrong"},
			wantErrOnNew: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.config.Kind, func(t *testing.T) {
			RegisterTestingT(t)
			svc, err := vault.NewVaultService(tt.config)
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

func happyPath(vault vault.VaultService) {

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
