package config

import "github.com/spf13/pflag"

type KeycloakConfig struct {
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
}

func NewKeycloakConfig() *KeycloakConfig {
	return &KeycloakConfig{
		Realm:                      "masdemo",
		BaseURL:                    "https://keycloak-sso-mas.apps.sso-mas.a4v1.s1.devshift.org",
		ClientIDFile:               "secrets/keycloak-service.clientId",
		ClientSecretFile:           "secrets/keycloak-service.clientSecret",
		TLSTrustedCertificatesFile: "secrets/keycloak-service.crt",
		Debug:                      true,
		InsecureSkipVerify:         true,
		GrantType:                  "client_credentials",
		UserNameClaim:              "preferred_username",
		JwksEndpointURI:            "https://keycloak-sso-mas.apps.sso-mas.a4v1.s1.devshift.org/auth/realms/masdemo/protocol/openid-connect/certs",
		TokenEndpointURI:           "https://keycloak-sso-mas.apps.sso-mas.a4v1.s1.devshift.org/auth/realms/masdemo/protocol/openid-connect/certs",
		ValidIssuerURI:             "https://keycloak-sso-mas.apps.sso-mas.a4v1.s1.devshift.org/auth/realms/masdemo",
		TLSTrustedCertificatesKey:  "keycloak.crt",
		MASClientSecretKey:         "ssoClientSecret",
	}
}

func (kc *KeycloakConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&kc.ClientIDFile, "keycloak-client-id-file", kc.ClientIDFile, "File containing Keycloak privileged account client-id")
	fs.StringVar(&kc.ClientSecretFile, "keycloak-client-secret-file", kc.ClientSecretFile, "File containing Keycloak privileged account client-secret")
	fs.StringVar(&kc.BaseURL, "keycloak-base-url", kc.BaseURL, "The base URL of the MAS Keycloak, integration by default")
	fs.BoolVar(&kc.Debug, "keycloak-debug", kc.Debug, "Debug flag for Keycloak API")
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
