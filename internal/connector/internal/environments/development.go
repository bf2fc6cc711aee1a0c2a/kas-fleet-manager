package environments

import "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"

// The development environment is intended for use while developing features, requiring manual verification
func NewDevelopmentEnvLoader() environments.EnvLoader {
	return environments.SimpleEnvLoader{
		"v":                             "10",
		"ocm-debug":                     "false",
		"ocm-base-url":                  "https://api.stage.openshift.com",
		"enable-ocm-mock":               "false",
		"enable-https":                  "false",
		"enable-metrics-https":          "false",
		"enable-terms-acceptance":       "false",
		"api-server-bindaddress":        "localhost:8000",
		"enable-sentry":                 "false",
		"enable-deny-list":              "true",
		"enable-instance-limit-control": "false",
		"mas-sso-base-url":              "https://identity.api.stage.openshift.com",
		"mas-sso-realm":                 "rhoas",
		"osd-idp-mas-sso-realm":         "rhoas-kafka-sre",
		"connector-eval-duration":       "48h",
	}
}
