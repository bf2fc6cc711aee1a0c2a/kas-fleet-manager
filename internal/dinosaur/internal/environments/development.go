package environments

import "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/environments"

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
		"sso-base-url":                                    "https://identity.api.stage.openshift.com",
		"sso-realm":                                       "rhoas",
		"osd-idp-sso-realm":                               "rhoas-dinosaur-sre",
		"enable-dinosaur-external-certificate":            "false",
		"cluster-compute-machine-type":                    "m5.2xlarge",
		"allow-evaluator-instance":                        "true",
		"quota-type":                                      "quota-management-list",
		"enable-deletion-of-expired-dinosaur":             "true",
		"dataplane-cluster-scaling-type":                  "manual",
		"dinosaur-operator-addon-id":                      "managed-dinosaur-qe",
		"fleetshard-addon-id":                             "fleetshard-operator-qe",
		"observability-red-hat-sso-auth-server-url":       "https://sso.redhat.com/auth",
		"observability-red-hat-sso-realm":                 "redhat-external",
		"observability-red-hat-sso-token-refresher-url":   "http://localhost:8085",
		"observability-red-hat-sso-observatorium-gateway": "https://observatorium-mst.api.stage.openshift.com",
		"observability-red-hat-sso-tenant":                "manageddinosaur",
	}
}
