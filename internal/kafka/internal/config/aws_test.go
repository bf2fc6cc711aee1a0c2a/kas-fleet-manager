package config

import (
	"testing"

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
				AccountIDFile:              "secrets/aws.accountid",
				AccessKeyFile:              "secrets/aws.accesskey",
				SecretAccessKeyFile:        "secrets/aws.secretaccesskey",
				Route53AccessKeyFile:       "secrets/aws.route53accesskey",
				Route53SecretAccessKeyFile: "secrets/aws.route53secretaccesskey",
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
				config.AccountIDFile = "invalid"
			},
			wantErr: true,
		},
		{
			name: "should return an error with misconfigured AccessKeyFile",
			fields: fields{
				config: NewAWSConfig(),
			},
			modifyFn: func(config *AWSConfig) {
				config.AccessKeyFile = "invalid"
			},
			wantErr: true,
		},
		{
			name: "should return an error with misconfigured SecretAccessKeyFile",
			fields: fields{
				config: NewAWSConfig(),
			},
			modifyFn: func(config *AWSConfig) {
				config.SecretAccessKeyFile = "invalid"
			},
			wantErr: true,
		},
		{
			name: "should return an error with misconfigured Route53AccessKeyFile",
			fields: fields{
				config: NewAWSConfig(),
			},
			modifyFn: func(config *AWSConfig) {
				config.Route53AccessKeyFile = "invalid"
			},
			wantErr: true,
		},
		{
			name: "should return an error with misconfigured Route53SecretAccessKeyFile",
			fields: fields{
				config: NewAWSConfig(),
			},
			modifyFn: func(config *AWSConfig) {
				config.Route53SecretAccessKeyFile = "invalid"
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
