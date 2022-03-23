package api

type SsoProvider struct {
	BaseUrl     string `json:"base_url,omitempty"`
	TokenUrl    string `json:"token_url,omitempty"`
	Jwks        string `json:"jwks,omitempty"`
	ValidIssuer string `json:"valid_issuer,omitempty"`
}
