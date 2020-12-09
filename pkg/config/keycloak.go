package config

import "github.com/spf13/pflag"

type KeycloakConfig struct {
	EnableAuthenticationOnKafka bool   `json:"enable_auth"`
	Realm                       string `json:"realm"`
	BaseURL                     string `json:"base_url"`
	ClientID                    string `json:"client-id"`
	ClientIDFile                string `json:"client-id_file"`
	ClientSecret                string `json:"client-secret"`
	ClientSecretFile            string `json:"client-secret_file"`
	Debug                       bool   `json:"debug"`
	InsecureSkipVerify          bool   `json:"insecure-skip-verify"`
	GrantType                   string `json:"grant_type"`
	TokenEndpointURI            string `json:"token_endpoint_uri"`
	JwksEndpointURI             string `json:"jwks_endpoint_uri"`
	UserNameClaim               string `json:"user_name_claim"`
	ValidIssuerURI              string `json:"valid_issuer_uri"`
	TLSTrustedCertificatesKey   string `json:"tls_trusted_certificates_key"`
	TLSTrustedCertificatesValue string `json:"tls_trusted_certificates_value"`
	TLSTrustedCertificatesFile  string `json:"tls_trusted_certificates_file"`
	MASClientSecretKey          string `json:"mas_client_secret_key"`
	MASClientSecretValue        string `json:"mas_client_secret_value"`
	EnablePlain                 bool   `json:"enable_plain"`
	EnableOauthBearer           bool   `json:"enable_oauth_bearer"`
}

func NewKeycloakConfig() *KeycloakConfig {
	kc := &KeycloakConfig{
		EnableAuthenticationOnKafka: true,
		ClientIDFile:                "secrets/keycloak-service.clientId",
		ClientSecretFile:            "secrets/keycloak-service.clientSecret",
		TLSTrustedCertificatesFile:  "secrets/keycloak-service.crt",
		Debug:                       false,
		InsecureSkipVerify:          false,
		GrantType:                   "client_credentials",
		UserNameClaim:               "preferred_username",
		TLSTrustedCertificatesKey:   "keycloak.crt",
		MASClientSecretKey:          "ssoClientSecret",
		EnablePlain:                 true,
		EnableOauthBearer:           false,
	}
	return kc
}

func (kc *KeycloakConfig) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&kc.EnableAuthenticationOnKafka, "mas-sso-enable-auth", kc.EnableAuthenticationOnKafka, "Enable authentication mas-sso integration, enabled by default")
	fs.StringVar(&kc.ClientIDFile, "mas-sso-client-id-file", kc.ClientIDFile, "File containing Keycloak privileged account client-id")
	fs.StringVar(&kc.ClientSecretFile, "mas-sso-client-secret-file", kc.ClientSecretFile, "File containing Keycloak privileged account client-secret")
	fs.StringVar(&kc.BaseURL, "mas-sso-base-url", kc.BaseURL, "The base URL of the mas-sso, integration by default")
	fs.StringVar(&kc.Realm, "mas-sso-realm", kc.Realm, "Realm for the mas-sso")
	fs.StringVar(&kc.TLSTrustedCertificatesFile, "mas-sso-cert-file", kc.TLSTrustedCertificatesFile, "File containing tls cert for the mas-sso")
	fs.BoolVar(&kc.Debug, "mas-sso-debug", kc.Debug, "Debug flag for Keycloak API")
	fs.BoolVar(&kc.InsecureSkipVerify, "mas-sso-insecure", kc.InsecureSkipVerify, "Disable tls verification with mas-sso")
}

func (kc *KeycloakConfig) ReadFiles() error {
	err := readFileValueString(kc.ClientIDFile, &kc.ClientID)
	if err != nil {
		return err
	}
	err = readFileValueString(kc.ClientSecretFile, &kc.ClientSecret)
	if err != nil {
		return err
	}
	err = readFileValueString(kc.TLSTrustedCertificatesFile, &kc.TLSTrustedCertificatesValue)
	if err != nil {
		return err
	}
	return nil
}
