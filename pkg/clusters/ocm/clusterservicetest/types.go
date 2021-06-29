// Package clusterservicetest provides simple mock helpers for OCM Cluster Service types, to make table driven tests
// easier to define.
package clusterservicetest

import (
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const (
	MockClusterID            = "000"
	MockClusterExternalID    = "000"
	MockClusterMultiAZ       = true
	MockClusterCloudProvider = "aws"
	MockClusterRegion        = "us-east-1"
	MockClusterState         = "pending"
	MockClusterBYOC          = false
	MockClusterManaged       = true
)

// NewMockCluster create a default OCM Cluster Service cluster struct and apply modifications if provided.
func NewMockCluster(modifyFn func(*clustersmgmtv1.ClusterBuilder)) (*clustersmgmtv1.Cluster, error) {
	mock := clustersmgmtv1.NewCluster()
	mock.CloudProvider(clustersmgmtv1.NewCloudProvider().ID(MockClusterCloudProvider))
	mock.Region(clustersmgmtv1.NewCloudRegion().ID(MockClusterRegion))
	mock.CCS(clustersmgmtv1.NewCCS().Enabled(MockClusterBYOC))
	mock.Managed(MockClusterManaged)
	if modifyFn != nil {
		modifyFn(mock)
	}
	return mock.Build()
}
