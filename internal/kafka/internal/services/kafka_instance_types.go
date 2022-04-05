package services

import (
	"fmt"
	"sort"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

//go:generate moq -out kafka_instance_types_moq.go . SupportedKafkaInstanceTypesService
type SupportedKafkaInstanceTypesService interface {
	GetSupportedKafkaInstanceTypesByRegion(providerId string, regionId string) ([]public.SupportedKafkaInstanceType, *errors.ServiceError)
}

type supportedKafkaInstanceTypesService struct {
	providerConfig *config.ProviderConfig
	kafkaConfig    *config.KafkaConfig
}

func NewSupportedKafkaInstanceTypesService(providerConfig *config.ProviderConfig, kafkaConfig *config.KafkaConfig) SupportedKafkaInstanceTypesService {
	return &supportedKafkaInstanceTypesService{
		providerConfig: providerConfig,
		kafkaConfig:    kafkaConfig,
	}
}

func (t supportedKafkaInstanceTypesService) GetSupportedKafkaInstanceTypesByRegion(providerId string, regionId string) ([]public.SupportedKafkaInstanceType, *errors.ServiceError) {
	instanceTypeList := []public.SupportedKafkaInstanceType{}
	provider, providerFound := t.providerConfig.ProvidersConfig.SupportedProviders.GetByName(providerId)
	if !providerFound {
		return nil, errors.ProviderNotSupported(fmt.Sprintf("cloud provider '%s' is unsupported", providerId))
	}

	region, regionFound := provider.Regions.GetByName(regionId)
	if !regionFound {
		return nil, errors.RegionNotSupported(fmt.Sprintf("cloud region '%s' is unsupported", regionId))
	}

	for k := range region.SupportedInstanceTypes {
		instanceType, err := t.kafkaConfig.SupportedInstanceTypes.Configuration.GetKafkaInstanceTypeByID(k)
		if err != nil {
			return nil, errors.InstanceTypeNotSupported(fmt.Sprintf("instance type '%s' is unsupported", k))
		}
		sizes := instanceType.Sizes
		supportedSizesList := []public.SupportedKafkaSize{}
		for _, size := range sizes {
			//errors from Quantity.ToFloat32() ignored. Strings already validated as resource.Quantity
			ingressBytes, _ := size.IngressThroughputPerSec.ToFloat32()
			egressBytes, _ := size.EgressThroughputPerSec.ToFloat32()
			retentionSizeBytes, _ := size.MaxDataRetentionSize.ToFloat32()
			supportedSizesList = append(supportedSizesList, public.SupportedKafkaSize{
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
			})
		}
		instanceTypeList = append(instanceTypeList, public.SupportedKafkaInstanceType{
			Id:          k,
			DisplayName: instanceType.DisplayName,
			Sizes:       supportedSizesList,
		})
	}
	sort.Slice(instanceTypeList, func(i, j int) bool {
		return instanceTypeList[i].Id < instanceTypeList[j].Id
	})

	return instanceTypeList, nil
}
