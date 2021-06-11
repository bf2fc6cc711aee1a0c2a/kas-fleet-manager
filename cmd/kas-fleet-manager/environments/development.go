package environments

// The development environment is intended for use while developing features, requiring manual verification
func newDevelopmentEnvLoader() EnvLoader {
	return SimpleEnvLoader{
		"v":                                 "10",
		"ocm-debug":                         "false",
		"ocm-base-url":                      "https://api.stage.openshift.com",
		"enable-ocm-mock":                   "false",
		"enable-https":                      "false",
		"enable-metrics-https":              "false",
		"enable-terms-acceptance":           "false",
		"api-server-bindaddress":            "localhost:8000",
		"enable-sentry":                     "false",
		"enable-deny-list":                  "true",
		"enable-instance-limit-control":     "false",
		"mas-sso-base-url":                  "https://identity.api.stage.openshift.com",
		"mas-sso-realm":                     "rhoas",
		"osd-idp-mas-sso-realm":             "rhoas-kafka-sre",
		"enable-kafka-external-certificate": "false",
		"cluster-compute-machine-type":      "m5.4xlarge",
		"ingress-controller-replicas":       "3",
		"quota-type":                        "allow-list",
		"enable-deletion-of-expired-kafka":  "true",
		"dataplane-cluster-scaling-type":    "manual",
		//TODO: change these values to the qe ones for development environment once they are available
		"strimzi-operator-addon-id": "managed-kafka",
		"kas-fleetshard-addon-id":   "kas-fleetshard-operator",
	}
}
