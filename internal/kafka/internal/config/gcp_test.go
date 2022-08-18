package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cloudproviders"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/onsi/gomega"
)

func Test_GCPConfig_ReadFiles(t *testing.T) {
	testTempFilePrefix := "test_gcpconfig_readfiles"
	type fields struct {
		GCPConfigFactory func() GCPConfig
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "When file exists and it is a valid JSON no error is returned",
			fields: fields{
				GCPConfigFactory: func() GCPConfig {
					correctlyFormattedJSON := "{}"
					gcpCredentialsFile, err := shared.CreateTempFileFromStringData(testTempFilePrefix, correctlyFormattedJSON)
					if err != nil {
						panic(fmt.Errorf("test error: %v", err))
					}
					return GCPConfig{
						gcpCredentialsFilePath: gcpCredentialsFile,
					}
				},
			},
			wantErr: false,
		},
		{
			name: "When file does not exist no error is returned",
			fields: fields{
				GCPConfigFactory: func() GCPConfig {
					return GCPConfig{
						gcpCredentialsFilePath: "unexistingfilename",
					}
				},
			},
			wantErr: false,
		},
		{
			name: "When file exists but it is not a valid JSON an error is returned",
			fields: fields{
				GCPConfigFactory: func() GCPConfig {
					incorrectJSON := "anincorrect: j son"
					gcpCredentialsFile, err := shared.CreateTempFileFromStringData(testTempFilePrefix, incorrectJSON)
					if err != nil {
						panic(fmt.Errorf("test error: %v", err))
					}

					return GCPConfig{
						gcpCredentialsFilePath: gcpCredentialsFile,
					}
				},
			},
			wantErr: true,
		},
		{
			name: "When file exists and it is empty no error is returned",
			fields: fields{
				GCPConfigFactory: func() GCPConfig {
					emptyFileContent := ""
					gcpCredentialsFile, err := shared.CreateTempFileFromStringData(testTempFilePrefix, emptyFileContent)
					if err != nil {
						panic(fmt.Errorf("test error: %v", err))
					}

					return GCPConfig{
						gcpCredentialsFilePath: gcpCredentialsFile,
					}
				},
			},
			wantErr: false,
		},
		{
			name: "When file exists and it is blank no error is returned",
			fields: fields{
				GCPConfigFactory: func() GCPConfig {
					emptyFileContent := "\n\n \n   \n  \t  \n  "
					gcpCredentialsFile, err := shared.CreateTempFileFromStringData(testTempFilePrefix, emptyFileContent)
					if err != nil {
						panic(fmt.Errorf("test error: %v", err))
					}

					return GCPConfig{
						gcpCredentialsFilePath: gcpCredentialsFile,
					}
				},
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			gcpConfig := tt.fields.GCPConfigFactory()
			// cleanup of test temporary files
			defer os.Remove(gcpConfig.gcpCredentialsFilePath)

			err := gcpConfig.ReadFiles()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_GCPConfig_Validate(t *testing.T) {
	// Test that when the providers configuration can be resolved
	// and the GCP credentials are valid there is no panic nor error returned
	env, err := environments.New(environments.GetEnvironmentStrFromEnv())
	if err != nil {
		t.Errorf("failed to set provider configuration")
	}

	testProvidersConfig := NewSupportedProvidersConfig()
	testProvidersConfig.ProvidersConfig = ProviderConfiguration{
		SupportedProviders: ProviderList{
			Provider{Name: cloudproviders.AWS.String()},
			Provider{Name: cloudproviders.GCP.String()},
		},
	}
	if err := env.ConfigContainer.ProvideValue(testProvidersConfig); err != nil {
		t.Errorf("failed to set test supported providers configuration")
	}

	testGCPConfig := NewGCPConfig()
	testGCPConfig.GCPCredentials = GCPCredentials{
		AuthProviderX509CertURL: "test",
		AuthURI:                 "test",
		ClientEmail:             "test",
		ClientID:                "test",
		ClientX509CertURL:       "test",
		PrivateKey:              "test",
		PrivateKeyID:            "test",
		ProjectID:               "test",
		TokenURI:                "test",
		Type:                    "test",
	}
	g := gomega.NewWithT(t)
	res := testGCPConfig.Validate(env)
	g.Expect(res).NotTo(gomega.HaveOccurred())
}

func Test_GCPCredentials_validate(t *testing.T) {
	type fields struct {
		credentials *GCPCredentials
	}

	type args struct {
		providerList ProviderList
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "validation is skipped when gcp is not in the list of providers",
			fields: fields{
				credentials: &GCPCredentials{},
			},
			args: args{
				providerList: ProviderList{
					Provider{
						Name: cloudproviders.AWS.String(),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "validation fails for the default GCP credentials object when gcp is in the list of providers",
			fields: fields{
				credentials: &GCPCredentials{},
			},
			args: args{
				providerList: ProviderList{
					Provider{Name: cloudproviders.GCP.String()},
					Provider{Name: cloudproviders.AWS.String()},
				},
			},
			wantErr: true,
		},
		{
			name: "validation is successful when GCP credentials attributes are all valid",
			fields: fields{
				credentials: &GCPCredentials{
					AuthProviderX509CertURL: "test",
					AuthURI:                 "test",
					ClientEmail:             "test",
					ClientID:                "test",
					ClientX509CertURL:       "test",
					PrivateKey:              "test",
					PrivateKeyID:            "test",
					ProjectID:               "test",
					TokenURI:                "test",
					Type:                    "test",
				},
			},
			args: args{
				providerList: ProviderList{
					Provider{Name: cloudproviders.GCP.String()},
					Provider{Name: cloudproviders.AWS.String()},
				},
			},
			wantErr: false,
		},
		{
			name: "validation fails when one of the GCP credentials attributes is not valid",
			fields: fields{
				credentials: &GCPCredentials{
					AuthProviderX509CertURL: "test",
					AuthURI:                 "test",
					ClientEmail:             "test",
					ClientID:                "test",
					ClientX509CertURL:       "test",
					PrivateKey:              "test",
					PrivateKeyID:            "test",
					ProjectID:               "test",
					TokenURI:                "test",
					Type:                    "",
				},
			},
			args: args{
				providerList: ProviderList{
					Provider{Name: cloudproviders.GCP.String()},
					Provider{Name: cloudproviders.AWS.String()},
				},
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)

			gcpCredentials := tt.fields.credentials
			err := gcpCredentials.validate(tt.args.providerList)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}

}
