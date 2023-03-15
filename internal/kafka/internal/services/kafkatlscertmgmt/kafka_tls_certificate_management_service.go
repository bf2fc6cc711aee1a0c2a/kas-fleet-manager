package kafkatlscertmgmt

import (
	"context"
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/caddyserver/certmagic"
	"github.com/libdns/route53"
)

// CertificateManagementOutput is the output indicating the certificates references
type CertificateManagementOutput struct {
	TLSCertRef string
	TLSKeyRef  string
}

// GetCertificateRequest is the certificate request object.
type GetCertificateRequest struct {
	TLSCertRef string
	TLSKeyRef  string
}

// areCertRefsDefined returns true if all certificate references are defined.
// otherwise returns false
func (req GetCertificateRequest) areCertRefsDefined() bool {
	return !(shared.StringEmpty(req.TLSCertRef) || shared.StringEmpty(req.TLSKeyRef))
}

// Certificate is the content of the certificate
type Certificate struct {
	TLSCert string
	TLSKey  string
}

//go:generate moq -out kafka_tls_certificate_management_service_moq.go . KafkaTLSCertificateManagementService
type KafkaTLSCertificateManagementService interface {
	// ManageCertificate manages wildcard tls certificate of a given domain automatically.
	// If the certificate does not exists, it generates them
	// If they exists, checks if they've expired and in this case it renews them before they do.
	// It returns the keys referencing the wildcard certificates location in the Storage
	ManageCertificate(ctx context.Context, domain string) (CertificateManagementOutput, error)

	// GetCertificate returns the tls certificate given the request.
	// The certificate is returned from the underlying certificate storage when certificate management is automatic
	// and that the certificate keys are defined i.e non empty string.
	// Otherwise, the certificate is returned from the manual tls configuration files
	GetCertificate(ctx context.Context, request GetCertificateRequest) (Certificate, error)

	// RevokeCertificate revoke the certificate of given domain with a given reason
	RevokeCertificate(ctx context.Context, domain string, reason CertificateRevocationReason) error

	// IsKafkaExternalCertificateEnabled returns whether kafka external certificate is enabled
	IsKafkaExternalCertificateEnabled() bool

	// IsAutomaticCertificateManagementEnabled returns whether automatic certificate management is enabled
	IsAutomaticCertificateManagementEnabled() bool
}

// certificateManagementClientWrapper wrapps the certmagic.Config (https://github.com/caddyserver/certmagic/blob/91cbe177810730b91352b5069474f1eca8c0a9a0/config.go) struct used for certificate management.
// The intention is that we can easily unit test the kafkaTLSCertificateManagementService which otherwise would have been difficult with the certmagic.Config.
// certificateManagementClientWrapper being a wrapper, just proxies all calls to certmagic.Config
//
//go:generate moq -out certmagic_client_wrapper_moq.go . certMagicClientWrapper
type certMagicClientWrapper interface {
	//ManageCertificate manages certificate (generation and renewals) by relying on the certificate management library
	ManageCertificate(ctx context.Context, domainNames []string) error
	//RevokeCertificate revoke the certificate for the given domain. This relies on the certificate management library revocation method.
	RevokeCertificate(ctx context.Context, domain string, reason int) error
	//GetCerticateRefs returns the certificate's references keys in the Storage for a given domain
	GetCerticateRefs(domain string) CertificateManagementOutput
}

type wrapper struct {
	wrappedClient *certmagic.Config
}

func (w wrapper) ManageCertificate(ctx context.Context, domainNames []string) error {
	return w.wrappedClient.ManageSync(ctx, domainNames)
}

func (w wrapper) RevokeCertificate(ctx context.Context, domain string, reason int) error {
	return w.wrappedClient.RevokeCert(ctx, domain, reason, false)
}

func (w wrapper) GetCerticateRefs(domain string) CertificateManagementOutput {
	issuer := w.wrappedClient.Issuers[0]
	issuerKey := issuer.IssuerKey()
	return CertificateManagementOutput{
		TLSCertRef: certmagic.StorageKeys.SiteCert(issuerKey, domain),
		TLSKeyRef:  certmagic.StorageKeys.SitePrivateKey(issuerKey, domain),
	}
}

type kafkaTLSCertificateManagementService struct {
	config               *config.KafkaTLSCertificateManagementConfig
	storage              certmagic.Storage
	certManagementClient certMagicClientWrapper
}

func (certManagementService *kafkaTLSCertificateManagementService) GetCertificate(ctx context.Context, request GetCertificateRequest) (Certificate, error) {
	if !certManagementService.IsAutomaticCertificateManagementEnabled() || !request.areCertRefsDefined() {
		return Certificate{
			TLSCert: certManagementService.config.ManualCertificateManagementConfig.KafkaTLSCert,
			TLSKey:  certManagementService.config.ManualCertificateManagementConfig.KafkaTLSKey,
		}, nil
	}

	tlsCertValue, err := certManagementService.storage.Load(ctx, request.TLSCertRef)

	if err != nil {
		return Certificate{}, err
	}

	tlsKeyValue, err := certManagementService.storage.Load(ctx, request.TLSKeyRef)

	if err != nil {
		return Certificate{}, err
	}

	return Certificate{
		TLSCert: string(tlsCertValue),
		TLSKey:  string(tlsKeyValue),
	}, nil
}

func (certManagementService *kafkaTLSCertificateManagementService) ManageCertificate(ctx context.Context, domain string) (CertificateManagementOutput, error) {
	if certManagementService.config.CertificateManagementStrategy == config.ManualCertificateManagement {
		return CertificateManagementOutput{}, nil // the certificate is managed manually in manual mode
	}

	// We ask the wildcard certificate of the given domain
	// see ADR-90 https://github.com/bf2fc6cc711aee1a0c2a/architecture/blob/main/_adr/90/index.adoc for context
	wildcardDomain := fmt.Sprintf("*.%s", domain)
	err := certManagementService.certManagementClient.ManageCertificate(ctx, []string{wildcardDomain})

	if err != nil {
		return CertificateManagementOutput{}, err
	}

	return certManagementService.certManagementClient.GetCerticateRefs(wildcardDomain), nil
}

func (certManagementService *kafkaTLSCertificateManagementService) RevokeCertificate(ctx context.Context, domain string, reason CertificateRevocationReason) error {
	if certManagementService.config.CertificateManagementStrategy == config.ManualCertificateManagement {
		return nil // the certificate is revoked manually in manual mode
	}

	// We revoke the wildcard certificate of the given domain
	// see ADR-90 https://github.com/bf2fc6cc711aee1a0c2a/architecture/blob/main/_adr/90/index.adoc for context
	return certManagementService.certManagementClient.RevokeCertificate(ctx, fmt.Sprintf("*.%s", domain), reason.AsInt())
}

func (certManagementService *kafkaTLSCertificateManagementService) IsKafkaExternalCertificateEnabled() bool {
	return certManagementService.config.EnableKafkaExternalCertificate
}

func (certManagementService *kafkaTLSCertificateManagementService) IsAutomaticCertificateManagementEnabled() bool {
	return certManagementService.config.CertificateManagementStrategy == config.AutomaticCertificateManagement
}

func NewKafkaTLSCertificateManagementService(
	awsConfig *config.AWSConfig,
	kafkaTLSCertificateManagementConfig *config.KafkaTLSCertificateManagementConfig,
) (KafkaTLSCertificateManagementService, error) {
	var storage certmagic.Storage
	var err error
	switch kafkaTLSCertificateManagementConfig.StorageType {
	case config.FileTLSCertStorageType:
		storage = &certmagic.FileStorage{
			Path: "secrets/tls/",
		}
	case config.InMemoryTLSCertStorageType:
		storage = newInMemoryStorage()
	case config.SecureTLSCertStorageType:
		storage, err = newSecureStorage(awsConfig, kafkaTLSCertificateManagementConfig.AutomaticCertificateManagementConfig)
	}

	var certManagementClient certMagicClientWrapper
	if kafkaTLSCertificateManagementConfig.CertificateManagementStrategy == config.AutomaticCertificateManagement {
		certManagementClient = wrapper{
			wrappedClient: createCertMagicClient(awsConfig, kafkaTLSCertificateManagementConfig, storage),
		}
	}

	return &kafkaTLSCertificateManagementService{
		storage:              storage,
		config:               kafkaTLSCertificateManagementConfig,
		certManagementClient: certManagementClient,
	}, err
}

func createCertMagicClient(awsConfig *config.AWSConfig,
	kafkaTLSCertificateManagementConfig *config.KafkaTLSCertificateManagementConfig,
	storage certmagic.Storage) *certmagic.Config {
	provider := &route53.Provider{
		WaitForPropagation: false,
		AccessKeyId:        awsConfig.Route53.AccessKey,
		SecretAccessKey:    awsConfig.Route53.SecretAccessKey,
	}

	certmagic.Default.RenewalWindowRatio = kafkaTLSCertificateManagementConfig.AutomaticCertificateManagementConfig.RenewalWindowRatio
	certmagic.Default.MustStaple = kafkaTLSCertificateManagementConfig.AutomaticCertificateManagementConfig.MustStaple

	magic := certmagic.NewDefault()

	magic.Storage = storage

	myACME := certmagic.NewACMEIssuer(magic, certmagic.ACMEIssuer{
		CA:                      kafkaTLSCertificateManagementConfig.CertificateAuthorityEndpoint,
		Email:                   kafkaTLSCertificateManagementConfig.AutomaticCertificateManagementConfig.EmailToSendNotificationTo,
		Agreed:                  true,
		DisableHTTPChallenge:    true,
		DisableTLSALPNChallenge: true,
		AccountKeyPEM:           kafkaTLSCertificateManagementConfig.AutomaticCertificateManagementConfig.AcmeIssuerAccountKey,
		DNS01Solver: &certmagic.DNS01Solver{
			DNSProvider: provider,
		},
	})

	magic.Issuers = []certmagic.Issuer{myACME}
	magic.KeySource = certmagic.StandardKeyGenerator{KeyType: certmagic.RSA4096}

	return magic
}
