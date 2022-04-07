package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
)

func PresentSupportedKafkaInstanceTypes(supportedInstanceType *config.KafkaInstanceType) public.SupportedKafkaInstanceType {
	reference := PresentReference(supportedInstanceType.Id, supportedInstanceType)
	return public.SupportedKafkaInstanceType{
		Id:          reference.Id,
		DisplayName: supportedInstanceType.DisplayName,
		Sizes:       GetSupportedSizes(supportedInstanceType),
	}
}

func GetSupportedSizes(supportedInstanceType *config.KafkaInstanceType) []public.SupportedKafkaSize {
	supportedSizes := make([]public.SupportedKafkaSize, len(supportedInstanceType.Sizes))
	for i, size := range supportedInstanceType.Sizes {
		//errors from Quantity.ToFloat32() ignored. Strings already validated as resource.Quantity
		ingressBytes, _ := size.IngressThroughputPerSec.ToFloat32()
		egressBytes, _ := size.EgressThroughputPerSec.ToFloat32()
		retentionSizeBytes, _ := size.MaxDataRetentionSize.ToFloat32()
		supportedSizes[i] = public.SupportedKafkaSize{
			Id: size.Id,
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
			QuotaConsumed:               int32(size.QuotaConsumed),
			QuotaType:                   size.QuotaType,
			CapacityConsumed:            int32(size.CapacityConsumed),
		}
	}
	return supportedSizes
}
