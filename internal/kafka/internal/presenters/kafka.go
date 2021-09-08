package presenters

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
)

// ConvertKafkaRequest from payload to KafkaRequest
func ConvertKafkaRequest(kafkaRequest public.DinosaurRequestPayload) *dbapi.KafkaRequest {
	return &dbapi.KafkaRequest{
		Region:        kafkaRequest.Region,
		Name:          kafkaRequest.Name,
		CloudProvider: kafkaRequest.CloudProvider,
		MultiAZ:       kafkaRequest.MultiAz,
	}
}

// PresentDinosaurRequest - create DinosaurRequest in an appropriate format ready to be returned by the API
func PresentKafkaRequest(kafkaRequest *dbapi.KafkaRequest) public.DinosaurRequest {
	reference := PresentReference(kafkaRequest.ID, kafkaRequest)

	return public.DinosaurRequest{
		Id:            reference.Id,
		Kind:          reference.Kind,
		Href:          reference.Href,
		Region:        kafkaRequest.Region,
		Name:          kafkaRequest.Name,
		CloudProvider: kafkaRequest.CloudProvider,
		MultiAz:       kafkaRequest.MultiAZ,
		Owner:         kafkaRequest.Owner,
		Status:        kafkaRequest.Status,
		CreatedAt:     kafkaRequest.CreatedAt,
		UpdatedAt:     kafkaRequest.UpdatedAt,
		FailedReason:  kafkaRequest.FailedReason,
		Version:       kafkaRequest.ActualKafkaVersion,
		InstanceType:  kafkaRequest.InstanceType,
	}
}

func setBootstrapServerHost(bootstrapServerHost string) string {
	if bootstrapServerHost != "" {
		return fmt.Sprintf("%s:443", bootstrapServerHost)
	}
	return bootstrapServerHost
}
