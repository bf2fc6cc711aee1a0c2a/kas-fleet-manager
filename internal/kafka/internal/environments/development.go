package environments

import "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"

// The development environment is intended for use while developing features, requiring manual verification
func NewDevelopmentEnvLoader() environments.EnvLoader {
	return environments.SimpleEnvLoader{
		"v":                                               "10",
		"ocm-debug":                                       "false",
		"ams-base-url":                                    "https://api.stage.openshift.com",
		"ocm-base-url":                                    "https://api.stage.openshift.com",
		"enable-ocm-mock":                                 "false",
		"enable-https":                                    "false",
		"enable-metrics-https":                            "false",
		"enable-terms-acceptance":                         "false",
		"api-server-bindaddress":                          "localhost:8000",
		"enable-sentry":                                   "false",
		"enable-deny-list":                                "true",
		"enable-instance-limit-control":                   "false",
		"mas-sso-base-url":                                "https://identity.api.stage.openshift.com",
		"mas-sso-realm":                                   "rhoas",
		"osd-idp-mas-sso-realm":                           "rhoas-kafka-sre",
		"enable-kafka-external-certificate":               "false",
		"cluster-compute-machine-type":                    "m5.2xlarge",
		"allow-developer-instance":                        "true",
		"quota-type":                                      "quota-management-list",
		"enable-deletion-of-expired-kafka":                "true",
		"dataplane-cluster-scaling-type":                  "manual",
		"strimzi-operator-addon-id":                       "managed-kafka-qe",
		"kas-fleetshard-addon-id":                         "kas-fleetshard-operator-qe",
		"observability-red-hat-sso-auth-server-url":       "https://sso.redhat.com/auth",
		"observability-red-hat-sso-realm":                 "redhat-external",
		"observability-red-hat-sso-token-refresher-url":   "http://localhost:8085",
		"observability-red-hat-sso-observatorium-gateway": "https://observatorium-mst.api.stage.openshift.com",
		"observability-red-hat-sso-tenant":                "managedkafka",
		"observatorium-auth-type":                         "dex",
	}
}
