package mocks

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
	v1 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/managedkafkas.managedkafka.bf2.org/v1"
)

var (
	kafkaVersion     = "2.8.0"
	ibpVersion       = "2.8"
	strimziVersion   = "2.8.0"
	dummyCert        = "dummy cert"
	dummyKey         = "dummy key"
	trustedCertValue = "trusted cert"
)

func BuildManagedKafka(modifyFn func(managedKafka *v1.ManagedKafka)) *v1.ManagedKafka {
	managedKafka := &v1.ManagedKafka{
		Spec: v1.ManagedKafkaSpec{
			Versions: v1.VersionsSpec{
				Kafka:    kafkaVersion,
				KafkaIBP: ibpVersion,
				Strimzi:  strimziVersion,
			},
		},
	}
	if modifyFn != nil {
		modifyFn(managedKafka)
	}
	return managedKafka
}

func BuildPrivateKafka(modifyFn func(privatekafka *private.ManagedKafka)) *private.ManagedKafka {
	privatekafka := &private.ManagedKafka{
		Spec: private.ManagedKafkaAllOfSpec{
			Versions: private.ManagedKafkaVersions{
				Kafka:    kafkaVersion,
				KafkaIbp: ibpVersion,
				Strimzi:  strimziVersion,
			},
		},
	}
	if modifyFn != nil {
		modifyFn(privatekafka)
	}
	return privatekafka
}

func BuildTlsSpec(modifyFn func(tlsSpec *v1.TlsSpec)) *v1.TlsSpec {
	tlsSpec := &v1.TlsSpec{
		Cert: dummyCert,
		Key:  dummyKey,
	}
	if modifyFn != nil {
		modifyFn(tlsSpec)
	}
	return tlsSpec
}

func BuildManagedKafkaAllOfSpecEndpointTls(modifyFn func(tls *private.ManagedKafkaAllOfSpecEndpointTls)) *private.ManagedKafkaAllOfSpecEndpointTls {
	tlsSpec := &private.ManagedKafkaAllOfSpecEndpointTls{
		Cert: dummyCert,
		Key:  dummyKey,
	}
	if modifyFn != nil {
		modifyFn(tlsSpec)
	}
	return tlsSpec
}

func BuildOAuthSpec(modifyFn func(oauthSpec *v1.OAuthSpec)) *v1.OAuthSpec {
	oauthSpec := &v1.OAuthSpec{
		TlsTrustedCertificate: &trustedCertValue,
	}
	if modifyFn != nil {
		modifyFn(oauthSpec)
	}
	return oauthSpec
}

func GetTrustedCertValue() *string {
	return &trustedCertValue
}
