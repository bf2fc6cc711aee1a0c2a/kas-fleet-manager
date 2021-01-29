package converters

import (
	"encoding/json"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
)

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

// ConvertKafkaRequestList converts a KafkaRequestList to the response type expected by mocket
func ConvertKafkaRequestList(kafkaList api.KafkaList) ([]map[string]interface{}, error) {
	var kafkaRequestList []map[string]interface{}

	for _, kafkaRequest := range kafkaList {
		data, err := json.Marshal(kafkaRequest)
		if err != nil {
			return nil, err
		}

		var converted map[string]interface{}
		if err = json.Unmarshal(data, &converted); err != nil {
			return nil, err
		}
		kafkaRequestList = append(kafkaRequestList, converted)
	}

	return kafkaRequestList, nil
}
