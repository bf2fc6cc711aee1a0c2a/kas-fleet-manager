package converters

import "gitlab.cee.redhat.com/service/managed-services-api/pkg/api"

func ConvertKafkaRequest(request *api.KafkaRequest) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"id":                    request.ID,
			"region":                request.Region,
			"cloud_provider":        request.CloudProvider,
			"multi_az":              request.MultiAZ,
			"name":                  request.Name,
			"status":                request.Status,
			"owner":                 request.Owner,
			"cluster_id":            request.ClusterID,
			"bootstrap_server_host": request.BootstrapServerHost,
			"created_at":            request.Meta.CreatedAt,
			"updated_at":            request.Meta.UpdatedAt,
			"deleted_at":            request.Meta.DeletedAt,
		},
	}
}
