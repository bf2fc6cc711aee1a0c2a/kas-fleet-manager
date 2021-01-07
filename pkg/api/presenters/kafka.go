package presenters

import (
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/constants"
)

func ConvertKafkaRequest(kafkaRequest openapi.KafkaRequestPayload) *api.KafkaRequest {
	return &api.KafkaRequest{
		Region:        kafkaRequest.Region,
		Name:          kafkaRequest.Name,
		CloudProvider: kafkaRequest.CloudProvider,
		MultiAZ:       kafkaRequest.MultiAz,
	}
}

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
		BootstrapServerHost: kafkaRequest.BootstrapServerHost,
		Status:              setStatus(kafkaRequest.Status),
		CreatedAt:           kafkaRequest.CreatedAt,
		UpdatedAt:           kafkaRequest.UpdatedAt,
		FailedReason:        kafkaRequest.FailedReason,
	}
}

//set the status provisioning if it's resource_creating on UI
func setStatus(status string) string {
	if status == constants.KafkaRequestStatusResourceCreation.String() {
		return constants.KafkaRequestStatusProvisioning.String()
	}
	return status
}
