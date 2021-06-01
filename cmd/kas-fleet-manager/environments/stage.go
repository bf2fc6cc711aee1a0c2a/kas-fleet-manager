package environments

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
)

var stageConfigDefaults map[string]string = map[string]string{
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

func loadStage(env *Env) error {
	env.DBFactory = db.NewConnectionFactory(env.Config.Database)

	err := env.LoadClients()
	if err != nil {
		return err
	}
	err = env.LoadServices()
	if err != nil {
		return err
	}

	return env.InitializeSentry()
}
