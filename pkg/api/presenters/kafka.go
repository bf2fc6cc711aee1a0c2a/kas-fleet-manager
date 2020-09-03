package presenters

import (
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
)

func ConvertKafka(org openapi.Kafka) *api.Kafka {
	return &api.Kafka{
		Meta: api.Meta{
			ID: org.Id,
		},
		Region:    org.Region,
		Name:      org.Name,
		ClusterID: org.ClusterID,
	}
}

func PresentKafka(kafka *api.Kafka) openapi.Kafka {
	reference := PresentReference(kafka.ID, kafka)
	return openapi.Kafka{
		Id:        reference.Id,
		Kind:      reference.Kind,
		Href:      reference.Href,
		Region:    kafka.Region,
		Name:      kafka.Name,
		ClusterID: kafka.ClusterID,
		CreatedAt: kafka.CreatedAt,
		UpdatedAt: kafka.UpdatedAt,
	}
}
