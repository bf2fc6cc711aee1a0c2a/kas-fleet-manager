package environments

func newProductionEnvLoader() EnvLoader {
	return SimpleEnvLoader{
		"v":                                 "1",
		"ocm-debug":                         "false",
		"enable-ocm-mock":                   "false",
		"enable-sentry":                     "true",
		"enable-deny-list":                  "true",
		"max-allowed-instances":             "1",
		"mas-sso-realm":                     "rhoas",
		"mas-sso-base-url":                  "https://identity.api.openshift.com",
		"enable-kafka-external-certificate": "true",
		"cluster-compute-machine-type":      "m5.4xlarge",
		"ingress-controller-replicas":       "9",
	}
}
