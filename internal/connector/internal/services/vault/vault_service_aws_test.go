package vault

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/onsi/gomega"
	"math/rand"
	"os"
	"testing"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func Test_awsVaultService(t *testing.T) {
	g := gomega.NewWithT(t)
	vc := NewConfig()

	// Enable testing against aws if the access keys are configured..
	if content, err := os.ReadFile(shared.BuildFullFilePath(vc.AccessKeyFile)); err == nil && len(content) > 0 {
		vc.Kind = "aws"
	}
	g.Expect(vc.ReadFiles()).To(gomega.BeNil())

	// only run the tests below when aws vault service is configured
	if vc.Kind == "aws" {

		tests := []struct {
			config        *Config
			name          string
			validationErr bool
		}{
			{
				config: &Config{
					Kind:               KindAws,
					AccessKey:          vc.AccessKey,
					SecretAccessKey:    vc.SecretAccessKey,
					Region:             vc.Region,
					SecretPrefixEnable: false,
					SecretPrefix:       "managed-connectors",
				},
				name: "aws-secrets-no-prefix",
			},
			{
				config: &Config{
					Kind:               KindAws,
					AccessKey:          vc.AccessKey,
					SecretAccessKey:    vc.SecretAccessKey,
					Region:             vc.Region,
					SecretPrefixEnable: true,
					SecretPrefix:       "",
				},
				name:          "aws-secrets-invalid-prefix",
				validationErr: true,
			},
			{
				config: &Config{
					Kind:               KindAws,
					AccessKey:          vc.AccessKey,
					SecretAccessKey:    vc.SecretAccessKey,
					Region:             vc.Region,
					SecretPrefixEnable: true,
					SecretPrefix:       "managed-connectors",
				},
				name: "aws-secrets-with-prefix",
			},
		}

		for _, testcase := range tests {
			tt := testcase
			t.Run(tt.name, func(t *testing.T) {
				g = gomega.NewWithT(t)

				svc, err := NewVaultService(tt.config)
				g.Expect(err).To(gomega.BeNil())

				err = tt.config.Validate(nil)
				g.Expect(err != nil).To(gomega.Equal(tt.validationErr))
				if tt.validationErr {
					return
				}

				name := fmt.Sprintf("testkey%d", rand.Uint64())
				value := "testvalue"
				err = svc.SetSecretString(name, value, "/v1/connector/test")
				g.Expect(err).To(gomega.BeNil())

				// validate that aws secret has appropriate name
				var secretName string
				if tt.config.SecretPrefixEnable {
					secretName = tt.config.SecretPrefix + "/" + name
				} else {
					secretName = name
				}
				result, err := svc.(*awsVaultService).secretCache.GetSecretString(secretName)
				g.Expect(err).To(gomega.BeNil())
				g.Expect(result).To(gomega.Equal(value))

				result, err = svc.GetSecretString(name)
				g.Expect(err).To(gomega.BeNil())
				g.Expect(result).To(gomega.Equal(value))

				err = svc.DeleteSecretString(name)
				g.Expect(err).To(gomega.BeNil())
			})
		}
	}
}
