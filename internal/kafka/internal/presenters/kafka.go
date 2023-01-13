package presenters

import (
	"fmt"
	"strings"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"

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

	kafka.BillingCloudAccountId = shared.SafeString(kafkaRequestPayload.BillingCloudAccountId)
	kafka.Marketplace = shared.SafeString(kafkaRequestPayload.Marketplace)
	kafka.DesiredKafkaBillingModel = shared.SafeString(kafkaRequestPayload.BillingModel)

	if kafkaRequestPayload.ReauthenticationEnabled != nil {
		kafka.ReauthenticationEnabled = *kafkaRequestPayload.ReauthenticationEnabled
	} else {
		kafka.ReauthenticationEnabled = true // true by default
	}

	return kafka
}

// PresentKafkaRequest - create KafkaRequest in an appropriate format ready to be returned by the API
func PresentKafkaRequest(kafkaRequest *dbapi.KafkaRequest, kafkaConfig *config.KafkaConfig) (public.KafkaRequest, *errors.ServiceError) {
	reference := PresentReference(kafkaRequest.ID, kafkaRequest)

	var ingressThroughputPerSec, egressThroughputPerSec, maxDataRetentionPeriod string
	var totalMaxConnections, maxPartitions, maxConnectionAttemptsPerSec int
	if kafkaConfig != nil {
		instanceSize, err := kafkaConfig.GetKafkaInstanceSize(kafkaRequest.InstanceType, kafkaRequest.SizeId)
		if err != nil {
			logger.Logger.Error(err)
		} else {
			ingressThroughputPerSec = instanceSize.IngressThroughputPerSec.String()
			egressThroughputPerSec = instanceSize.EgressThroughputPerSec.String()
			totalMaxConnections = instanceSize.TotalMaxConnections
			maxPartitions = instanceSize.MaxPartitions
			maxDataRetentionPeriod = instanceSize.MaxDataRetentionPeriod
			maxConnectionAttemptsPerSec = instanceSize.MaxConnectionAttemptsPerSec
		}
	}

	var expiresAt *time.Time
	if kafkaRequest.ExpiresAt.Valid {
		expiresAt = &kafkaRequest.ExpiresAt.Time
	}

	displayName, err := getDisplayName(kafkaRequest.InstanceType, kafkaConfig)
	if err != nil {
		return public.KafkaRequest{}, err
	}

	// convert kafka storage size to bytes
	maxDataRetentionSizeQuantity := config.Quantity(kafkaRequest.KafkaStorageSize)
	maxDataRetentionSizeBytes, conversionErr := maxDataRetentionSizeQuantity.ToInt64()
	if conversionErr != nil {
		return public.KafkaRequest{}, errors.NewWithCause(errors.ErrorGeneral, conversionErr, "failed to get bytes value for max_data_retention_size")
	}

	return public.KafkaRequest{
		Id:                         reference.Id,
		Kind:                       reference.Kind,
		Href:                       reference.Href,
		Region:                     kafkaRequest.Region,
		Name:                       kafkaRequest.Name,
		CloudProvider:              kafkaRequest.CloudProvider,
		MultiAz:                    kafkaRequest.MultiAZ,
		Owner:                      kafkaRequest.Owner,
		BootstrapServerHost:        setBootstrapServerHost(kafkaRequest.BootstrapServerHost),
		AdminApiServerUrl:          kafkaRequest.AdminApiServerURL,
		Status:                     kafkaRequest.Status,
		CreatedAt:                  kafkaRequest.CreatedAt,
		UpdatedAt:                  kafkaRequest.UpdatedAt,
		ExpiresAt:                  expiresAt,
		FailedReason:               kafkaRequest.FailedReason,
		Version:                    kafkaRequest.ActualKafkaVersion,
		InstanceType:               kafkaRequest.InstanceType,
		ReauthenticationEnabled:    kafkaRequest.ReauthenticationEnabled,
		DeprecatedKafkaStorageSize: kafkaRequest.KafkaStorageSize,
		MaxDataRetentionSize: public.SupportedKafkaSizeBytesValueItem{
			Bytes: maxDataRetentionSizeBytes,
		},
		BrowserUrl:                            fmt.Sprintf("%s/%s/dashboard", strings.TrimSuffix(kafkaConfig.BrowserUrl, "/"), reference.Id),
		SizeId:                                kafkaRequest.SizeId,
		DeprecatedInstanceTypeName:            displayName,
		DeprecatedIngressThroughputPerSec:     ingressThroughputPerSec,
		DeprecatedEgressThroughputPerSec:      egressThroughputPerSec,
		DeprecatedTotalMaxConnections:         int32(totalMaxConnections),
		DeprecatedMaxPartitions:               int32(maxPartitions),
		DeprecatedMaxDataRetentionPeriod:      maxDataRetentionPeriod,
		DeprecatedMaxConnectionAttemptsPerSec: int32(maxConnectionAttemptsPerSec),
		BillingCloudAccountId:                 kafkaRequest.BillingCloudAccountId,
		Marketplace:                           kafkaRequest.Marketplace,
		BillingModel:                          kafkaRequest.ActualKafkaBillingModel,
		PromotionStatus:                       kafkaRequest.PromotionStatus.String(),
		PromotionDetails:                      kafkaRequest.PromotionDetails,
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
			return "", errors.NewWithCause(errors.ErrorGeneral, err, "unable to get kafka display name for '%s' instance type", instanceType)
		}
		return kafkaInstanceType.DisplayName, nil
	}
	return "", nil
}
