package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
)

func PresentSupportedKafkaInstanceType(supportedInstanceType *public.SupportedKafkaInstanceType) public.SupportedKafkaInstanceType {
	reference := PresentReference(supportedInstanceType.Id, supportedInstanceType)
	return public.SupportedKafkaInstanceType{
		Id:    reference.Id,
		Sizes: GetSupportedKafkaSizes(supportedInstanceType.Sizes),
	}
}

func GetSupportedKafkaSizes(supportedKafkaSizes []public.SupportedKafkaSize) []public.SupportedKafkaSize {
	sizes := make([]public.SupportedKafkaSize, len(supportedKafkaSizes))
	for _, c := range supportedKafkaSizes {
		sizes = append(sizes, public.SupportedKafkaSize{Id: c.Id, IngressThroughputPerSec: public.SupportedKafkaSizeBytesValueItem(c.IngressThroughputPerSec), EgressThroughputPerSec: public.SupportedKafkaSizeBytesValueItem(c.EgressThroughputPerSec),
			TotalMaxConnections: c.TotalMaxConnections, MaxDataRetentionSize: public.SupportedKafkaSizeBytesValueItem(c.MaxDataRetentionSize), MaxPartitions: c.MaxPartitions, MaxDataRetentionPeriod: c.MaxDataRetentionPeriod,
			MaxConnectionAttemptsPerSec: c.MaxConnectionAttemptsPerSec, QuotaConsumed: c.QuotaConsumed, QuotaType: c.QuotaType, CapacityConsumed: c.CapacityConsumed})
	}
	return sizes
}
