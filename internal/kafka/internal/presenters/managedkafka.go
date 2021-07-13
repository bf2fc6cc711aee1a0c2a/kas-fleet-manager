package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
	v1 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/managedkafkas.managedkafka.bf2.org/v1"
)

func PresentManagedKafka(from *v1.ManagedKafka) private.ManagedKafka {
	res := private.ManagedKafka{
		Id:   from.Annotations["id"],
		Kind: from.Kind,
		Metadata: private.ManagedKafkaAllOfMetadata{
			Name:      from.Name,
			Namespace: from.Namespace,
			Annotations: private.ManagedKafkaAllOfMetadataAnnotations{
				DeprecatedBf2OrgId:          from.Annotations["bf2.org/id"],
				DeprecatedBf2OrgPlacementId: from.Annotations["bf2.org/placementId"],
				Id:                          from.Annotations["bf2.org/id"],
				PlacementId:                 from.Annotations["bf2.org/placementId"],
			},
		},
		Spec: private.ManagedKafkaAllOfSpec{
			Capacity: private.ManagedKafkaCapacity{
				IngressEgressThroughputPerSec:           from.Spec.Capacity.IngressEgressThroughputPerSec,
				TotalMaxConnections:                     int32(from.Spec.Capacity.TotalMaxConnections),
				MaxDataRetentionSize:                    from.Spec.Capacity.MaxDataRetentionSize,
				MaxPartitions:                           int32(from.Spec.Capacity.MaxPartitions),
				MaxDataRetentionPeriod:                  from.Spec.Capacity.MaxDataRetentionPeriod,
				MaxConnectionAttemptsPerSec:             int32(from.Spec.Capacity.MaxConnectionAttemptsPerSec),
				DeprecatedIngressEgressThroughputPerSec: from.Spec.Capacity.IngressEgressThroughputPerSec,
				DeprecatedTotalMaxConnections:           int32(from.Spec.Capacity.TotalMaxConnections),
				DeprecatedMaxDataRetentionSize:          from.Spec.Capacity.MaxDataRetentionSize,
				DeprecatedMaxPartitions:                 int32(from.Spec.Capacity.MaxPartitions),
				DeprecatedMaxDataRetentionPeriod:        from.Spec.Capacity.MaxDataRetentionPeriod,
				DeprecatedMaxConnectionAttemptsPerSec:   int32(from.Spec.Capacity.MaxConnectionAttemptsPerSec),
			},
			Oauth: private.ManagedKafkaAllOfSpecOauth{
				ClientId:                         from.Spec.OAuth.ClientId,
				ClientSecret:                     from.Spec.OAuth.ClientSecret,
				TokenEndpointUri:                 from.Spec.OAuth.TokenEndpointURI,
				JwksEndpointUri:                  from.Spec.OAuth.JwksEndpointURI,
				ValidIssuerEndpointUri:           from.Spec.OAuth.ValidIssuerEndpointURI,
				UserNameClaim:                    from.Spec.OAuth.UserNameClaim,
				TlsTrustedCertificate:            getOpenAPIManagedKafkaOAuthTLSTrustedCertificate(&from.Spec.OAuth),
				CustomClaimCheck:                 from.Spec.OAuth.CustomClaimCheck,
				DeprecatedClientId:               from.Spec.OAuth.ClientId,
				DeprecatedClientSecret:           from.Spec.OAuth.ClientSecret,
				DeprecatedTokenEndpointURI:       from.Spec.OAuth.TokenEndpointURI,
				DeprecatedJwksEndpointURI:        from.Spec.OAuth.JwksEndpointURI,
				DeprecatedValidIssuerEndpointURI: from.Spec.OAuth.ValidIssuerEndpointURI,
				DeprecatedUserNameClaim:          from.Spec.OAuth.UserNameClaim,
				DeprecatedTlsTrustedCertificate:  getOpenAPIManagedKafkaOAuthTLSTrustedCertificate(&from.Spec.OAuth),
				DeprecatedCustomClaimCheck:       from.Spec.OAuth.CustomClaimCheck,
			},
			Endpoint: private.ManagedKafkaAllOfSpecEndpoint{
				Tls:                           getOpenAPIManagedKafkaEndpointTLS(from.Spec.Endpoint.Tls),
				BootstrapServerHost:           from.Spec.Endpoint.BootstrapServerHost,
				DeprecatedBootstrapServerHost: from.Spec.Endpoint.BootstrapServerHost,
			},
			Versions: private.ManagedKafkaVersions{
				Kafka:   from.Spec.Versions.Kafka,
				Strimzi: from.Spec.Versions.Strimzi,
			},
			Deleted: from.Spec.Deleted,
			Owners:  from.Spec.Owners,
		},
	}

	return res
}

func getOpenAPIManagedKafkaEndpointTLS(from *v1.TlsSpec) *private.ManagedKafkaAllOfSpecEndpointTls {
	var res *private.ManagedKafkaAllOfSpecEndpointTls
	if from != nil {
		res = &private.ManagedKafkaAllOfSpecEndpointTls{
			Cert: from.Cert,
			Key:  from.Key,
		}
	}
	return res
}

func getOpenAPIManagedKafkaOAuthTLSTrustedCertificate(from *v1.OAuthSpec) *string {
	var res *string
	if from.TlsTrustedCertificate != nil {
		res = from.TlsTrustedCertificate
	}
	return res
}
