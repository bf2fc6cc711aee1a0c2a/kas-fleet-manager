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
			name: "should return an error with misconfigured LogsClientIdFile when reading observatorium config files",
			fields: fields{
				config: NewObservabilityConfigurationConfig(),
			},
			modifyFn: func(config *ObservabilityConfiguration) {
				config.LogsClientIdFile = "invalid"
			},
			wantErr: true,
		},
		{
			name: "should return no error with with all config provided but ObservabilityConfigAccessToken(File)",
			fields: fields{
				config: NewObservabilityConfigurationConfig(),
			},
			modifyFn: func(config *ObservabilityConfiguration) {
				config.ObservabilityConfigAccessToken = ""
				config.ObservabilityConfigAccessTokenFile = ""
			},
			wantErr: false,
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

func Test_ReadObservatoriumConfigFiles_ObservabilityConfiguration(t *testing.T) {
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
			name: "should return no error when running ReadObservatoriumConfigFiles with default ObservabilityConfiguration",
			fields: fields{
				config: NewObservabilityConfigurationConfig(),
			},
			wantErr: false,
		},
		{
			name: "should return an error with misconfigured LogsClientIdFile",
			fields: fields{
				config: NewObservabilityConfigurationConfig(),
			},
			modifyFn: func(config *ObservabilityConfiguration) {
				config.LogsClientIdFile = "invalid"
			},
			wantErr: true,
		},
		{
			name: "should return an error with misconfigured LogsSecretFile",
			fields: fields{
				config: NewObservabilityConfigurationConfig(),
			},
			modifyFn: func(config *ObservabilityConfiguration) {
				config.LogsSecretFile = "invalid"
			},
			wantErr: true,
		},
		{
			name: "should return an error with misconfigured MetricsClientIdFile",
			fields: fields{
				config: NewObservabilityConfigurationConfig(),
			},
			modifyFn: func(config *ObservabilityConfiguration) {
				config.MetricsClientIdFile = "invalid"
			},
			wantErr: true,
		},
		{
			name: "should return an error with misconfigured MetricsSecretFile when reading observatorium config files",
			fields: fields{
				config: NewObservabilityConfigurationConfig(),
			},
			modifyFn: func(config *ObservabilityConfiguration) {
				config.MetricsSecretFile = "invalid"
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
			g.Expect(config.ReadObservatoriumConfigFiles() != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}
