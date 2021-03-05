package config

import "github.com/spf13/pflag"

type KeycloakConfig struct {
	EnableAuthenticationOnKafka bool                 `json:"enable_auth"`
	BaseURL                     string               `json:"base_url"`
	Debug                       bool                 `json:"debug"`
	InsecureSkipVerify          bool                 `json:"insecure-skip-verify"`
	UserNameClaim               string               `json:"user_name_claim"`
	TLSTrustedCertificatesKey   string               `json:"tls_trusted_certificates_key"`
	TLSTrustedCertificatesValue string               `json:"tls_trusted_certificates_value"`
	TLSTrustedCertificatesFile  string               `json:"tls_trusted_certificates_file"`
	EnablePlain                 bool                 `json:"enable_plain"`
	EnableOauthBearer           bool                 `json:"enable_oauth_bearer"`
	KafkaRealm                  *KeycloakRealmConfig `json:"kafka_realm"`
}

type KeycloakRealmConfig struct {
	Realm            string `json:"realm"`
	ClientID         string `json:"client-id"`
	ClientIDFile     string `json:"client-id_file"`
	ClientSecret     string `json:"client-secret"`
	ClientSecretFile string `json:"client-secret_file"`
	GrantType        string `json:"grant_type"`
	TokenEndpointURI string `json:"token_endpoint_uri"`
	JwksEndpointURI  string `json:"jwks_endpoint_uri"`
	ValidIssuerURI   string `json:"valid_issuer_uri"`
}

func NewKeycloakConfig() *KeycloakConfig {
	kc := &KeycloakConfig{
		EnableAuthenticationOnKafka: false,
		KafkaRealm: &KeycloakRealmConfig{
			ClientIDFile:     "secrets/keycloak-service.clientId",
			ClientSecretFile: "secrets/keycloak-service.clientSecret",
			GrantType:        "client_credentials",
		},
		TLSTrustedCertificatesFile: "secrets/keycloak-service.crt",
		Debug:                      false,
		InsecureSkipVerify:         false,
		UserNameClaim:              "preferred_username",
		TLSTrustedCertificatesKey:  "keycloak.crt",
		EnablePlain:                true,
		EnableOauthBearer:          false,
	}
	return kc
}

func (kc *KeycloakConfig) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&kc.EnableAuthenticationOnKafka, "mas-sso-enable-auth", kc.EnableAuthenticationOnKafka, "Enable authentication mas-sso integration, enabled by default")
	fs.StringVar(&kc.KafkaRealm.ClientIDFile, "mas-sso-client-id-file", kc.KafkaRealm.ClientIDFile, "File containing Keycloak privileged account client-id that has access to the Kafka service accounts realm")
	fs.StringVar(&kc.KafkaRealm.ClientSecretFile, "mas-sso-client-secret-file", kc.KafkaRealm.ClientSecretFile, "File containing Keycloak privileged account client-secret that has access to the Kafka service accounts realm")
	fs.StringVar(&kc.BaseURL, "mas-sso-base-url", kc.BaseURL, "The base URL of the mas-sso, integration by default")
	fs.StringVar(&kc.KafkaRealm.Realm, "mas-sso-realm", kc.KafkaRealm.Realm, "Realm for Kafka service accounts in the mas-sso")
	fs.StringVar(&kc.TLSTrustedCertificatesFile, "mas-sso-cert-file", kc.TLSTrustedCertificatesFile, "File containing tls cert for the mas-sso")
	fs.BoolVar(&kc.Debug, "mas-sso-debug", kc.Debug, "Debug flag for Keycloak API")
	fs.BoolVar(&kc.InsecureSkipVerify, "mas-sso-insecure", kc.InsecureSkipVerify, "Disable tls verification with mas-sso")
}

func (kc *KeycloakConfig) ReadFiles() error {
	err := readFileValueString(kc.KafkaRealm.ClientIDFile, &kc.KafkaRealm.ClientID)
	if err != nil {
		return err
	}
	err = readFileValueString(kc.KafkaRealm.ClientSecretFile, &kc.KafkaRealm.ClientSecret)
	if err != nil {
		return err
	}
	err = readFileValueString(kc.TLSTrustedCertificatesFile, &kc.TLSTrustedCertificatesValue)
	if err != nil {
		return err
	}
	return nil
}
