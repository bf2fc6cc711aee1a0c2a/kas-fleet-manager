package converters

import (
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
)

// ConvertCluster from *api.Cluster to []map[string]interface{}
func ConvertCluster(cluster *api.Cluster) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"id":             cluster.Meta.ID,
			"region":         cluster.Region,
			"cloud_provider": cluster.CloudProvider,
			"multi_az":       cluster.MultiAZ,
			"managed":        cluster.Managed,
			"status":         cluster.Status,
			"byoc":           cluster.BYOC,
			"cluster_id":     cluster.ClusterID,
			"external_id":    cluster.ExternalID,
			"created_at":     cluster.Meta.CreatedAt,
			"updated_at":     cluster.Meta.UpdatedAt,
			"deleted_at":     cluster.Meta.DeletedAt,
		},
	}
}
