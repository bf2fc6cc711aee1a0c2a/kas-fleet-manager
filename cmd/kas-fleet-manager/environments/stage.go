package environments

func newStageEnvLoader() EnvLoader {
	return SimpleEnvLoader{
		"ocm-base-url":                      "https://api.stage.openshift.com",
		"enable-ocm-mock":                   "false",
		"enable-deny-list":                  "true",
		"max-allowed-instances":             "1",
		"mas-sso-base-url":                  "https://identity.api.stage.openshift.com",
		"mas-sso-realm":                     "rhoas",
		"enable-kafka-external-certificate": "true",
		"cluster-compute-machine-type":      "m5.4xlarge",
		"ingress-controller-replicas":       "9",
	}
}
