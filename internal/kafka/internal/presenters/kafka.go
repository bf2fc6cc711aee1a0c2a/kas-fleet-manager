package presenters

import (
	"fmt"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
)

// ConvertKafkaRequest from payload to KafkaRequest
func ConvertKafkaRequest(kafkaRequestPayload public.KafkaRequestPayload, dbKafkarequest ...*dbapi.KafkaRequest) *dbapi.KafkaRequest {
	var kafka *dbapi.KafkaRequest
	if len(dbKafkarequest) == 0 {
		kafka = &dbapi.KafkaRequest{}
	} else {
		kafka = dbKafkarequest[0]
	}

	kafka.Region = kafkaRequestPayload.Region
	kafka.Name = kafkaRequestPayload.Name
	kafka.CloudProvider = kafkaRequestPayload.CloudProvider
	kafka.MultiAZ = kafkaRequestPayload.MultiAz

	if kafkaRequestPayload.ReauthenticationEnabled != nil {
		kafka.ReauthenticationEnabled = *kafkaRequestPayload.ReauthenticationEnabled
	} else {
		kafka.ReauthenticationEnabled = true // true by default
	}

	return kafka
}

// PresentKafkaRequest - create KafkaRequest in an appropriate format ready to be returned by the API
func PresentKafkaRequest(kafkaRequest *dbapi.KafkaRequest, config *config.KafkaConfig) (public.KafkaRequest, *errors.ServiceError) {
	reference := PresentReference(kafkaRequest.ID, kafkaRequest)

	var ingressThroughputPerSec, egressThroughputPerSec, maxDataRetentionPeriod string
	var totalMaxConnections, maxPartitions, maxConnectionAttemptsPerSec int
	if config != nil {
		kafkaConfig, err := config.GetKafkaInstanceSize(kafkaRequest.InstanceType, kafkaRequest.SizeId)
		if err != nil {
			logger.Logger.Error(err)
		} else {
			ingressThroughputPerSec = kafkaConfig.IngressThroughputPerSec
			egressThroughputPerSec = kafkaConfig.EgressThroughputPerSec
			totalMaxConnections = kafkaConfig.TotalMaxConnections
			maxPartitions = kafkaConfig.MaxPartitions
			maxDataRetentionPeriod = kafkaConfig.MaxDataRetentionPeriod
			maxConnectionAttemptsPerSec = kafkaConfig.MaxConnectionAttemptsPerSec
		}
	}

	displayName, err := getDisplayName(kafkaRequest.InstanceType, config)

	if err != nil {
		return public.KafkaRequest{}, err
	}

	return public.KafkaRequest{
		Id:                          reference.Id,
		Kind:                        reference.Kind,
		Href:                        reference.Href,
		Region:                      kafkaRequest.Region,
		Name:                        kafkaRequest.Name,
		CloudProvider:               kafkaRequest.CloudProvider,
		MultiAz:                     kafkaRequest.MultiAZ,
		Owner:                       kafkaRequest.Owner,
		BootstrapServerHost:         setBootstrapServerHost(kafkaRequest.BootstrapServerHost),
		Status:                      kafkaRequest.Status,
		CreatedAt:                   kafkaRequest.CreatedAt,
		UpdatedAt:                   kafkaRequest.UpdatedAt,
		FailedReason:                kafkaRequest.FailedReason,
		Version:                     kafkaRequest.ActualKafkaVersion,
		InstanceType:                kafkaRequest.InstanceType,
		ReauthenticationEnabled:     kafkaRequest.ReauthenticationEnabled,
		KafkaStorageSize:            kafkaRequest.KafkaStorageSize,
		BrowserUrl:                  fmt.Sprintf("%s/%s/dashboard", strings.TrimSuffix(config.BrowserUrl, "/"), reference.Id),
		SizeId:                      kafkaRequest.SizeId,
		InstanceTypeName:            displayName,
		IngressThroughputPerSec:     ingressThroughputPerSec,
		EgressThroughputPerSec:      egressThroughputPerSec,
		TotalMaxConnections:         totalMaxConnections,
		MaxPartitions:               maxPartitions,
		MaxDataRetentionPeriod:      maxDataRetentionPeriod,
		MaxConnectionAttemptsPerSec: maxConnectionAttemptsPerSec,
	}, nil
}

func setBootstrapServerHost(bootstrapServerHost string) string {
	if bootstrapServerHost != "" {
		return fmt.Sprintf("%s:443", bootstrapServerHost)
	}
	return bootstrapServerHost
}

func getDisplayName(instanceType string, config *config.KafkaConfig) (string, *errors.ServiceError) {
	if config != nil && strings.Trim(instanceType, " ") != "" {
		kafkaInstanceType, err := config.SupportedInstanceTypes.Configuration.GetKafkaInstanceTypeByID(instanceType)
		if err != nil {
			return "", errors.NewWithCause(errors.ErrorGeneral, err, "Unable to get kafka display name for '%s' instance type", instanceType)
		}
		return kafkaInstanceType.DisplayName, nil
	}
	return "", nil
}
