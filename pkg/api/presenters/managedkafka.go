package presenters

import (
	v1 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/managedkafkas.managedkafka.bf2.org/v1"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
)

func PresentManagedKafka(from *v1.ManagedKafka) openapi.ManagedKafka {
	res := openapi.ManagedKafka{
		Name: from.Name,
		Id:   from.Id,
		Kind: from.Kind,
		Href: from.Name, // We don't have an Href for ManagedKafka
		Annotation: openapi.ManagedKafkaAllOfAnnotation{
			Id:          from.Id,
			PlacementId: from.PlacementId,
		},
		Spec: openapi.ManagedKafkaAllOfSpec{
			Capacity: openapi.ManagedKafkaCapacity{
				IngressEgressThroughputPerSec: from.Spec.Capacity.IngressEgressThroughputPerSec,
				TotalMaxConnections:           int32(from.Spec.Capacity.TotalMaxConnections),
				MaxDataRetentionSize:          from.Spec.Capacity.MaxDataRetentionSize,
				MaxPartitions:                 int32(from.Spec.Capacity.MaxPartitions),
				MaxDataRetentionPeriod:        from.Spec.Capacity.MaxDataRetentionPeriod,
			},
			OAuth: openapi.ManagedKafkaAllOfSpecOAuth{
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

	if cnds := from.Status.Conditions; cnds != nil {
		for _, cnd := range cnds {
			res.Status.Conditions = append(res.Status.Conditions, openapi.MetaV1Condition{
				Type:               cnd.Type,
				Reason:             cnd.Reason,
				Message:            cnd.Message,
				Status:             string(cnd.Status),
				LastTransitionTime: cnd.LastTransitionTime.String(),
			})
		}
	}

	return res
}
