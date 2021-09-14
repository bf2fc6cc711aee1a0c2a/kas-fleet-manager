/*
 * Pineapple Service Fleet Manager
 *
 * Pineapple Service Fleet Manager APIs that are used by internal services e.g kas-fleetshard operators.
 *
 * API version: 1.3.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// ManagedPineappleAllOfSpecOauth OAuth configuration for a Pineapple cluster
type ManagedPineappleAllOfSpecOauth struct {
	ClientId               string  `json:"clientId,omitempty"`
	ClientSecret           string  `json:"clientSecret,omitempty"`
	TokenEndpointURI       string  `json:"tokenEndpointURI,omitempty"`
	JwksEndpointURI        string  `json:"jwksEndpointURI,omitempty"`
	ValidIssuerEndpointURI string  `json:"validIssuerEndpointURI,omitempty"`
	UserNameClaim          string  `json:"userNameClaim,omitempty"`
	TlsTrustedCertificate  *string `json:"tlsTrustedCertificate,omitempty"`
	CustomClaimCheck       string  `json:"customClaimCheck,omitempty"`
}
