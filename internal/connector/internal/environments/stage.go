package environments

import "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"

func NewStageEnvLoader() environments.EnvLoader {
	return environments.SimpleEnvLoader{
		"ocm-base-url":            "https://api.stage.openshift.com",
		"enable-ocm-mock":         "false",
		"enable-deny-list":        "true",
		"max-allowed-instances":   "1",
		"mas-sso-base-url":        "https://identity.api.stage.openshift.com",
		"mas-sso-realm":           "rhoas",
		"connector-eval-duration": "48h",
	}
}
