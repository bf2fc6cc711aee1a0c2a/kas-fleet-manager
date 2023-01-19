package kafka_tls_certificate_management

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/caddyserver/certmagic"
	"github.com/onsi/gomega"
)

func Test_kafkaTLSCertificateManagementService_GetCertificate(t *testing.T) {
	type fields struct {
		storage certmagic.Storage
		config  *KafkaTLSCertificateManagementConfig
	}
	type args struct {
		request GetCertificateRequest
	}

	storageWithCerts := newInMemoryStorage()
	crtRef := "some-crt-ref"
	keyRef := "some-key-ref"

	_ = storageWithCerts.Store(context.TODO(), crtRef, []byte("some-crt-from-storage"))
	_ = storageWithCerts.Store(context.TODO(), keyRef, []byte("some-key-from-storage"))

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    Certificate
		wantErr bool
	}{
		{
			name: "returns certificate content from tls certificate configuration when in manual mode",
			fields: fields{
				storage: &certmagic.FileStorage{},
				config: &KafkaTLSCertificateManagementConfig{
					CertificateManagementStrategy: manualCertificateManagement,
					ManualCertificateManagementConfig: ManualCertificateManagementConfig{
						KafkaTLSCert: "cert",
						KafkaTLSKey:  "key",
					},
				},
			},
			args: args{
				GetCertificateRequest{
					TLSCertRef: "some-ref",
					TLSKeyRef:  "some-key",
				},
			},
			want: Certificate{
				TLSCert: "cert",
				TLSKey:  "key",
			},
			wantErr: false,
		},
		{
			name: "returns certificate content from storage when in auto mode",
			fields: fields{
				storage: storageWithCerts,
				config: &KafkaTLSCertificateManagementConfig{
					CertificateManagementStrategy: automaticCertificateManagement,
				},
			},
			args: args{
				GetCertificateRequest{
					TLSCertRef: crtRef,
					TLSKeyRef:  keyRef,
				},
			},
			want: Certificate{
				TLSCert: "some-crt-from-storage",
				TLSKey:  "some-key-from-storage",
			},
			wantErr: false,
		},
		{
			name: "should return an error when loading from the storage returns an error",
			fields: fields{
				storage: newInMemoryStorage(),
				config: &KafkaTLSCertificateManagementConfig{
					CertificateManagementStrategy: automaticCertificateManagement,
				},
			},
			args: args{
				GetCertificateRequest{
					TLSCertRef: crtRef,
					TLSKeyRef:  keyRef,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			k := &kafkaTLSCertificateManagementService{
				storage: testcase.fields.storage,
				config:  testcase.fields.config,
			}

			certificate, err := k.GetCertificate(context.Background(), testcase.args.request)
			g := gomega.NewWithT(t)
			g.Expect(err != nil).To(gomega.Equal(testcase.wantErr))
			g.Expect(certificate).To(gomega.Equal(testcase.want))
		})
	}
}

func Test_kafkaTLSCertificateManagementService_RevokeCertificate(t *testing.T) {
	type fields struct {
		config               *KafkaTLSCertificateManagementConfig
		certManagementClient certMagicClientWrapper
	}
	type args struct {
		domain string
		reason CertificateRevocationReason
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "should return an error when revoking the certificate returns an error",
			fields: fields{
				config: &KafkaTLSCertificateManagementConfig{
					CertificateManagementStrategy: automaticCertificateManagement,
				},
				certManagementClient: &certMagicClientWrapperMock{
					RevokeCertificateFunc: func(ctx context.Context, domain string, reason int) error {
						return errors.New("some error")
					},
				},
			},
			args: args{
				domain: "some-domain",
				reason: AACompromise,
			},
			wantErr: true,
		},
		{
			name: "should not revoke the certificae if running in manual mode",
			fields: fields{
				config: &KafkaTLSCertificateManagementConfig{
					CertificateManagementStrategy: manualCertificateManagement,
				},
				certManagementClient: &certMagicClientWrapperMock{
					RevokeCertificateFunc: nil, // it should never be called
				},
			},
			args: args{
				domain: "some-domain",
				reason: AACompromise,
			},
			wantErr: false,
		},
		{
			name: "should succeed when revoking the certificate is successfully",
			fields: fields{
				config: &KafkaTLSCertificateManagementConfig{
					CertificateManagementStrategy: automaticCertificateManagement,
				},
				certManagementClient: &certMagicClientWrapperMock{
					RevokeCertificateFunc: func(ctx context.Context, domain string, reason int) error {
						return nil
					},
				},
			},
			args: args{
				domain: "some-domain",
				reason: AACompromise,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			certManagementService := &kafkaTLSCertificateManagementService{
				certManagementClient: testcase.fields.certManagementClient,
				config:               testcase.fields.config,
			}

			err := certManagementService.RevokeCertificate(context.Background(), testcase.args.domain, testcase.args.reason)
			g := gomega.NewWithT(t)
			g.Expect(err != nil).To(gomega.Equal(testcase.wantErr))
		})
	}
}

func Test_kafkaTLSCertificateManagementService_ManageCertificate(t *testing.T) {
	type fields struct {
		config               *KafkaTLSCertificateManagementConfig
		certManagementClient certMagicClientWrapper
	}
	type args struct {
		domain string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    CertificateManagementOutput
		wantErr bool
	}{
		{
			name: "should not manage the certificate if in manual mode",
			fields: fields{
				config: &KafkaTLSCertificateManagementConfig{
					CertificateManagementStrategy: manualCertificateManagement,
				},
				certManagementClient: &certMagicClientWrapperMock{
					ManageCertificateFunc: nil, // it should never be called
				},
			},
			args: args{
				domain: "some-domain",
			},
			wantErr: false,
		},
		{
			name: "should return an error when managing the certificate returns an error",
			fields: fields{
				config: &KafkaTLSCertificateManagementConfig{
					CertificateManagementStrategy: automaticCertificateManagement,
				},
				certManagementClient: &certMagicClientWrapperMock{
					ManageCertificateFunc: func(ctx context.Context, domainNames []string) error {
						return errors.New("some errors")
					},
				},
			},
			args: args{
				domain: "some-domain",
			},
			wantErr: true,
		},
		{
			name: "should return certificate key refs",
			fields: fields{
				config: &KafkaTLSCertificateManagementConfig{
					CertificateManagementStrategy: automaticCertificateManagement,
				},
				certManagementClient: &certMagicClientWrapperMock{
					ManageCertificateFunc: func(ctx context.Context, domainNames []string) error {
						if !strings.HasPrefix(domainNames[0], "*") {
							panic("wildcard certificate has to be asked as per ADR-90: https://github.com/bf2fc6cc711aee1a0c2a/architecture/blob/main/_adr/90/index.adoc")
						}
						return nil
					},
					GetCerticateRefsFunc: func(domain string) CertificateManagementOutput {
						return CertificateManagementOutput{
							TLSCertRef: "certificates/acme-v02.api.letsencrypt.org-directory/wildcard_.some-domain/wildcard_.some-domain.crt",
							TLSKeyRef:  "certificates/acme-v02.api.letsencrypt.org-directory/wildcard_.some-domain/wildcard_.some-domain.key",
						}
					},
				},
			},
			args: args{
				domain: "some-domain",
			},
			wantErr: false,
			want: CertificateManagementOutput{
				TLSCertRef: "certificates/acme-v02.api.letsencrypt.org-directory/wildcard_.some-domain/wildcard_.some-domain.crt",
				TLSKeyRef:  "certificates/acme-v02.api.letsencrypt.org-directory/wildcard_.some-domain/wildcard_.some-domain.key",
			},
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			certManagementService := &kafkaTLSCertificateManagementService{
				config:               testcase.fields.config,
				certManagementClient: testcase.fields.certManagementClient,
			}
			output, err := certManagementService.ManageCertificate(context.Background(), testcase.args.domain)
			g := gomega.NewWithT(t)
			g.Expect(err != nil).To(gomega.Equal(testcase.wantErr))
			g.Expect(output).To(gomega.Equal(testcase.want))
		})
	}
}

func Test_kafkaTLSCertificateManagementService_IsKafkaExternalCertificateEnabled(t *testing.T) {
	type fields struct {
		config *KafkaTLSCertificateManagementConfig
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "return true if external certificate is enabled",
			fields: fields{
				config: &KafkaTLSCertificateManagementConfig{
					EnableKafkaExternalCertificate: true,
				},
			},
			want: true,
		},
		{
			name: "return false if external certificate is not enabled",
			fields: fields{
				config: &KafkaTLSCertificateManagementConfig{
					EnableKafkaExternalCertificate: false,
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			certManagementService := &kafkaTLSCertificateManagementService{
				config: testcase.fields.config,
			}
			g := gomega.NewWithT(t)
			enabled := certManagementService.IsKafkaExternalCertificateEnabled()
			g.Expect(enabled).To(gomega.Equal(testcase.want))
		})
	}
}

func Test_kafkaTLSCertificateManagementService_IsAutomaticCertificateManagementEnabled(t *testing.T) {
	type fields struct {
		config *KafkaTLSCertificateManagementConfig
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "return true if the certificate management strategy automatic",
			fields: fields{
				config: &KafkaTLSCertificateManagementConfig{
					CertificateManagementStrategy: automaticCertificateManagement,
				},
			},
			want: true,
		},
		{
			name: "return false if the certificate management strategy manual",
			fields: fields{
				config: &KafkaTLSCertificateManagementConfig{
					CertificateManagementStrategy: manualCertificateManagement,
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			certManagementService := &kafkaTLSCertificateManagementService{
				config: testcase.fields.config,
			}
			g := gomega.NewWithT(t)
			enabled := certManagementService.IsAutomaticCertificateManagementEnabled()
			g.Expect(enabled).To(gomega.Equal(testcase.want))
		})
	}
}
