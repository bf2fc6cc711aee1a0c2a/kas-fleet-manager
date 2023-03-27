package presenters

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
)

func PresentEnterpriseClusterAddonParameters(cluster api.Cluster, fleetShardParams services.ParameterList) (public.EnterpriseClusterAddonParameters, *errors.ServiceError) {
	fsoParams := []public.FleetshardParameter{}

	for _, param := range fleetShardParams {
		fsoParams = append(fsoParams, public.FleetshardParameter{
			Id:    param.Id,
			Value: param.Value,
		})
	}

	reference := PresentReference(cluster.ClusterID, public.EnterpriseClusterAddonParameters{})
	c := public.EnterpriseClusterAddonParameters{
		Kind:                 reference.Kind,
		Id:                   reference.Id,
		Href:                 reference.Href,
		FleetshardParameters: fsoParams,
	}

	return c, nil
}

func PresentEnterpriseClusterRegistrationResponse(cluster api.Cluster, consumedStreamingUnitsInTheCluster int32, kafkaConfig *config.KafkaConfig, fleetShardParams services.ParameterList) (public.EnterpriseClusterRegistrationResponse, *errors.ServiceError) {
	fsoParams := []public.FleetshardParameter{}

	for _, param := range fleetShardParams {
		fsoParams = append(fsoParams, public.FleetshardParameter{
			Id:    param.Id,
			Value: param.Value,
		})
	}

	enterpriseClusterRepresentation, err := PresentEnterpriseCluster(cluster, consumedStreamingUnitsInTheCluster, kafkaConfig)
	if err != nil {
		return public.EnterpriseClusterRegistrationResponse{}, nil
	}

	reference := PresentReference(cluster.ClusterID, cluster)
	c := public.EnterpriseClusterRegistrationResponse{
		Kind:                          reference.Kind,
		Id:                            reference.Id,
		Href:                          reference.Href,
		ClusterId:                     enterpriseClusterRepresentation.ClusterId,
		Status:                        enterpriseClusterRepresentation.Status,
		CloudProvider:                 enterpriseClusterRepresentation.CloudProvider,
		Region:                        enterpriseClusterRepresentation.Region,
		MultiAz:                       enterpriseClusterRepresentation.MultiAz,
		AccessKafkasViaPrivateNetwork: enterpriseClusterRepresentation.AccessKafkasViaPrivateNetwork,
		SupportedInstanceTypes:        enterpriseClusterRepresentation.SupportedInstanceTypes,
		FleetshardParameters:          fsoParams,
		CapacityInformation:           enterpriseClusterRepresentation.CapacityInformation,
	}

	return c, nil
}

func PresentEnterpriseCluster(cluster api.Cluster, consumedStreamingUnitsInTheCluster int32, kafkaConfig *config.KafkaConfig) (public.EnterpriseCluster, error) {
	reference := PresentReference(cluster.ClusterID, cluster)
	presentedCluster := public.EnterpriseCluster{
		Id:                            cluster.ClusterID,
		Status:                        cluster.Status.String(),
		ClusterId:                     cluster.ClusterID,
		Kind:                          reference.Kind,
		Href:                          reference.Href,
		CloudProvider:                 cluster.CloudProvider,
		Region:                        cluster.Region,
		MultiAz:                       cluster.MultiAZ,
		AccessKafkasViaPrivateNetwork: cluster.AccessKafkasViaPrivateNetwork,
		SupportedInstanceTypes:        public.SupportedKafkaInstanceTypesList{},
	}

	// enterprise clusters only supports standard instance type for now. It is safe to hardcode this.
	storedCapacityInfo, ok := cluster.RetrieveDynamicCapacityInfo()[types.STANDARD.String()]
	if ok {
		presentedCluster.CapacityInformation = presentEnterpriseClusterCapacityInfo(consumedStreamingUnitsInTheCluster, storedCapacityInfo)
		supportedInstanceTypes, err := presentEnterpriseClusterSupportedInstanceTypes(kafkaConfig)
		if err != nil {
			return public.EnterpriseCluster{}, err
		}

		presentedCluster.SupportedInstanceTypes = supportedInstanceTypes
	} else { // this should never happen, let's log an error in case it happens
		err := fmt.Errorf("cluster with cluster_id %q is missing capacity information", cluster.ClusterID)
		logger.Logger.Error(err)
		return public.EnterpriseCluster{}, err
	}

	return presentedCluster, nil
}

func presentEnterpriseClusterCapacityInfo(consumedStreamingUnitsInTheCluster int32, storedCapacityInfo api.DynamicCapacityInfo) public.EnterpriseClusterAllOfCapacityInformation {
	return public.EnterpriseClusterAllOfCapacityInformation{
		ConsumedKafkaStreamingUnits:  consumedStreamingUnitsInTheCluster,
		KafkaMachinePoolNodeCount:    storedCapacityInfo.MaxNodes,
		RemainingKafkaStreamingUnits: storedCapacityInfo.MaxUnits - consumedStreamingUnitsInTheCluster,
		MaximumKafkaStreamingUnits:   storedCapacityInfo.MaxUnits,
	}
}

func presentEnterpriseClusterSupportedInstanceTypes(kafkaConfig *config.KafkaConfig) (public.SupportedKafkaInstanceTypesList, error) {
	// enterprise clusters only supports standard instance type for now. It is safe to hardcode this.
	standardInstanceType, err := kafkaConfig.SupportedInstanceTypes.Configuration.GetKafkaInstanceTypeByID(types.STANDARD.String())
	if err != nil { // this should never happen, lets log an error in case it happens.
		logger.Logger.Errorf("failed to find standard instance type from supported instance type config due to %q.", err.Error())
		return public.SupportedKafkaInstanceTypesList{
			InstanceTypes: []public.SupportedKafkaInstanceType{},
		}, err
	}

	// only enlist enterprise billing model as the supported billing model
	enterpriseBillingModel, err := standardInstanceType.GetKafkaSupportedBillingModelByID(constants.BillingModelEnterprise.String())
	if err != nil { // this should never happen, lets log an error in case it happens.
		logger.Logger.Errorf("failed to find enterprise billing model for standard instance due to %q.", err.Error())
		return public.SupportedKafkaInstanceTypesList{
			InstanceTypes: []public.SupportedKafkaInstanceType{},
		}, err
	}

	presentedSizes := GetSupportedSizes(&config.KafkaInstanceType{Sizes: standardInstanceType.Sizes})

	return public.SupportedKafkaInstanceTypesList{
		InstanceTypes: []public.SupportedKafkaInstanceType{
			{
				Id:          standardInstanceType.Id,
				DisplayName: standardInstanceType.DisplayName,
				Sizes:       presentedSizes,
				SupportedBillingModels: GetSupportedBillingModels(&config.KafkaInstanceType{
					SupportedBillingModels: []config.KafkaBillingModel{*enterpriseBillingModel},
				}),
			},
		},
	}, nil
}
