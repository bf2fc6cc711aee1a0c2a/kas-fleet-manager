package dbapi

import "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

type DataPlaneClusterStatus struct {
	Conditions               []DataPlaneClusterStatusCondition
	AvailableStrimziVersions []api.StrimziVersion
}

type DataPlaneClusterStatusCondition struct {
	Type    string
	Reason  string
	Status  string
	Message string
}

type DataPlaneClusterConfigObservability struct {
	AccessToken string
	Channel     string
	Repository  string
	Tag         string
}

type DataPlaneClusterConfig struct {
	Observability DataPlaneClusterConfigObservability
}
