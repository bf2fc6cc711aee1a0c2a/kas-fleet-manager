package environments

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
)

// The development environment is intended for use while developing features, requiring manual verification
var developmentConfigDefaults map[string]string = map[string]string{
	"v":                                 "10",
	"enable-authz":                      "true",
	"ocm-debug":                         "false",
	"ocm-base-url":                      "https://api.stage.openshift.com",
	"enable-ocm-mock":                   "false",
	"enable-https":                      "false",
	"enable-metrics-https":              "false",
	"api-server-hostname":               "localhost",
	"api-server-bindaddress":            "localhost:8000",
	"enable-sentry":                     "false",
	"enable-allow-list":                 "true",
	"enable-deny-list":                  "true",
	"max-allowed-instances":             "3",
	"auto-osd-creation":                 "false",
	"mas-sso-base-url":                  "https://keycloak-edge-redhat-rhoam-user-sso.apps.mas-sso-stage.1gzl.s1.devshift.org",
	"mas-sso-realm":                     "mas-sso-playground",
	"osd-idp-mas-sso-realm":             "mas-sso-playground",
	"enable-kafka-external-certificate": "false",
	"cluster-compute-machine-type":      "m5.xlarge",
	"ingress-controller-replicas":       "3",
	"enable-quota-service":              "false",
}

func loadDevelopment(env *Env) error {
	env.DBFactory = db.NewConnectionFactory(env.Config.Database)

	err := env.LoadClients()
	if err != nil {
		return err
	}
	err = env.LoadServices()
	if err != nil {
		return err
	}

	return env.InitializeSentry()
}
