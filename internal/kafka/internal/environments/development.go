package environments

import "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"

// The development environment is intended for use while developing features, requiring manual verification
func NewDevelopmentEnvLoader() environments.EnvLoader {
	return environments.SimpleEnvLoader{
		"v":                             "10",
		"ocm-debug":                     "false",
		"ams-base-url":                  "https://api.stage.openshift.com",
		"ocm-base-url":                  "https://api.stage.openshift.com",
		"enable-ocm-mock":               "false",
		"enable-https":                  "false",
		"enable-metrics-https":          "false",
		"enable-terms-acceptance":       "false",
		"api-server-bindaddress":        "localhost:8000",
		"enable-sentry":                 "false",
		"enable-deny-list":              "true",
		"enable-access-list":            "false",
		"enable-instance-limit-control": "false",
		"mas-sso-base-url":              "http://127.0.0.1:8180",
		"redhat-sso-base-url":           "https://sso.stage.redhat.com",
		"mas-sso-realm":                 "rhoas",
		"sso-provider-type":             "mas_sso",
		"osd-idp-mas-sso-realm":         "rhoas-kafka-sre",
		"enable-kafka-sre-identity-provider-configuration": "false",
		"enable-kafka-external-certificate":                "false",
		"allow-developer-instance":                         "true",
		"quota-type":                                       "quota-management-list",
		"dataplane-cluster-scaling-type":                   "manual",
		"strimzi-operator-addon-id":                        "managed-kafka-qe",
		"kas-fleetshard-addon-id":                          "kas-fleetshard-operator-qe",
		"observability-red-hat-sso-token-refresher-url":    "http://localhost:8085",
		"observability-red-hat-sso-tenant":                 "managedkafka",
		"max-allowed-developer-instances":                  "1",
		"admin-api-sso-base-url":                           "http://127.0.0.1:8180",
		"admin-api-sso-endpoint-uri":                       "/auth/realms/rhoas-kafka-sre",
		"admin-api-sso-realm":                              "rhoas-kafka-sre",
		"dataplane-observability-config-enable":            "false",
	}
}
