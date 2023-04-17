package kafkatlscertmgmt

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/caddyserver/certmagic"
	"github.com/onsi/gomega"
)

func Test_kafkaTLSCertificateManagementService_GetCertificate(t *testing.T) {
	type fields struct {
		storage certmagic.Storage
		config  *config.KafkaTLSCertificateManagementConfig
	}
	type args struct {
		request GetCertificateRequest
	}

	storageWithCerts := newInMemoryStorage(db.NewMockConnectionFactory(nil))
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
				config: &config.KafkaTLSCertificateManagementConfig{
					CertificateManagementStrategy: config.ManualCertificateManagement,
					ManualCertificateManagementConfig: config.ManualCertificateManagementConfig{
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
			name: "returns certificate content from tls certificate configuration when certificate request fields are not defined i.e they are empty",
			fields: fields{
				storage: &certmagic.FileStorage{},
				config: &config.KafkaTLSCertificateManagementConfig{
					CertificateManagementStrategy: config.AutomaticCertificateManagement,
					ManualCertificateManagementConfig: config.ManualCertificateManagementConfig{
						KafkaTLSCert: "cert",
						KafkaTLSKey:  "key",
					},
				},
			},
			args: args{
				GetCertificateRequest{},
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
				config: &config.KafkaTLSCertificateManagementConfig{
					CertificateManagementStrategy: config.AutomaticCertificateManagement,
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
				storage: newInMemoryStorage(db.NewMockConnectionFactory(nil)),
				config: &config.KafkaTLSCertificateManagementConfig{
					CertificateManagementStrategy: config.AutomaticCertificateManagement,
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
		config               *config.KafkaTLSCertificateManagementConfig
		certManagementClient certMagicClientWrapper
		storage              certmagic.Storage
	}
	type args struct {
		domain string
		reason CertificateRevocationReason
	}

	inMemoryStorage := newInMemoryStorage(db.NewMockConnectionFactory(nil))
	certKey := "cert-key"
	privateKey := "private-key"
	_ = inMemoryStorage.Store(context.Background(), certKey, []byte{})
	_ = inMemoryStorage.Store(context.Background(), privateKey, []byte{})

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "should not revoke the certificate if running in manual mode",
			fields: fields{
				config: &config.KafkaTLSCertificateManagementConfig{
					CertificateManagementStrategy: config.ManualCertificateManagement,
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
			name: "should not return an error when storage doesn't contain the certificate to delete",
			fields: fields{
				config: &config.KafkaTLSCertificateManagementConfig{
					CertificateManagementStrategy: config.AutomaticCertificateManagement,
				},
				certManagementClient: &certMagicClientWrapperMock{
					RevokeCertificateFunc: nil,
					GetCerticateRefsFunc: func(domain string) CertificateManagementOutput {
						return CertificateManagementOutput{
							TLSCertRef: "some-certificate-cert-ref",
							TLSKeyRef:  "some-certificate-key-ref",
						}
					},
				},
				storage: inMemoryStorage,
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
				config: &config.KafkaTLSCertificateManagementConfig{
					CertificateManagementStrategy: config.AutomaticCertificateManagement,
				},
				certManagementClient: &certMagicClientWrapperMock{
					RevokeCertificateFunc: func(ctx context.Context, domain string, reason int) error {
						return nil
					},
					GetCerticateRefsFunc: func(domain string) CertificateManagementOutput {
						return CertificateManagementOutput{
							TLSCertRef: certKey,
							TLSKeyRef:  privateKey,
						}
					},
				},
				storage: inMemoryStorage,
			},
			args: args{
				domain: "some-domain",
				reason: AACompromise,
			},
			wantErr: false,
		},
		{
			name: "should return an error when revoking the certificate returns an error",
			fields: fields{
				config: &config.KafkaTLSCertificateManagementConfig{
					CertificateManagementStrategy: config.AutomaticCertificateManagement,
				},
				certManagementClient: &certMagicClientWrapperMock{
					RevokeCertificateFunc: func(ctx context.Context, domain string, reason int) error {
						return errors.New("some error")
					},
					GetCerticateRefsFunc: func(domain string) CertificateManagementOutput {
						return CertificateManagementOutput{
							TLSCertRef: certKey,
							TLSKeyRef:  privateKey,
						}
					},
				},
				storage: inMemoryStorage,
			},
			args: args{
				domain: "some-domain",
				reason: AACompromise,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			certManagementService := &kafkaTLSCertificateManagementService{
				certManagementClient: testcase.fields.certManagementClient,
				config:               testcase.fields.config,
				storage:              testcase.fields.storage,
			}
			err := certManagementService.RevokeCertificate(context.Background(), testcase.args.domain, testcase.args.reason)
			g := gomega.NewWithT(t)
			g.Expect(err != nil).To(gomega.Equal(testcase.wantErr))
		})
	}
}

func Test_kafkaTLSCertificateManagementService_ManageCertificate(t *testing.T) {
	type fields struct {
		config               *config.KafkaTLSCertificateManagementConfig
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
				config: &config.KafkaTLSCertificateManagementConfig{
					CertificateManagementStrategy: config.ManualCertificateManagement,
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
				config: &config.KafkaTLSCertificateManagementConfig{
					CertificateManagementStrategy: config.AutomaticCertificateManagement,
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
				config: &config.KafkaTLSCertificateManagementConfig{
					CertificateManagementStrategy: config.AutomaticCertificateManagement,
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
		config *config.KafkaTLSCertificateManagementConfig
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "return true if external certificate is enabled",
			fields: fields{
				config: &config.KafkaTLSCertificateManagementConfig{
					EnableKafkaExternalCertificate: true,
				},
			},
			want: true,
		},
		{
			name: "return false if external certificate is not enabled",
			fields: fields{
				config: &config.KafkaTLSCertificateManagementConfig{
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
		config *config.KafkaTLSCertificateManagementConfig
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "return true if the certificate management strategy automatic",
			fields: fields{
				config: &config.KafkaTLSCertificateManagementConfig{
					CertificateManagementStrategy: config.AutomaticCertificateManagement,
				},
			},
			want: true,
		},
		{
			name: "return false if the certificate management strategy manual",
			fields: fields{
				config: &config.KafkaTLSCertificateManagementConfig{
					CertificateManagementStrategy: config.ManualCertificateManagement,
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

func TestGetCertificateRequest_areCertRefsDefined(t *testing.T) {
	type fields struct {
		TLSCertRef string
		TLSKeyRef  string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "returns false if either of the refs is not defined",
			fields: fields{
				TLSCertRef: "",
				TLSKeyRef:  "dsldkslds",
			},
			want: false,
		},
		{
			name: "returns true if all the refs are defined",
			fields: fields{
				TLSCertRef: "fjkjds",
				TLSKeyRef:  "dsldkslds",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			req := GetCertificateRequest{
				TLSCertRef: testcase.fields.TLSCertRef,
				TLSKeyRef:  testcase.fields.TLSKeyRef,
			}
			got := req.areCertRefsDefined()
			g.Expect(got).To(gomega.Equal(testcase.want))
		})
	}
}
