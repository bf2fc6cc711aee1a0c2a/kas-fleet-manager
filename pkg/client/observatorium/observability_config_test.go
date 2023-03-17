package observatorium

import (
	"testing"

	"github.com/onsi/gomega"
)

func Test_ReadFiles_ObservabilityConfiguration(t *testing.T) {
	type fields struct {
		config *ObservabilityConfiguration
	}

	tests := []struct {
		name     string
		fields   fields
		modifyFn func(config *ObservabilityConfiguration)
		wantErr  bool
	}{
		{
			name: "should return no error when running ReadFiles with default ObservabilityConfiguration",
			fields: fields{
				config: NewObservabilityConfigurationConfig(),
			},
			wantErr: false,
		},
		{
			name: "should return an error with misconfigured DexPasswordFile",
			fields: fields{
				config: NewObservabilityConfigurationConfig(),
			},
			modifyFn: func(config *ObservabilityConfiguration) {
				config.DexPasswordFile = "invalid"
			},
			wantErr: true,
		},
		{
			name: "should return an error with misconfigured DexSecretFile",
			fields: fields{
				config: NewObservabilityConfigurationConfig(),
			},
			modifyFn: func(config *ObservabilityConfiguration) {
				config.DexSecretFile = "invalid"
			},
			wantErr: true,
		},
		{
			name: "should return an error with misconfigured AuthTokenFile",
			fields: fields{
				config: NewObservabilityConfigurationConfig(),
			},
			modifyFn: func(config *ObservabilityConfiguration) {
				config.AuthToken = ""
				config.AuthTokenFile = "invalid"
			},
			wantErr: true,
		},
		{
			name: "should return an error when observability cloudwatchlogs enabled but no file path for their configuration is provided",
			fields: fields{
				config: NewObservabilityConfigurationConfig(),
			},
			modifyFn: func(config *ObservabilityConfiguration) {
				config.ObservabilityCloudWatchLoggingConfig.CloudwatchLoggingEnabled = true
				config.ObservabilityCloudWatchLoggingConfig.configFilePath = ""
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			config := tt.fields.config
			if tt.modifyFn != nil {
				tt.modifyFn(config)
			}
			g.Expect(config.ReadFiles() != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_DataPlaneObservabilityConfig_validate(t *testing.T) {
	type fields struct {
		DataPlaneObservabilityConfig DataPlaneObservabilityConfig
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should succeed when disabled",
			fields: fields{
				DataPlaneObservabilityConfig: DataPlaneObservabilityConfig{
					Enabled: false,
				},
			},
			wantErr: false,
		},
		{
			name: "should succeed when remote write url is provided, but no oidc config",
			fields: fields{
				DataPlaneObservabilityConfig: DataPlaneObservabilityConfig{
					Enabled: true,
					RemoteWriteConfiguration: &DataPlaneObservabilityRemoteWriteConfiguration{
						RemoteWriteUrl: "https://dummy",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should succeed when remote write url is provided and full oidc configuration is provided",
			fields: fields{
				DataPlaneObservabilityConfig: DataPlaneObservabilityConfig{
					Enabled: true,
					RemoteWriteConfiguration: &DataPlaneObservabilityRemoteWriteConfiguration{
						RemoteWriteUrl: "https://dummy",
						OIDCConfiguration: &DataPlaneObservabilityOIDCConfiguration{
							AuthorizationServer: "https://dummy",
							Realm:               "dummy",
							Credentials: &DataPlaneObservabilityOIDCCredentials{
								ClientID:     "dummy",
								ClientSecret: "dummy",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should fail when remote write url is provided and oidc configuration is missing credentials",
			fields: fields{
				DataPlaneObservabilityConfig: DataPlaneObservabilityConfig{
					Enabled: true,
					RemoteWriteConfiguration: &DataPlaneObservabilityRemoteWriteConfiguration{
						RemoteWriteUrl: "https://dummy",
						OIDCConfiguration: &DataPlaneObservabilityOIDCConfiguration{
							AuthorizationServer: "https://dummy",
							Realm:               "dummy",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should fail when remote write url is provided and oidc configuration is missing realm",
			fields: fields{
				DataPlaneObservabilityConfig: DataPlaneObservabilityConfig{
					Enabled: true,
					RemoteWriteConfiguration: &DataPlaneObservabilityRemoteWriteConfiguration{
						RemoteWriteUrl: "https://dummy",
						OIDCConfiguration: &DataPlaneObservabilityOIDCConfiguration{
							AuthorizationServer: "https://dummy",
							Credentials: &DataPlaneObservabilityOIDCCredentials{
								ClientID:     "dummy",
								ClientSecret: "dummy",
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should fail when remote write configuration is missing",
			fields: fields{
				DataPlaneObservabilityConfig: DataPlaneObservabilityConfig{
					Enabled:                  true,
					GithubResourcesAuthToken: "dummy",
				},
			},
			wantErr: true,
		},
		{
			name: "should fail when remote write url is missing",
			fields: fields{
				DataPlaneObservabilityConfig: DataPlaneObservabilityConfig{
					Enabled: true,
					RemoteWriteConfiguration: &DataPlaneObservabilityRemoteWriteConfiguration{
						RemoteWriteUrl: "",
					},
					GithubResourcesAuthToken: "dummy",
				},
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			err := tt.fields.DataPlaneObservabilityConfig.validate()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_ObservabilityCloudWatchLoggingConfig_validate(t *testing.T) {
	type fields struct {
		ObservabilityCloudWatchLoggingConfig ObservabilityCloudWatchLoggingConfig
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Validation succeeds if the configuration is valid",
			fields: fields{
				ObservabilityCloudWatchLoggingConfig: ObservabilityCloudWatchLoggingConfig{
					CloudwatchLoggingEnabled: true,
					Credentials: ObservabilityCloudwatchLoggingConfigCredentials{
						AccessKey:       "testaccesskey",
						SecretAccessKey: "testsecretaccesskey",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Validation succeeds if the provided k8s namespace and secret are set explicitely with accepted values",
			fields: fields{
				ObservabilityCloudWatchLoggingConfig: ObservabilityCloudWatchLoggingConfig{
					CloudwatchLoggingEnabled: true,
					Credentials: ObservabilityCloudwatchLoggingConfigCredentials{
						AccessKey:       "testaccesskey",
						SecretAccessKey: "testsecretaccesskey",
					},
					K8sCredentialsSecretName:      defaultObservabilityCloudwatchCredentialsSecretName,
					K8sCredentialsSecretNamespace: defaultObservabilityCloudwatchCredentialsSecretNamespace,
				},
			},
			wantErr: false,
		},
		{
			name: "No error is returned if cloudwatch logging is disabled even if the configuration does not contain the mandatory attributes",
			fields: fields{
				ObservabilityCloudWatchLoggingConfig: ObservabilityCloudWatchLoggingConfig{
					CloudwatchLoggingEnabled: false,
				},
			},
			wantErr: false,
		},
		{
			name: "An error is returned if cloudwatch logging is enabled and the configuration does not contain the mandatory attributes",
			fields: fields{
				ObservabilityCloudWatchLoggingConfig: ObservabilityCloudWatchLoggingConfig{
					CloudwatchLoggingEnabled: true,
				},
			},
			wantErr: true,
		},
		{
			name: "An error is returned if the access key is not provided",
			fields: fields{
				ObservabilityCloudWatchLoggingConfig: ObservabilityCloudWatchLoggingConfig{
					CloudwatchLoggingEnabled: true,
					Credentials: ObservabilityCloudwatchLoggingConfigCredentials{
						SecretAccessKey: "testsecretaccesskey",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "An error is returned if the secret access key is not provided",
			fields: fields{
				ObservabilityCloudWatchLoggingConfig: ObservabilityCloudWatchLoggingConfig{
					CloudwatchLoggingEnabled: true,
					Credentials: ObservabilityCloudwatchLoggingConfigCredentials{
						AccessKey: "testaccesskey",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "An error is returned if the provided k8s secret name is not among the accepted values",
			fields: fields{
				ObservabilityCloudWatchLoggingConfig: ObservabilityCloudWatchLoggingConfig{
					CloudwatchLoggingEnabled: true,
					Credentials: ObservabilityCloudwatchLoggingConfigCredentials{
						AccessKey:       "testaccesskey",
						SecretAccessKey: "testsecretaccesskey",
					},
					K8sCredentialsSecretName: "nonvalidsecretname",
				},
			},
			wantErr: true,
		},
		{
			name: "An error is returned if the provided k8s secret namespace is not among the accepted values",
			fields: fields{
				ObservabilityCloudWatchLoggingConfig: ObservabilityCloudWatchLoggingConfig{
					CloudwatchLoggingEnabled: true,
					Credentials: ObservabilityCloudwatchLoggingConfigCredentials{
						AccessKey:       "testaccesskey",
						SecretAccessKey: "testsecretaccesskey",
					},
					K8sCredentialsSecretNamespace: "nonvalidsecretnamespace",
				},
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			err := tt.fields.ObservabilityCloudWatchLoggingConfig.validate()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}
