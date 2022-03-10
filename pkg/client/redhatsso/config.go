package redhatsso

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/spf13/pflag"
)

const (
	defaultRealm = "redhat-external"
)

type RedhatSSOConfig struct {
	BaseURL    string      `json:"base_url"`
	KafkaRealm RealmConfig `json:"redhatsso_realm"`
}

type RealmConfig struct {
	Realm            string `json:"realm"`
	ClientID         string `json:"client-id"`
	ClientIDFile     string `json:"client-id_file"`
	ClientSecret     string `json:"client-secret"`
	ClientSecretFile string `json:"client-secret_file"`
	TokenEndpointURI string `json:"token_endpoint_uri"`
	APIEndpointURI   string `json:"api_endpoint_uri"`
}

func (c *RealmConfig) setDefaultURIs(baseURL string) {
	c.TokenEndpointURI = baseURL + "/auth/realms/" + c.Realm + "/protocol/openid-connect/token"
	c.APIEndpointURI = baseURL + "/auth/realms/" + c.Realm
}

func NewRedhatSSOConfig() *RedhatSSOConfig {
	cfg := &RedhatSSOConfig{
		KafkaRealm: RealmConfig{
			Realm:            defaultRealm,
			ClientIDFile:     "secrets/redhatsso-service.clientId",
			ClientSecretFile: "secrets/redhatsso-service.clientSecret",
		},
	}
	return cfg
}

func (rc *RedhatSSOConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&rc.KafkaRealm.ClientIDFile, "redhat-sso-client-id-file", rc.KafkaRealm.ClientIDFile, "File containing privileged account client-id that has access to the Kafka service accounts realm")
	fs.StringVar(&rc.KafkaRealm.ClientSecretFile, "redhat-sso-client-secret-file", rc.KafkaRealm.ClientSecretFile, "File containing privileged account client-secret that has access to the Kafka service accounts realm")
	fs.StringVar(&rc.BaseURL, "redhat-sso-base-url", rc.BaseURL, "The base URL of the redhat-sso, integration by default")
	fs.StringVar(&rc.KafkaRealm.Realm, "redhat-sso-realm", rc.KafkaRealm.Realm, "Realm for Kafka service accounts in the redhat-sso")
}

func (rc *RedhatSSOConfig) ReadFiles() error {
	err := shared.ReadFileValueString(rc.KafkaRealm.ClientIDFile, &rc.KafkaRealm.ClientID)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(rc.KafkaRealm.ClientSecretFile, &rc.KafkaRealm.ClientSecret)
	if err != nil {
		return err
	}

	rc.KafkaRealm.setDefaultURIs(rc.BaseURL)
	return nil
}
