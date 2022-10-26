package environments

import "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"

// The development environment is intended for use while developing features, requiring manual verification
func NewDevelopmentEnvLoader() environments.EnvLoader {
	return environments.SimpleEnvLoader{
		"v":                          "10",
		"ocm-debug":                  "false",
		"ocm-base-url":               "https://api.stage.openshift.com",
		"enable-ocm-mock":            "false",
		"enable-https":               "false",
		"enable-metrics-https":       "false",
		"enable-terms-acceptance":    "false",
		"api-server-bindaddress":     "localhost:8000",
		"sso-provider-type":          "mas_sso",
		"enable-sentry":              "false",
		"enable-deny-list":           "true",
		"enable-access-list":         "false",
		"mas-sso-base-url":           "http://127.0.0.1:8180",
		"redhat-sso-base-url":        "https://sso.stage.redhat.com",
		"mas-sso-realm":              "rhoas",
		"osd-idp-mas-sso-realm":      "rhoas-kafka-sre",
		"connector-eval-duration":    "48h",
		"admin-api-sso-base-url":     "https://identity.api.stage.openshift.com",
		"admin-api-sso-endpoint-uri": "/auth/realms/rhoas-kafka-sre",
		"admin-api-sso-realm":        "rhoas-kafka-sre",
	}
}
