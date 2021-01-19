package presenters

import (
	"fmt"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
)

// ConvertKafkaRequest from payload to KafkaRequest
func ConvertKafkaRequest(kafkaRequest openapi.KafkaRequestPayload) *api.KafkaRequest {
	return &api.KafkaRequest{
		Region:        kafkaRequest.Region,
		Name:          kafkaRequest.Name,
		CloudProvider: kafkaRequest.CloudProvider,
		MultiAZ:       kafkaRequest.MultiAz,
	}
}

// PresentKafkaRequest - create KafkaRequest in an appropriate format ready to be returned by the API
func PresentKafkaRequest(kafkaRequest *api.KafkaRequest) openapi.KafkaRequest {
	reference := PresentReference(kafkaRequest.ID, kafkaRequest)
	return openapi.KafkaRequest{
		Id:                  reference.Id,
		Kind:                reference.Kind,
		Href:                reference.Href,
		Region:              kafkaRequest.Region,
		Name:                kafkaRequest.Name,
		CloudProvider:       kafkaRequest.CloudProvider,
		MultiAz:             kafkaRequest.MultiAZ,
		Owner:               kafkaRequest.Owner,
		BootstrapServerHost: setBootstrapServerHost(kafkaRequest.BootstrapServerHost),
		Status:              kafkaRequest.Status,
		CreatedAt:           kafkaRequest.CreatedAt,
		UpdatedAt:           kafkaRequest.UpdatedAt,
		FailedReason:        kafkaRequest.FailedReason,
	}
}

func setBootstrapServerHost(bootstrapServerHost string) string {
	if bootstrapServerHost != "" {
		return fmt.Sprintf("%s:443", bootstrapServerHost)
	}
	return bootstrapServerHost
}
