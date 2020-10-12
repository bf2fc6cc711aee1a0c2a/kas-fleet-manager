package converters

import (
	"encoding/json"

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

// ConvertClusterList - converts []api.Cluster to the response type expected by mocket
func ConvertClusterList(clusterList []api.Cluster) ([]map[string]interface{}, error) {
	var convertedClusterList []map[string]interface{}

	for _, cluster := range clusterList {
		data, err := json.Marshal(cluster)
		if err != nil {
			return nil, err
		}

		var converted map[string]interface{}
		if err = json.Unmarshal(data, &converted); err != nil {
			return nil, err
		}
		convertedClusterList = append(convertedClusterList, converted)
	}

	return convertedClusterList, nil
}
