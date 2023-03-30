package environments

import "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"

func NewStageEnvLoader() environments.EnvLoader {
	return environments.SimpleEnvLoader{
		"ocm-base-url":            "https://api.stage.openshift.com",
		"enable-ocm-mock":         "false",
		"enable-deny-list":        "true",
		"enable-access-list":      "false",
		"mas-sso-base-url":        "https://identity.api.stage.openshift.com",
		"mas-sso-realm":           "rhoas",
		"connector-eval-duration": "48h",
		"processors-enabled":      "false",
	}
}
