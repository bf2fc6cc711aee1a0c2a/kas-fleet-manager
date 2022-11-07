package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
)

func PresentSupportedKafkaInstanceTypes(supportedInstanceType *config.KafkaInstanceType) public.SupportedKafkaInstanceType {
	reference := PresentReference(supportedInstanceType.Id, supportedInstanceType)
	return public.SupportedKafkaInstanceType{
		Id:                     reference.Id,
		DisplayName:            supportedInstanceType.DisplayName,
		SupportedBillingModels: GetSupportedBillingModels(supportedInstanceType),
		Sizes:                  GetSupportedSizes(supportedInstanceType),
	}
}

func GetSupportedBillingModels(supportedInstanceType *config.KafkaInstanceType) []public.SupportedKafkaBillingModel {
	supportedBillingModels := make([]public.SupportedKafkaBillingModel, len(supportedInstanceType.SupportedBillingModels))

	for i, bm := range supportedInstanceType.SupportedBillingModels {
		supportedBillingModels[i] = public.SupportedKafkaBillingModel{
			Id:               bm.ID,
			AmsResource:      bm.AMSResource,
			AmsProduct:       bm.AMSProduct,
			AmsBillingModels: bm.AMSBillingModels,
		}
	}

	return supportedBillingModels
}

func GetSupportedSizes(supportedInstanceType *config.KafkaInstanceType) []public.SupportedKafkaSize {
	supportedSizes := make([]public.SupportedKafkaSize, len(supportedInstanceType.Sizes))
	for i, size := range supportedInstanceType.Sizes {
		//errors from Quantity.ToInt64() ignored. Strings already validated as resource.Quantity
		ingressBytes, _ := size.IngressThroughputPerSec.ToInt64()
		egressBytes, _ := size.EgressThroughputPerSec.ToInt64()
		retentionSizeBytes, _ := size.MaxDataRetentionSize.ToInt64()
		maxMessageSizeBytes, _ := size.MaxMessageSize.ToInt64()
		var lifespanSeconds *int32
		if size.LifespanSeconds != nil {
			tmpLifespanSeconds := int32(*size.LifespanSeconds)
			lifespanSeconds = &tmpLifespanSeconds
		}
		supportedSizes[i] = public.SupportedKafkaSize{
			Id:          size.Id,
			DisplayName: size.DisplayName,
			IngressThroughputPerSec: public.SupportedKafkaSizeBytesValueItem{
				Bytes: ingressBytes,
			},
			EgressThroughputPerSec: public.SupportedKafkaSizeBytesValueItem{
				Bytes: egressBytes,
			},
			TotalMaxConnections: int32(size.TotalMaxConnections),
			MaxDataRetentionSize: public.SupportedKafkaSizeBytesValueItem{
				Bytes: retentionSizeBytes,
			},
			MaxPartitions:               int32(size.MaxPartitions),
			MaxDataRetentionPeriod:      size.MaxDataRetentionPeriod,
			MaxConnectionAttemptsPerSec: int32(size.MaxConnectionAttemptsPerSec),
			MaxMessageSize: public.SupportedKafkaSizeBytesValueItem{
				Bytes: maxMessageSizeBytes,
			},
			MinInSyncReplicas: int32(size.MinInSyncReplicas),
			ReplicationFactor: int32(size.ReplicationFactor),
			QuotaConsumed:     int32(size.QuotaConsumed),
			QuotaType:         size.QuotaType,
			CapacityConsumed:  int32(size.CapacityConsumed),
			SupportedAzModes:  size.SupportedAZModes,
			LifespanSeconds:   lifespanSeconds,
			MaturityStatus:    string(size.MaturityStatus),
		}
	}
	return supportedSizes
}
