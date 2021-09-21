package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/private"
	v1 "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api/manageddinosaurs.manageddinosaur.bf2.org/v1"
)

func PresentManagedDinosaur(from *v1.ManagedDinosaur) private.ManagedPineapple {
	res := private.ManagedPineapple{
		Id:   from.Annotations["id"],
		Kind: from.Kind,
		Metadata: private.ManagedPineappleAllOfMetadata{
			Name:      from.Name,
			Namespace: from.Namespace,
			Annotations: private.ManagedPineappleAllOfMetadataAnnotations{
				MasId:          from.Annotations["mas/id"],
				MasPlacementId: from.Annotations["mas/placementId"],
			},
		},
		Spec: private.ManagedPineappleAllOfSpec{
			Capacity: map[string]interface{}{},
			Oauth: private.ManagedPineappleAllOfSpecOauth{
				ClientId:               from.Spec.OAuth.ClientId,
				ClientSecret:           from.Spec.OAuth.ClientSecret,
				TokenEndpointURI:       from.Spec.OAuth.TokenEndpointURI,
				JwksEndpointURI:        from.Spec.OAuth.JwksEndpointURI,
				ValidIssuerEndpointURI: from.Spec.OAuth.ValidIssuerEndpointURI,
				UserNameClaim:          from.Spec.OAuth.UserNameClaim,
				TlsTrustedCertificate:  getOpenAPIManagedDinosaurOAuthTLSTrustedCertificate(&from.Spec.OAuth),
				CustomClaimCheck:       from.Spec.OAuth.CustomClaimCheck,
			},
			Endpoint: private.ManagedPineappleAllOfSpecEndpoint{
				Tls:  getOpenAPIManagedDinosaurEndpointTLS(from.Spec.Endpoint.Tls),
				Host: from.Spec.Endpoint.BootstrapServerHost,
			},
			Versions: private.ManagedPineappleVersions{
				Pineapple:         from.Spec.Versions.Dinosaur,
				PineappleOperator: from.Spec.Versions.Strimzi,
			},
			Deleted:         from.Spec.Deleted,
			Owners:          from.Spec.Owners,
			ServiceAccounts: getServiceAccounts(from.Spec.ServiceAccounts),
		},
	}

	return res
}

func getOpenAPIManagedDinosaurEndpointTLS(from *v1.TlsSpec) *private.ManagedPineappleAllOfSpecEndpointTls {
	var res *private.ManagedPineappleAllOfSpecEndpointTls
	if from != nil {
		res = &private.ManagedPineappleAllOfSpecEndpointTls{
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

func getServiceAccounts(from []v1.ServiceAccount) []private.ManagedPineappleAllOfSpecServiceAccounts {
	accounts := []private.ManagedPineappleAllOfSpecServiceAccounts{}
	for _, managedServiceAccount := range from {
		accounts = append(accounts, private.ManagedPineappleAllOfSpecServiceAccounts{
			Name:      managedServiceAccount.Name,
			Principal: managedServiceAccount.Principal,
			Password:  managedServiceAccount.Password,
		})
	}
	return accounts
}
