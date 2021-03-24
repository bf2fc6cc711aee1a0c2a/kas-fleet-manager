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
			Name: from.Name,
			Annotation: openapi.ManagedKafkaAllOfMetadataAnnotation{
				Bf2OrgId:          from.Annotations["bf2.org/id"],
				Bf2OrgPlacementId: from.Annotations["bf2.org/placementId"],
			},
		},
		Spec: openapi.ManagedKafkaAllOfSpec{
			Capacity: openapi.ManagedKafkaCapacity{
				IngressEgressThroughputPerSec: from.Spec.Capacity.IngressEgressThroughputPerSec,
				TotalMaxConnections:           int32(from.Spec.Capacity.TotalMaxConnections),
				MaxDataRetentionSize:          from.Spec.Capacity.MaxDataRetentionSize,
				MaxPartitions:                 int32(from.Spec.Capacity.MaxPartitions),
				MaxDataRetentionPeriod:        from.Spec.Capacity.MaxDataRetentionPeriod,
			},
			Oauth: openapi.ManagedKafkaAllOfSpecOauth{
				ClientId:               from.Spec.OAuth.ClientId,
				ClientSecret:           from.Spec.OAuth.ClientSecret,
				TokenEndpointURI:       from.Spec.OAuth.TokenEndpointURI,
				JwksEndpointURI:        from.Spec.OAuth.JwksEndpointURI,
				ValidIssuerEndpointURI: from.Spec.OAuth.ValidIssuerEndpointURI,
				UserNameClaim:          from.Spec.OAuth.UserNameClaim,
				TlsTrustedCertificate:  from.Spec.OAuth.TlsTrustedCertificate,
			},
			Endpoint: openapi.ManagedKafkaAllOfSpecEndpoint{
				BootstrapServerHost: from.Spec.Endpoint.BootstrapServerHost,
				Tls: openapi.ManagedKafkaAllOfSpecEndpointTls{
					Cert: from.Spec.Endpoint.Tls.Cert,
					Key:  from.Spec.Endpoint.Tls.Key,
				},
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
