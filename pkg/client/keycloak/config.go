package keycloak

import (
	"os"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/shared"

	"github.com/golang/glog"
	"github.com/spf13/pflag"
)

type KeycloakConfig struct {
	EnableAuthenticationOnDinosaur bool                 `json:"enable_auth"`
	BaseURL                        string               `json:"base_url"`
	Debug                          bool                 `json:"debug"`
	InsecureSkipVerify             bool                 `json:"insecure-skip-verify"`
	UserNameClaim                  string               `json:"user_name_claim"`
	FallBackUserNameClaim          string               `json:"fall_back_user_name_claim"`
	TLSTrustedCertificatesKey      string               `json:"tls_trusted_certificates_key"`
	TLSTrustedCertificatesValue    string               `json:"tls_trusted_certificates_value"`
	TLSTrustedCertificatesFile     string               `json:"tls_trusted_certificates_file"`
	EnablePlain                    bool                 `json:"enable_plain"`
	EnableOauthBearer              bool                 `json:"enable_oauth_bearer"`
	EnableCustomClaimCheck         bool                 `json:"enable_custom_claim_check"`
	DinosaurRealm                  *KeycloakRealmConfig `json:"dinosaur_realm"`
	OSDClusterIDPRealm             *KeycloakRealmConfig `json:"osd_cluster_idp_realm"`
	MaxLimitForGetClients          int                  `json:"max_limit_for_get_clients"`
	KeycloakClientExpire           bool                 `json:"keycloak_client_expire"`
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

func (c *KeycloakRealmConfig) setDefaultURIs(baseURL string) {
	c.ValidIssuerURI = baseURL + "/auth/realms/" + c.Realm
	c.JwksEndpointURI = baseURL + "/auth/realms/" + c.Realm + "/protocol/openid-connect/certs"
	c.TokenEndpointURI = baseURL + "/auth/realms/" + c.Realm + "/protocol/openid-connect/token"
}

func NewKeycloakConfig() *KeycloakConfig {
	kc := &KeycloakConfig{
		EnableAuthenticationOnDinosaur: true,
		DinosaurRealm: &KeycloakRealmConfig{
			ClientIDFile:     "secrets/keycloak-service.clientId",
			ClientSecretFile: "secrets/keycloak-service.clientSecret",
			GrantType:        "client_credentials",
		},
		OSDClusterIDPRealm: &KeycloakRealmConfig{
			ClientIDFile:     "secrets/osd-idp-keycloak-service.clientId",
			ClientSecretFile: "secrets/osd-idp-keycloak-service.clientSecret",
			GrantType:        "client_credentials",
		},
		TLSTrustedCertificatesFile: "secrets/keycloak-service.crt",
		Debug:                      false,
		InsecureSkipVerify:         false,
		UserNameClaim:              "clientId",
		FallBackUserNameClaim:      "preferred_username",
		TLSTrustedCertificatesKey:  "keycloak.crt",
		EnablePlain:                true,
		EnableOauthBearer:          false,
		MaxLimitForGetClients:      100,
		KeycloakClientExpire:       false,
	}
	return kc
}

func (kc *KeycloakConfig) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&kc.EnableAuthenticationOnDinosaur, "mas-sso-enable-auth", kc.EnableAuthenticationOnDinosaur, "Enable authentication mas-sso integration, enabled by default")
	fs.StringVar(&kc.DinosaurRealm.ClientIDFile, "mas-sso-client-id-file", kc.DinosaurRealm.ClientIDFile, "File containing Keycloak privileged account client-id that has access to the Dinosaur service accounts realm")
	fs.StringVar(&kc.DinosaurRealm.ClientSecretFile, "mas-sso-client-secret-file", kc.DinosaurRealm.ClientSecretFile, "File containing Keycloak privileged account client-secret that has access to the Dinosaur service accounts realm")
	fs.StringVar(&kc.BaseURL, "mas-sso-base-url", kc.BaseURL, "The base URL of the mas-sso, integration by default")
	fs.StringVar(&kc.DinosaurRealm.Realm, "mas-sso-realm", kc.DinosaurRealm.Realm, "Realm for Dinosaur service accounts in the mas-sso")
	fs.StringVar(&kc.TLSTrustedCertificatesFile, "mas-sso-cert-file", kc.TLSTrustedCertificatesFile, "File containing tls cert for the mas-sso. Useful when mas-sso uses a self-signed certificate. If the provided file does not exist, is the empty string or the provided file content is empty then no custom MAS SSO certificate is used")
	fs.BoolVar(&kc.Debug, "mas-sso-debug", kc.Debug, "Debug flag for Keycloak API")
	fs.BoolVar(&kc.InsecureSkipVerify, "mas-sso-insecure", kc.InsecureSkipVerify, "Disable tls verification with mas-sso")
	fs.StringVar(&kc.OSDClusterIDPRealm.ClientIDFile, "osd-idp-mas-sso-client-id-file", kc.OSDClusterIDPRealm.ClientIDFile, "File containing Keycloak privileged account client-id that has access to the OSD Cluster IDP realm")
	fs.StringVar(&kc.OSDClusterIDPRealm.ClientSecretFile, "osd-idp-mas-sso-client-secret-file", kc.OSDClusterIDPRealm.ClientSecretFile, "File containing Keycloak privileged account client-secret that has access to the OSD Cluster IDP realm")
	fs.StringVar(&kc.OSDClusterIDPRealm.Realm, "osd-idp-mas-sso-realm", kc.OSDClusterIDPRealm.Realm, "Realm for OSD cluster IDP clients in the mas-sso")
	fs.IntVar(&kc.MaxLimitForGetClients, "max-limit-for-sso-get-clients", kc.MaxLimitForGetClients, "Max limits for SSO get clients")
	fs.StringVar(&kc.UserNameClaim, "user-name-claim", kc.UserNameClaim, "Human readable username token claim")
	fs.StringVar(&kc.FallBackUserNameClaim, "fall-back-user-name-claim", kc.FallBackUserNameClaim, "Fall back username token claim")
	fs.BoolVar(&kc.KeycloakClientExpire, "keycloak-client-expire", kc.KeycloakClientExpire, "Whether or not to tag Keycloak created Client to expire in 2 hours (useful for cleaning up after integrations tests)")
}

func (kc *KeycloakConfig) ReadFiles() error {
	err := shared.ReadFileValueString(kc.DinosaurRealm.ClientIDFile, &kc.DinosaurRealm.ClientID)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(kc.DinosaurRealm.ClientSecretFile, &kc.DinosaurRealm.ClientSecret)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(kc.OSDClusterIDPRealm.ClientIDFile, &kc.OSDClusterIDPRealm.ClientID)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(kc.OSDClusterIDPRealm.ClientSecretFile, &kc.OSDClusterIDPRealm.ClientSecret)
	if err != nil {
		return err
	}

	// We read the MAS SSO TLS certificate file. If it does not exist we
	// intentionally continue as if it was not provided
	err = shared.ReadFileValueString(kc.TLSTrustedCertificatesFile, &kc.TLSTrustedCertificatesValue)
	if err != nil {
		if os.IsNotExist(err) {
			glog.V(10).Infof("Specified MAS SSO TLS certificate file '%s' does not exist. Proceeding as if MAS SSO TLS certificate was not provided", kc.TLSTrustedCertificatesFile)
		} else {
			return err
		}
	}

	kc.DinosaurRealm.setDefaultURIs(kc.BaseURL)
	kc.OSDClusterIDPRealm.setDefaultURIs(kc.BaseURL)
	return nil
}
