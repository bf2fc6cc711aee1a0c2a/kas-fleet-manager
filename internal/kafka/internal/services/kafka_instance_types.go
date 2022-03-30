package services

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"k8s.io/apimachinery/pkg/api/resource"
)

//go:generate moq -out kafka_instance_types_moq.go . SupportedKafkaInstanceTypesService
type SupportedKafkaInstanceTypesService interface {
	GetSupportedKafkaInstanceTypesByRegion(providerId string, regionId string) ([]api.SupportedKafkaInstanceType, *errors.ServiceError)
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

func (t supportedKafkaInstanceTypesService) GetSupportedKafkaInstanceTypesByRegion(providerId string, regionId string) ([]api.SupportedKafkaInstanceType, *errors.ServiceError) {
	instanceTypeList := []api.SupportedKafkaInstanceType{}

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
		supportedSizesList := []api.SupportedKafkaSize{}
		for _, size := range sizes {
			supportedSizesList = append(supportedSizesList, api.SupportedKafkaSize{
				Id: size.Id,
				IngressThroughputPerSec: api.SupportedKafkaSizeBytesValueItem{
					Bytes: convertStringToValue(size.IngressThroughputPerSec),
				},
				EgressThroughputPerSec: api.SupportedKafkaSizeBytesValueItem{
					Bytes: convertStringToValue(size.EgressThroughputPerSec),
				},
				TotalMaxConnections: int32(size.TotalMaxConnections),
				MaxDataRetentionSize: api.SupportedKafkaSizeBytesValueItem{
					Bytes: convertStringToValue(size.MaxDataRetentionSize),
				},
				MaxPartitions:               int32(size.MaxPartitions),
				MaxDataRetentionPeriod:      size.MaxDataRetentionPeriod,
				MaxConnectionAttemptsPerSec: int32(size.MaxConnectionAttemptsPerSec),
				QuotaConsumed:               int32(size.QuotaConsumed),
				QuotaType:                   size.QuotaType,
				CapacityConsumed:            int32(size.CapacityConsumed),
			})
		}
		instanceTypeList = append(instanceTypeList, api.SupportedKafkaInstanceType{
			Id:                  k,
			SupportedKafkaSizes: supportedSizesList,
		})
	}

	return instanceTypeList, nil
}

//convertStringToValue converts string value used to create a resource.Quantity type to float32
//input string values have already been validated
func convertStringToValue(input string) float32 {
	q, _ := resource.ParseQuantity(input)
	value := float32(q.Value())
	return value
}
