package api

import (
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

type Cluster struct {
	Meta
	CloudProvider string `json:"cloud_provider"`
	ClusterID     string
	ExternalID    string
	MultiAZ       bool   `json:"multi_az"`
	Region        string `json:"region"`
	State         clustersmgmtv1.ClusterState
}
