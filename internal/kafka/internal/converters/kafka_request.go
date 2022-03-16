package converters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
)

func ConvertKafkaRequest(request *dbapi.KafkaRequest) []map[string]interface{} {
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
			"deleted_at":            request.Meta.DeletedAt.Time,
			"size_id":               request.SizeId,
			"instance_type":         request.InstanceType,
		},
	}
}

// ConvertKafkaRequestList converts a KafkaRequestList to the response type expected by mocket
func ConvertKafkaRequestList(kafkaList dbapi.KafkaList) []map[string]interface{} {
	var kafkaRequestList []map[string]interface{}

	for _, kafkaRequest := range kafkaList {
		data := ConvertKafkaRequest(kafkaRequest)
		kafkaRequestList = append(kafkaRequestList, data...)
	}

	return kafkaRequestList
}
