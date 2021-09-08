package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/dinosaur/internal/api/private"
	v1 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/manageddinosaurs.manageddinosaur.bf2.org/v1"
)

func PresentManagedDinosaur(from *v1.ManagedDinosaur) private.ManagedDinosaur {
	res := private.ManagedDinosaur{
		Id:   from.Annotations["id"],
		Kind: from.Kind,
		Metadata: private.ManagedDinosaurAllOfMetadata{
			Name:      from.Name,
			Namespace: from.Namespace,
			Annotations: private.ManagedDinosaurAllOfMetadataAnnotations{
				Bf2OrgId:          from.Annotations["bf2.org/id"],
				Bf2OrgPlacementId: from.Annotations["bf2.org/placementId"],
			},
		},
		Spec: private.ManagedDinosaurAllOfSpec{
			Capacity: private.ManagedDinosaurCapacity{
				IngressEgressThroughputPerSec: from.Spec.Capacity.IngressEgressThroughputPerSec,
				TotalMaxConnections:           int32(from.Spec.Capacity.TotalMaxConnections),
				MaxDataRetentionSize:          from.Spec.Capacity.MaxDataRetentionSize,
				MaxPartitions:                 int32(from.Spec.Capacity.MaxPartitions),
				MaxDataRetentionPeriod:        from.Spec.Capacity.MaxDataRetentionPeriod,
				MaxConnectionAttemptsPerSec:   int32(from.Spec.Capacity.MaxConnectionAttemptsPerSec),
			},
			Oauth: private.ManagedDinosaurAllOfSpecOauth{
				ClientId:               from.Spec.OAuth.ClientId,
				ClientSecret:           from.Spec.OAuth.ClientSecret,
				TokenEndpointURI:       from.Spec.OAuth.TokenEndpointURI,
				JwksEndpointURI:        from.Spec.OAuth.JwksEndpointURI,
				ValidIssuerEndpointURI: from.Spec.OAuth.ValidIssuerEndpointURI,
				UserNameClaim:          from.Spec.OAuth.UserNameClaim,
				TlsTrustedCertificate:  getOpenAPIManagedDinosaurOAuthTLSTrustedCertificate(&from.Spec.OAuth),
				CustomClaimCheck:       from.Spec.OAuth.CustomClaimCheck,
			},
			Endpoint: private.ManagedDinosaurAllOfSpecEndpoint{
				Tls:                 getOpenAPIManagedDinosaurEndpointTLS(from.Spec.Endpoint.Tls),
				BootstrapServerHost: from.Spec.Endpoint.BootstrapServerHost,
			},
			Versions: private.ManagedDinosaurVersions{
				Dinosaur: from.Spec.Versions.Dinosaur,
				Strimzi:  from.Spec.Versions.Strimzi,
			},
			Deleted:         from.Spec.Deleted,
			Owners:          from.Spec.Owners,
			ServiceAccounts: getServiceAccounts(from.Spec.ServiceAccounts),
		},
	}

	return res
}

func getOpenAPIManagedDinosaurEndpointTLS(from *v1.TlsSpec) *private.ManagedDinosaurAllOfSpecEndpointTls {
	var res *private.ManagedDinosaurAllOfSpecEndpointTls
	if from != nil {
		res = &private.ManagedDinosaurAllOfSpecEndpointTls{
			Cert: from.Cert,
			Key:  from.Key,
		}
	}
	return res
}

func getOpenAPIManagedDinosaurOAuthTLSTrustedCertificate(from *v1.OAuthSpec) *string {
	var res *string
	if from.TlsTrustedCertificate != nil {
		res = from.TlsTrustedCertificate
	}
	return res
}

func getServiceAccounts(from []v1.ServiceAccount) []private.ManagedDinosaurAllOfSpecServiceAccounts {
	accounts := []private.ManagedDinosaurAllOfSpecServiceAccounts{}
	for _, managedServiceAccount := range from {
		accounts = append(accounts, private.ManagedDinosaurAllOfSpecServiceAccounts{
			Name:      managedServiceAccount.Name,
			Principal: managedServiceAccount.Principal,
			Password:  managedServiceAccount.Password,
		})
	}
	return accounts
}
