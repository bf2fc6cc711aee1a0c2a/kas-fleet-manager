package presenters

import (
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
)

func ConvertKafkaRequest(kafkaRequest openapi.KafkaRequest) *api.KafkaRequest {
	return &api.KafkaRequest{
		Meta: api.Meta{
			ID: kafkaRequest.Id,
		},
		Region:        kafkaRequest.Region,
		Name:          kafkaRequest.Name,
		ClusterID:     kafkaRequest.ClusterID,
		CloudProvider: kafkaRequest.CloudProvider,
		MultiAZ:       kafkaRequest.MultiAz,
		Owner:         kafkaRequest.Owner,
	}
}

func PresentKafkaRequest(kafkaRequest *api.KafkaRequest) openapi.KafkaRequest {
	reference := PresentReference(kafkaRequest.ID, kafkaRequest)
	return openapi.KafkaRequest{
		Id:            reference.Id,
		Kind:          reference.Kind,
		Href:          reference.Href,
		Region:        kafkaRequest.Region,
		Name:          kafkaRequest.Name,
		ClusterID:     kafkaRequest.ClusterID,
		CloudProvider: kafkaRequest.CloudProvider,
		MultiAz:       kafkaRequest.MultiAZ,
		Owner:         kafkaRequest.Owner,
		Status:        kafkaRequest.Status,
		CreatedAt:     kafkaRequest.CreatedAt,
		UpdatedAt:     kafkaRequest.UpdatedAt,
	}
}
