package presenters

import (
	v1 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/managedkafkas.managedkafka.bf2.org/v1"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
)

func PresentManagedKafka(from *v1.ManagedKafka) openapi.ManagedKafka {
	res := openapi.ManagedKafka{
		Id:   from.Annotations["id"],
		Kind: from.Kind,
		Metadata: openapi.ManagedKafkaAllOfMetadata{
			Name:      from.Name,
			Namespace: from.Namespace,
			Annotations: openapi.ManagedKafkaAllOfMetadataAnnotations{
				DeprecatedBf2OrgId:          from.Annotations["bf2.org/id"],
				DeprecatedBf2OrgPlacementId: from.Annotations["bf2.org/placementId"],
				Id:                          from.Annotations["bf2.org/id"],
				PlacementId:                 from.Annotations["bf2.org/placementId"],
			},
		},
		Spec: openapi.ManagedKafkaAllOfSpec{
			Capacity: openapi.ManagedKafkaCapacity{
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
			Oauth: openapi.ManagedKafkaAllOfSpecOauth{
				ClientId:                         from.Spec.OAuth.ClientId,
				ClientSecret:                     from.Spec.OAuth.ClientSecret,
				TokenEndpointUri:                 from.Spec.OAuth.TokenEndpointURI,
				JwksEndpointUri:                  from.Spec.OAuth.JwksEndpointURI,
				ValidIssuerEndpointUri:           from.Spec.OAuth.ValidIssuerEndpointURI,
				UserNameClaim:                    from.Spec.OAuth.UserNameClaim,
				TlsTrustedCertificate:            from.Spec.OAuth.TlsTrustedCertificate,
				CustomClaimCheck:                 from.Spec.OAuth.CustomClaimCheck,
				DeprecatedClientId:               from.Spec.OAuth.ClientId,
				DeprecatedClientSecret:           from.Spec.OAuth.ClientSecret,
				DeprecatedTokenEndpointURI:       from.Spec.OAuth.TokenEndpointURI,
				DeprecatedJwksEndpointURI:        from.Spec.OAuth.JwksEndpointURI,
				DeprecatedValidIssuerEndpointURI: from.Spec.OAuth.ValidIssuerEndpointURI,
				DeprecatedUserNameClaim:          from.Spec.OAuth.UserNameClaim,
				DeprecatedTlsTrustedCertificate:  from.Spec.OAuth.TlsTrustedCertificate,
				DeprecatedCustomClaimCheck:       from.Spec.OAuth.CustomClaimCheck,
			},
			Endpoint: openapi.ManagedKafkaAllOfSpecEndpoint{
				Tls:                           getOpenAPIManagedKafkaEndpointTLS(from.Spec.Endpoint.Tls),
				BootstrapServerHost:           from.Spec.Endpoint.BootstrapServerHost,
				DeprecatedBootstrapServerHost: from.Spec.Endpoint.BootstrapServerHost,
			},
			Versions: openapi.ManagedKafkaVersions{
				Kafka:   from.Spec.Versions.Kafka,
				Strimzi: from.Spec.Versions.Strimzi,
			},
			Deleted: from.Spec.Deleted,
		},
	}

	return res
}

func getOpenAPIManagedKafkaEndpointTLS(from *v1.TlsSpec) *openapi.ManagedKafkaAllOfSpecEndpointTls {
	var res *openapi.ManagedKafkaAllOfSpecEndpointTls
	if from != nil {
		res = &openapi.ManagedKafkaAllOfSpecEndpointTls{
			Cert: from.Cert,
			Key:  from.Key,
		}
	}
	return res
}
