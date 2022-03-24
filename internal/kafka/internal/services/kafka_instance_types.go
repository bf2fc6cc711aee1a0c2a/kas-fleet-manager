package services

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
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
			return nil, errors.InstanceTypeNotSupported(fmt.Sprintf("instance type '%s' is unsupported", err.Error()))
		}
		sizes := instanceType.Sizes
		alreadyCollectedSize := map[string]bool{}
		supportedSizesList := []api.SupportedKafkaSize{}
		for _, size := range sizes {
			_, sizeCollected := alreadyCollectedSize[size.Id]
			if sizeCollected {
				continue
			}
			supportedSizesList = append(supportedSizesList, api.SupportedKafkaSize{
				Id:                          size.Id,
				IngressThroughputPerSec:     size.IngressThroughputPerSec,
				EgressThroughputPerSec:      size.EgressThroughputPerSec,
				TotalMaxConnections:         int32(size.TotalMaxConnections),
				MaxDataRetentionSize:        size.MaxDataRetentionSize,
				MaxPartitions:               int32(size.MaxPartitions),
				MaxDataRetentionPeriod:      size.MaxDataRetentionPeriod,
				MaxConnectionAttemptsPerSec: int32(size.MaxConnectionAttemptsPerSec),
				QuotaConsumed:               int32(size.QuotaConsumed),
				QuotaType:                   size.QuotaType,
			})
		}
		instanceTypeList = append(instanceTypeList, api.SupportedKafkaInstanceType{
			Id:                  k,
			SupportedKafkaSizes: supportedSizesList,
		})
	}

	return instanceTypeList, nil
}
