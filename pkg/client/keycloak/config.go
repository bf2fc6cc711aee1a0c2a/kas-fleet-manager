package keycloak

import (
	"fmt"
	"os"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"

	"github.com/golang/glog"
	"github.com/spf13/pflag"
)

const (
	MAS_SSO                       string = "mas_sso"
	REDHAT_SSO                    string = "redhat_sso"
	INTERNAL_SSO_REALM            string = "internal_sso"
	SSO_SPEICAL_MGMT_ORG_ID_STAGE string = "13640203"
	//AUTH_SSO SSOProvider ="auth_sso"
)

type KeycloakConfig struct {
	EnableAuthenticationOnKafka                bool                 `json:"enable_auth"`
	BaseURL                                    string               `json:"base_url"`
	SsoBaseUrl                                 string               `json:"sso_base_url"`
	Debug                                      bool                 `json:"debug"`
	InsecureSkipVerify                         bool                 `json:"insecure-skip-verify"`
	UserNameClaim                              string               `json:"user_name_claim"`
	FallBackUserNameClaim                      string               `json:"fall_back_user_name_claim"`
	TLSTrustedCertificatesKey                  string               `json:"tls_trusted_certificates_key"`
	TLSTrustedCertificatesValue                string               `json:"tls_trusted_certificates_value"`
	TLSTrustedCertificatesFile                 string               `json:"tls_trusted_certificates_file"`
	KafkaRealm                                 *KeycloakRealmConfig `json:"kafka_realm"`
	OSDClusterIDPRealm                         *KeycloakRealmConfig `json:"osd_cluster_idp_realm"`
	RedhatSSORealm                             *KeycloakRealmConfig `json:"redhat_sso_config"`
	AdminAPISSORealm                           *KeycloakRealmConfig `json:"internal_sso_config"`
	MaxAllowedServiceAccounts                  int                  `json:"max_allowed_service_accounts"`
	MaxLimitForGetClients                      int                  `json:"max_limit_for_get_clients"`
	SelectSSOProvider                          string               `json:"select_sso_provider"`
	SSOSpecialManagementOrgID                  string               `json:"-"`
	ServiceAccounttLimitCheckSkipOrgIdListFile string               `json:"-"`
	ServiceAccounttLimitCheckSkipOrgIdList     []string             `json:"-"`
}

type KeycloakRealmConfig struct {
	BaseURL          string `json:"base_url"`
	Realm            string `json:"realm"`
	ClientID         string `json:"client-id"`
	ClientIDFile     string `json:"client-id_file"`
	ClientSecret     string `json:"client-secret"`
	ClientSecretFile string `json:"client-secret_file"`
	GrantType        string `json:"grant_type"`
	TokenEndpointURI string `json:"token_endpoint_uri"`
	JwksEndpointURI  string `json:"jwks_endpoint_uri"`
	ValidIssuerURI   string `json:"valid_issuer_uri"`
	APIEndpointURI   string `json:"api_endpoint_uri"`
	Scope            string `json:"scope"`
}

func (kc *KeycloakConfig) SSOProviderRealm() *KeycloakRealmConfig {
	provider := kc.SelectSSOProvider
	switch provider {
	case MAS_SSO:
		return kc.KafkaRealm
	case REDHAT_SSO:
		return kc.RedhatSSORealm
	case INTERNAL_SSO_REALM:
		return kc.AdminAPISSORealm
	default:
		return kc.KafkaRealm
	}
}
func (c *KeycloakRealmConfig) setDefaultURIs(baseURL string) {
	c.BaseURL = baseURL
	c.ValidIssuerURI = baseURL + "/auth/realms/" + c.Realm
	c.JwksEndpointURI = baseURL + "/auth/realms/" + c.Realm + "/protocol/openid-connect/certs"
	c.TokenEndpointURI = baseURL + "/auth/realms/" + c.Realm + "/protocol/openid-connect/token"
}

func NewKeycloakConfig() *KeycloakConfig {
	kc := &KeycloakConfig{
		SsoBaseUrl:                  "https://sso.redhat.com",
		EnableAuthenticationOnKafka: true,
		KafkaRealm: &KeycloakRealmConfig{
			ClientIDFile:     "secrets/keycloak-service.clientId",
			ClientSecretFile: "secrets/keycloak-service.clientSecret",
			GrantType:        "client_credentials",
		},
		OSDClusterIDPRealm: &KeycloakRealmConfig{
			ClientIDFile:     "secrets/osd-idp-keycloak-service.clientId",
			ClientSecretFile: "secrets/osd-idp-keycloak-service.clientSecret",
			GrantType:        "client_credentials",
		},
		RedhatSSORealm: &KeycloakRealmConfig{
			APIEndpointURI:   "/auth/realms/redhat-external",
			Realm:            "redhat-external",
			ClientIDFile:     "secrets/redhatsso-service.clientId",
			ClientSecretFile: "secrets/redhatsso-service.clientSecret",
			GrantType:        "client_credentials",
			Scope:            "api.iam.service_accounts",
		},
		AdminAPISSORealm: &KeycloakRealmConfig{
			BaseURL:        "https://auth.redhat.com",
			APIEndpointURI: "/auth/realms/EmployeeIDP",
			Realm:          "EmployeeIDP",
		},
		TLSTrustedCertificatesFile:                 "secrets/keycloak-service.crt",
		Debug:                                      false,
		InsecureSkipVerify:                         false,
		UserNameClaim:                              "clientId",
		FallBackUserNameClaim:                      "preferred_username",
		TLSTrustedCertificatesKey:                  "keycloak.crt",
		MaxAllowedServiceAccounts:                  50,
		MaxLimitForGetClients:                      100,
		SelectSSOProvider:                          MAS_SSO,
		SSOSpecialManagementOrgID:                  SSO_SPEICAL_MGMT_ORG_ID_STAGE,
		ServiceAccounttLimitCheckSkipOrgIdListFile: "config/service-account-limits-check-skip-org-id-list.yaml",
	}
	return kc
}

func (kc *KeycloakConfig) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&kc.EnableAuthenticationOnKafka, "mas-sso-enable-auth", kc.EnableAuthenticationOnKafka, "Enable authentication mas-sso integration, enabled by default")
	fs.StringVar(&kc.KafkaRealm.ClientIDFile, "mas-sso-client-id-file", kc.KafkaRealm.ClientIDFile, "File containing Keycloak privileged account client-id that has access to the Kafka service accounts realm")
	fs.StringVar(&kc.KafkaRealm.ClientSecretFile, "mas-sso-client-secret-file", kc.KafkaRealm.ClientSecretFile, "File containing Keycloak privileged account client-secret that has access to the Kafka service accounts realm")
	fs.StringVar(&kc.BaseURL, "mas-sso-base-url", kc.BaseURL, "The base URL of the mas-sso, integration by default")
	fs.StringVar(&kc.KafkaRealm.Realm, "mas-sso-realm", kc.KafkaRealm.Realm, "Realm for Kafka service accounts in the mas-sso")
	fs.StringVar(&kc.TLSTrustedCertificatesFile, "mas-sso-cert-file", kc.TLSTrustedCertificatesFile, "File containing tls cert for the mas-sso. Useful when mas-sso uses a self-signed certificate. If the provided file does not exist, is the empty string or the provided file content is empty then no custom MAS SSO certificate is used")
	fs.BoolVar(&kc.Debug, "mas-sso-debug", kc.Debug, "Debug flag for Keycloak API")
	fs.StringVar(&kc.RedhatSSORealm.Scope, "redhat-sso-scope", kc.RedhatSSORealm.Scope, "Scope for client credentials grant request in sso")
	fs.BoolVar(&kc.InsecureSkipVerify, "mas-sso-insecure", kc.InsecureSkipVerify, "Disable tls verification with mas-sso")
	fs.StringVar(&kc.OSDClusterIDPRealm.ClientIDFile, "osd-idp-mas-sso-client-id-file", kc.OSDClusterIDPRealm.ClientIDFile, "File containing Keycloak privileged account client-id that has access to the OSD Cluster IDP realm")
	fs.StringVar(&kc.OSDClusterIDPRealm.ClientSecretFile, "osd-idp-mas-sso-client-secret-file", kc.OSDClusterIDPRealm.ClientSecretFile, "File containing Keycloak privileged account client-secret that has access to the OSD Cluster IDP realm")
	fs.StringVar(&kc.OSDClusterIDPRealm.Realm, "osd-idp-mas-sso-realm", kc.OSDClusterIDPRealm.Realm, "Realm for OSD cluster IDP clients in the mas-sso")
	fs.IntVar(&kc.MaxAllowedServiceAccounts, "max-allowed-service-accounts", kc.MaxAllowedServiceAccounts, "Max allowed service accounts per org")
	fs.IntVar(&kc.MaxLimitForGetClients, "max-limit-for-sso-get-clients", kc.MaxLimitForGetClients, "Max limits for SSO get clients")
	fs.StringVar(&kc.UserNameClaim, "user-name-claim", kc.UserNameClaim, "Human readable username token claim")
	fs.StringVar(&kc.FallBackUserNameClaim, "fall-back-user-name-claim", kc.FallBackUserNameClaim, "Fall back username token claim")
	fs.StringVar(&kc.RedhatSSORealm.ClientIDFile, "redhat-sso-client-id-file", kc.RedhatSSORealm.ClientIDFile, "File containing Keycloak privileged account client-id that has access to the OSD Cluster IDP realm")
	fs.StringVar(&kc.RedhatSSORealm.ClientSecretFile, "redhat-sso-client-secret-file", kc.RedhatSSORealm.ClientSecretFile, "File containing Keycloak privileged account client-secret that has access to the OSD Cluster IDP realm")
	fs.StringVar(&kc.SsoBaseUrl, "redhat-sso-base-url", kc.SsoBaseUrl, "The base URL of the mas-sso, integration by default")
	fs.StringVar(&kc.SSOSpecialManagementOrgID, "sso-special-management-org-id", SSO_SPEICAL_MGMT_ORG_ID_STAGE, "The Special Management Organization ID used for creating internal Service accounts")
	fs.StringVar(&kc.ServiceAccounttLimitCheckSkipOrgIdListFile, "service-account-limits-check-skip-org-id-list-file", kc.ServiceAccounttLimitCheckSkipOrgIdListFile, "File containing a list of Org IDs for which service account limits check will be skipped")
	fs.StringVar(&kc.SelectSSOProvider, "sso-provider-type", kc.SelectSSOProvider, "Option to choose between sso providers i.e, mas_sso or redhat_sso, mas_sso by default")
	fs.StringVar(&kc.AdminAPISSORealm.BaseURL, "admin-api-sso-base-url", kc.AdminAPISSORealm.BaseURL, "Base url of admin api sso realm, 'https://auth.redhat.com' by default")
	fs.StringVar(&kc.AdminAPISSORealm.APIEndpointURI, "admin-api-sso-endpoint-uri", kc.AdminAPISSORealm.APIEndpointURI, "API Endpoint URI of admin api sso realm, '/auth/realms/EmployeeIDP' by default")
	fs.StringVar(&kc.AdminAPISSORealm.Realm, "admin-api-sso-realm", kc.AdminAPISSORealm.Realm, "Admin api sso realm, 'EmployeeIDP' by default")
}

func (kc *KeycloakConfig) Validate(env *environments.Env) error {
	if kc.SelectSSOProvider != REDHAT_SSO && kc.SelectSSOProvider != MAS_SSO {
		return fmt.Errorf("Invalid sso provider selected must be `mas_sso` or `redhat_sso`")
	}
	return nil
}

func (kc *KeycloakConfig) ReadFiles() error {
	err := shared.ReadFileValueString(kc.KafkaRealm.ClientIDFile, &kc.KafkaRealm.ClientID)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(kc.KafkaRealm.ClientSecretFile, &kc.KafkaRealm.ClientSecret)
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
	err = shared.ReadFileValueString(kc.OSDClusterIDPRealm.ClientSecretFile, &kc.OSDClusterIDPRealm.ClientSecret)
	if err != nil {
		return err
	}
	if kc.SelectSSOProvider == REDHAT_SSO {
		err = shared.ReadFileValueString(kc.RedhatSSORealm.ClientIDFile, &kc.RedhatSSORealm.ClientID)
		if err != nil {
			return err
		}
		err = shared.ReadFileValueString(kc.RedhatSSORealm.ClientSecretFile, &kc.RedhatSSORealm.ClientSecret)
		if err != nil {
			return err
		}
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

	//Read the service account limits check skip org ID yaml file
	err = shared.ReadYamlFile(kc.ServiceAccounttLimitCheckSkipOrgIdListFile, &kc.ServiceAccounttLimitCheckSkipOrgIdList)
	if err != nil {
		if os.IsNotExist(err) {
			glog.V(10).Infof("Specified service account limits skip org IDs file '%s' does not exist. Proceeding as if no service account org ID skip list was provided", kc.ServiceAccounttLimitCheckSkipOrgIdListFile)
		} else {
			return err
		}
	}

	kc.KafkaRealm.setDefaultURIs(kc.BaseURL)
	kc.OSDClusterIDPRealm.setDefaultURIs(kc.BaseURL)
	kc.RedhatSSORealm.setDefaultURIs(kc.SsoBaseUrl)
	kc.AdminAPISSORealm.setDefaultURIs((kc.AdminAPISSORealm.BaseURL))
	return nil
}
