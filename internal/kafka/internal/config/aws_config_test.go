package config

import (
	"testing"
	"time"

	"github.com/onsi/gomega"
)

func Test_NewAwsConfig(t *testing.T) {
	tests := []struct {
		name string
		want *AWSConfig
	}{
		{
			name: "should return NewAWSConfig",
			want: &AWSConfig{
				ConfigForOSDClusterCreation: awsConfigForOSDClusterCreation{
					accountIDFilePath:       "secrets/aws.accountid",
					accessKeyFilePath:       "secrets/aws.accesskey",
					secretAccessKeyFilePath: "secrets/aws.secretaccesskey",
				},
				Route53: awsRoute53Config{
					accessKeyFilePath:       "secrets/aws.route53accesskey",
					secretAccessKeyFilePath: "secrets/aws.route53secretaccesskey",
					RecordTTL:               300 * time.Second,
				},
				SecretManager: awsSecretManagerConfig{
					accessKeyFilePath:       "secrets/aws-secret-manager/aws_access_key_id",
					secretAccessKeyFilePath: "secrets/aws-secret-manager/aws_secret_access_key",
					Region:                  "us-east-1",
					SecretPrefix:            "kas-fleet-manager",
				},
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(NewAWSConfig()).To(gomega.Equal(tt.want))
		})
	}
}

func Test_ReadFilesAWSConfig(t *testing.T) {
	type fields struct {
		config *AWSConfig
	}

	tests := []struct {
		name     string
		fields   fields
		modifyFn func(config *AWSConfig)
		wantErr  bool
	}{
		{
			name: "should return no error when running ReadFiles with default AWSConfig",
			fields: fields{
				config: NewAWSConfig(),
			},
			wantErr: false,
		},
		{
			name: "should return an error with misconfigured AccountIDFile",
			fields: fields{
				config: NewAWSConfig(),
			},
			modifyFn: func(config *AWSConfig) {
				config.ConfigForOSDClusterCreation.accountIDFilePath = "invalid"
			},
			wantErr: true,
		},
		{
			name: "should return an error with misconfigured AccessKeyFile",
			fields: fields{
				config: NewAWSConfig(),
			},
			modifyFn: func(config *AWSConfig) {
				config.ConfigForOSDClusterCreation.accessKeyFilePath = "invalid"
			},
			wantErr: true,
		},
		{
			name: "should return an error with misconfigured SecretAccessKeyFile",
			fields: fields{
				config: NewAWSConfig(),
			},
			modifyFn: func(config *AWSConfig) {
				config.ConfigForOSDClusterCreation.secretAccessKeyFilePath = "invalid"
			},
			wantErr: true,
		},
		{
			name: "should return an error with misconfigured Route53AccessKeyFile",
			fields: fields{
				config: NewAWSConfig(),
			},
			modifyFn: func(config *AWSConfig) {
				config.Route53.accessKeyFilePath = "invalid"
			},
			wantErr: true,
		},
		{
			name: "should return an error with misconfigured Route53SecretAccessKeyFile",
			fields: fields{
				config: NewAWSConfig(),
			},
			modifyFn: func(config *AWSConfig) {
				config.Route53.secretAccessKeyFilePath = "invalid"
			},
			wantErr: true,
		},
		{
			name: "should return an error with misconfigured SecretManagerAccessKeyFile",
			fields: fields{
				config: NewAWSConfig(),
			},
			modifyFn: func(config *AWSConfig) {
				config.SecretManager.accessKeyFilePath = "invalid"
			},
			wantErr: true,
		},
		{
			name: "should return an error with misconfigured SecretManagerSecretAccessKeyFile",
			fields: fields{
				config: NewAWSConfig(),
			},
			modifyFn: func(config *AWSConfig) {
				config.SecretManager.secretAccessKeyFilePath = "invalid"
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
