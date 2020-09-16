package converters

import (
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
)

// ConvertCluster convert an OCM cluster type from the Cluster Service to an internal cluster type for this
// service.
func ConvertCluster(cluster *clustersmgmtv1.Cluster) *api.Cluster {
	return &api.Cluster{
		CloudProvider: cluster.CloudProvider().ID(),
		ClusterID:     cluster.ID(),
		ExternalID:    cluster.ExternalID(),
		MultiAZ:       cluster.MultiAZ(),
		Region:        cluster.Region().ID(),
		BYOC:          cluster.BYOC(),
		Managed:       cluster.Managed(),
	}
}
