package dbapi

import "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

type DataPlaneClusterStatus struct {
	Conditions               []DataPlaneClusterStatusCondition
	NodeInfo                 DataPlaneClusterStatusNodeInfo
	ResizeInfo               DataPlaneClusterStatusResizeInfo
	Remaining                DataPlaneClusterStatusCapacity
	AvailableStrimziVersions []api.StrimziVersion
}

type DataPlaneClusterStatusCondition struct {
	Type    string
	Reason  string
	Status  string
	Message string
}

type DataPlaneClusterStatusNodeInfo struct {
	Ceiling                int
	Floor                  int
	Current                int
	CurrentWorkLoadMinimum int
}

type DataPlaneClusterStatusResizeInfo struct {
	NodeDelta int
	Delta     DataPlaneClusterStatusCapacity
}

type DataPlaneClusterStatusCapacity struct {
	IngressEgressThroughputPerSec string
	Connections                   int
	DataRetentionSize             string
	Partitions                    int
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
