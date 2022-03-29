package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
	v1 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/managedkafkas.managedkafka.bf2.org/v1"
)

func PresentManagedKafka(from *v1.ManagedKafka) private.ManagedKafka {
	res := private.ManagedKafka{
		Id:   from.Annotations["bf2.org/id"],
		Kind: from.Kind,
		Metadata: private.ManagedKafkaAllOfMetadata{
			Name:      from.Name,
			Namespace: from.Namespace,
			Annotations: private.ManagedKafkaAllOfMetadataAnnotations{
				Bf2OrgId:          from.Annotations["bf2.org/id"],
				Bf2OrgPlacementId: from.Annotations["bf2.org/placementId"],
			},
			Labels: private.ManagedKafkaAllOfMetadataLabels{
				Bf2OrgKafkaInstanceProfileType:          from.Labels["bf2.org/kafkaInstanceProfileType"],
				Bf2OrgKafkaInstanceProfileQuotaConsumed: from.Labels["bf2.org/kafkaInstanceProfileQuotaConsumed"],
			},
		},
		Spec: private.ManagedKafkaAllOfSpec{
			Capacity: private.ManagedKafkaCapacity{
				IngressThroughputPerSec:     from.Spec.Capacity.IngressThroughputPerSec,
				EgressThroughputPerSec:      from.Spec.Capacity.EgressThroughputPerSec,
				TotalMaxConnections:         int32(from.Spec.Capacity.TotalMaxConnections),
				MaxDataRetentionSize:        from.Spec.Capacity.MaxDataRetentionSize,
				MaxPartitions:               int32(from.Spec.Capacity.MaxPartitions),
				MaxDataRetentionPeriod:      from.Spec.Capacity.MaxDataRetentionPeriod,
				MaxConnectionAttemptsPerSec: int32(from.Spec.Capacity.MaxConnectionAttemptsPerSec),
			},
			Oauth: private.ManagedKafkaAllOfSpecOauth{
				ClientId:               from.Spec.OAuth.ClientId,
				ClientSecret:           from.Spec.OAuth.ClientSecret,
				TokenEndpointURI:       from.Spec.OAuth.TokenEndpointURI,
				JwksEndpointURI:        from.Spec.OAuth.JwksEndpointURI,
				ValidIssuerEndpointURI: from.Spec.OAuth.ValidIssuerEndpointURI,
				UserNameClaim:          from.Spec.OAuth.UserNameClaim,
				FallbackUserNameClaim:  from.Spec.OAuth.FallBackUserNameClaim,
				TlsTrustedCertificate:  getOpenAPIManagedKafkaOAuthTLSTrustedCertificate(&from.Spec.OAuth),
				CustomClaimCheck:       from.Spec.OAuth.CustomClaimCheck,
				MaximumSessionLifetime: from.Spec.OAuth.MaximumSessionLifetime,
			},
			Endpoint: private.ManagedKafkaAllOfSpecEndpoint{
				Tls:                 getOpenAPIManagedKafkaEndpointTLS(from.Spec.Endpoint.Tls),
				BootstrapServerHost: from.Spec.Endpoint.BootstrapServerHost,
			},
			Versions: private.ManagedKafkaVersions{
				Kafka:    from.Spec.Versions.Kafka,
				KafkaIbp: from.Spec.Versions.KafkaIBP,
				Strimzi:  from.Spec.Versions.Strimzi,
			},
			Deleted:         from.Spec.Deleted,
			Owners:          from.Spec.Owners,
			ServiceAccounts: getServiceAccounts(from.Spec.ServiceAccounts),
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

func getServiceAccounts(from []v1.ServiceAccount) []private.ManagedKafkaAllOfSpecServiceAccounts {
	accounts := []private.ManagedKafkaAllOfSpecServiceAccounts{}
	for _, managedServiceAccount := range from {
		accounts = append(accounts, private.ManagedKafkaAllOfSpecServiceAccounts{
			Name:      managedServiceAccount.Name,
			Principal: managedServiceAccount.Principal,
			Password:  managedServiceAccount.Password,
		})
	}
	return accounts
}
