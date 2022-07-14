package config

import (
	"testing"

	"github.com/onsi/gomega"
)

func Test_NewKafkaConfig(t *testing.T) {
	tests := []struct {
		name string
		want *KafkaConfig
	}{
		{
			name: "should return NewKafkaConfig",
			want: &KafkaConfig{
				KafkaTLSCertFile:               "secrets/kafka-tls.crt",
				KafkaTLSKeyFile:                "secrets/kafka-tls.key",
				EnableKafkaExternalCertificate: false,
				KafkaDomainName:                "kafka.bf2.dev",
				KafkaLifespan:                  NewKafkaLifespanConfig(),
				Quota:                          NewKafkaQuotaConfig(),
				BrowserUrl:                     "http://localhost:8080/",
				SupportedInstanceTypes:         NewKafkaSupportedInstanceTypesConfig(),
				EnableKafkaOwnerConfig:         false,
				KafkaOwnerListFile:             "config/kafka-owner-list.yaml",
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(NewKafkaConfig()).To(gomega.Equal(tt.want))
		})
	}
}

func Test_ReadFilesKafkaConfig(t *testing.T) {
	type fields struct {
		config *KafkaConfig
	}

	tests := []struct {
		name     string
		fields   fields
		modifyFn func(config *KafkaConfig)
		wantErr  bool
	}{
		{
			name: "should return no error when running ReadFiles with default KafkaConfig",
			fields: fields{
				config: NewKafkaConfig(),
			},
			wantErr: false,
		},
		{
			name: "should return an error with misconfigured KafkaTLSCertFile",
			fields: fields{
				config: NewKafkaConfig(),
			},
			modifyFn: func(config *KafkaConfig) {
				config.KafkaTLSCertFile = "invalid"
			},
			wantErr: true,
		},
		{
			name: "should return an error with misconfigured KafkaTLSKeyFile",
			fields: fields{
				config: NewKafkaConfig(),
			},
			modifyFn: func(config *KafkaConfig) {
				config.KafkaTLSKeyFile = "invalid"
			},
			wantErr: true,
		},
		{
			name: "should return an error with misconfigured SupportedInstanceTypes",
			fields: fields{
				config: NewKafkaConfig(),
			},
			modifyFn: func(config *KafkaConfig) {
				config.SupportedInstanceTypes.ConfigurationFile = "invalid"
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
