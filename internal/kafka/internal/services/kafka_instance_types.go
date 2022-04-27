package services

import (
	"fmt"
	"sort"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

//go:generate moq -out kafka_instance_types_moq.go . SupportedKafkaInstanceTypesService
type SupportedKafkaInstanceTypesService interface {
	GetSupportedKafkaInstanceTypesByRegion(providerId string, regionId string) ([]config.KafkaInstanceType, *errors.ServiceError)
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

func (t *supportedKafkaInstanceTypesService) GetSupportedKafkaInstanceTypesByRegion(providerId string, regionId string) ([]config.KafkaInstanceType, *errors.ServiceError) {
	instanceTypeList := []config.KafkaInstanceType{}
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

		instanceTypeList = append(instanceTypeList, config.KafkaInstanceType{
			Id:          k,
			DisplayName: instanceType.DisplayName,
			Sizes:       instanceType.Sizes,
		})
	}
	sort.Slice(instanceTypeList, func(i, j int) bool {
		return instanceTypeList[i].Id < instanceTypeList[j].Id
	})

	return instanceTypeList, nil
}
