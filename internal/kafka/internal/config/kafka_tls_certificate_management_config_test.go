package config

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/onsi/gomega"
)

func TestKafkaTLSCertificateManagementConfig_Validate(t *testing.T) {
	type fields struct {
		StorageType                    string
		CertificateManagementStrategy  string
		RenewalWindowRatio             float64
		EmailToSendNotificationTo      string
		AcmeIssuerAccountKeyPEMFile    string
		EnableKafkaExternalCertificate bool
		KafkaTLSCertFile               string
		KafkaTLSKeyFile                string
	}

	type args struct {
		env *environments.Env
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "should return an error when storage type is invalid",
			fields: fields{
				StorageType:                   "some-storage-type",
				CertificateManagementStrategy: ManualCertificateManagement,
			},
			args: args{
				&environments.Env{},
			},
			wantErr: true,
		},
		{
			name: "should return an error when certificate management strategy is invalid",
			fields: fields{
				StorageType:                   "secure-storage",
				CertificateManagementStrategy: "fake-strategy",
			},
			args: args{
				&environments.Env{},
			},
			wantErr: true,
		},
		{
			name: "should return an error when renewal window ratio is less than 0",
			fields: fields{
				StorageType:                    "secure-storage",
				CertificateManagementStrategy:  AutomaticCertificateManagement,
				RenewalWindowRatio:             -0.1,
				EmailToSendNotificationTo:      "some-email@gmail.com",
				AcmeIssuerAccountKeyPEMFile:    "some-key",
				EnableKafkaExternalCertificate: true,
			},
			args: args{
				&environments.Env{},
			},
			wantErr: true,
		},
		{
			name: "should return an error when renewal window ratio is greater than 1",
			fields: fields{
				StorageType:                    "secure-storage",
				CertificateManagementStrategy:  AutomaticCertificateManagement,
				RenewalWindowRatio:             1.2,
				EmailToSendNotificationTo:      "some-email@gmail.com",
				AcmeIssuerAccountKeyPEMFile:    "some-key",
				EnableKafkaExternalCertificate: true,
			},
			args: args{
				&environments.Env{},
			},
			wantErr: true,
		},
		{
			name: "should return an error when email is invalid",
			fields: fields{
				StorageType:                    "secure-storage",
				CertificateManagementStrategy:  AutomaticCertificateManagement,
				RenewalWindowRatio:             0.2,
				EmailToSendNotificationTo:      "some-email@gmail",
				AcmeIssuerAccountKeyPEMFile:    "some-key",
				EnableKafkaExternalCertificate: true,
			},
			args: args{
				&environments.Env{},
			},
			wantErr: true,
		},
		{
			name: "should return an error when account key is missing",
			fields: fields{
				StorageType:                    "secure-storage",
				CertificateManagementStrategy:  AutomaticCertificateManagement,
				RenewalWindowRatio:             0.2,
				EmailToSendNotificationTo:      "some-email@gmail.com",
				AcmeIssuerAccountKeyPEMFile:    "",
				EnableKafkaExternalCertificate: true,
			},
			args: args{
				&environments.Env{},
			},
			wantErr: true,
		},
		{
			name: "should not return an error when configuration is valid for manual management of certificates",
			fields: fields{
				StorageType:                   "in-memory",
				CertificateManagementStrategy: ManualCertificateManagement,
			},
			args: args{
				&environments.Env{},
			},
			wantErr: false,
		},
		{
			name: "should not return an error when manual configuration is invalid but external certificate is disabled",
			fields: fields{
				StorageType:                    "in-memory",
				CertificateManagementStrategy:  ManualCertificateManagement,
				KafkaTLSCertFile:               "",
				KafkaTLSKeyFile:                "",
				EnableKafkaExternalCertificate: false,
			},
			args: args{
				&environments.Env{},
			},
			wantErr: false,
		},
		{
			name: "should return an error when manual configuration is invalid",
			fields: fields{
				StorageType:                    "in-memory",
				CertificateManagementStrategy:  ManualCertificateManagement,
				KafkaTLSCertFile:               "",
				KafkaTLSKeyFile:                "",
				EnableKafkaExternalCertificate: true,
			},
			args: args{
				&environments.Env{},
			},
			wantErr: true,
		},
		{
			name: "should not return an error when configuration is valid for automatic management of certificates",
			fields: fields{
				StorageType:                    "secure-storage",
				CertificateManagementStrategy:  AutomaticCertificateManagement,
				RenewalWindowRatio:             0.2,
				EmailToSendNotificationTo:      "some-email@gmail.com",
				AcmeIssuerAccountKeyPEMFile:    "some-keyfile.pem",
				EnableKafkaExternalCertificate: true,
			},
			args: args{
				&environments.Env{},
			},
			wantErr: false,
		},
		{
			name: "should not return an error when configuration is invalid for automatic management of certificates but external certificate flag is not enabled",
			fields: fields{
				StorageType:                    "secure-storage",
				CertificateManagementStrategy:  AutomaticCertificateManagement,
				RenewalWindowRatio:             2.5,
				EmailToSendNotificationTo:      "some-email.gmail.com",
				AcmeIssuerAccountKeyPEMFile:    "some-keyfile.pem",
				EnableKafkaExternalCertificate: false,
			},
			args: args{
				&environments.Env{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			c := &KafkaTLSCertificateManagementConfig{
				StorageType:                    testcase.fields.StorageType,
				CertificateManagementStrategy:  testcase.fields.CertificateManagementStrategy,
				EnableKafkaExternalCertificate: testcase.fields.EnableKafkaExternalCertificate,
				AutomaticCertificateManagementConfig: AutomaticCertificateManagementConfig{
					RenewalWindowRatio:           testcase.fields.RenewalWindowRatio,
					AcmeIssuerAccountKeyFilePath: testcase.fields.AcmeIssuerAccountKeyPEMFile,
					EmailToSendNotificationTo:    testcase.fields.EmailToSendNotificationTo,
				},
			}

			err := c.Validate(testcase.args.env)
			g.Expect(err != nil).To(gomega.Equal(testcase.wantErr))
		})
	}
}
