package presenters

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
)

// ConvertKafkaRequest from payload to KafkaRequest
func ConvertKafkaRequest(kafkaRequest public.KafkaRequestPayload) *dbapi.KafkaRequest {
	kafka := &dbapi.KafkaRequest{
		Region:        kafkaRequest.Region,
		Name:          kafkaRequest.Name,
		CloudProvider: kafkaRequest.CloudProvider,
		MultiAZ:       kafkaRequest.MultiAz,
	}

	if kafkaRequest.ReauthenticationEnabled != nil {
		kafka.ReauthenticationEnabled = *kafkaRequest.ReauthenticationEnabled
	} else {
		kafka.ReauthenticationEnabled = true // true by default
	}

	return kafka
}

// PresentKafkaRequest - create KafkaRequest in an appropriate format ready to be returned by the API
func PresentKafkaRequest(kafkaRequest *dbapi.KafkaRequest) public.KafkaRequest {
	reference := PresentReference(kafkaRequest.ID, kafkaRequest)

	return public.KafkaRequest{
		Id:                      reference.Id,
		Kind:                    reference.Kind,
		Href:                    reference.Href,
		Region:                  kafkaRequest.Region,
		Name:                    kafkaRequest.Name,
		CloudProvider:           kafkaRequest.CloudProvider,
		MultiAz:                 kafkaRequest.MultiAZ,
		Owner:                   kafkaRequest.Owner,
		BootstrapServerHost:     setBootstrapServerHost(kafkaRequest.BootstrapServerHost),
		Status:                  kafkaRequest.Status,
		CreatedAt:               kafkaRequest.CreatedAt,
		UpdatedAt:               kafkaRequest.UpdatedAt,
		FailedReason:            kafkaRequest.FailedReason,
		Version:                 kafkaRequest.ActualKafkaVersion,
		InstanceType:            kafkaRequest.InstanceType,
		ReauthenticationEnabled: kafkaRequest.ReauthenticationEnabled,
	}
}

func setBootstrapServerHost(bootstrapServerHost string) string {
	if bootstrapServerHost != "" {
		return fmt.Sprintf("%s:443", bootstrapServerHost)
	}
	return bootstrapServerHost
}
