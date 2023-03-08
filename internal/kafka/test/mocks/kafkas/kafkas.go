package mocks

import (
	"database/sql"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

	mocksupportedinstancetypes "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/supported_instance_types"

	"gorm.io/gorm"
)

const (
	DefaultKafkaRequestRegion               = "us-east-1"
	DefaultKafkaRequestProvider             = "aws"
	DefaultKafkaRequestName                 = "test-cluster"
	DefaultClusterID                        = "test-cluster-id"
	user                                    = "test-user"
	DefaultMultiAz                          = true
	DefaultBootstrapServerHost              = "test-bootstrap-server-host"
	DefaultPlacementId                      = "test-placement-id"
	DefaultOrganisationId                   = "13640203"
	DefaultKafkaID                          = "test-kafka-id"
	defaultCanaryServiceAccountClientID     = "canary-service-account-client-id"
	defaultCanaryServiceAccountClientSecret = "canary-service-acccount-client-secret"
)

var (
	DefaultInstanceType = types.STANDARD.String()
)

type KafkaRequestAttribute int

const (
	STATUS KafkaRequestAttribute = iota
	OWNER
	CLUSTER_ID
	BOOTSTRAP_SERVER_HOST
	FAILED_REASON
	ACTUAL_KAFKA_VERSION
	DESIRED_KAFKA_VERSION
	INSTANCE_TYPE
	STORAGE_SIZE
	SIZE_ID
	NAME
	NAMESPACE
	REGION
	CLOUD_PROVIDER
	ACTUAL_STRIMZI_VERSION
	DESIRED_STRIMZI_VERSION
	ACTUAL_KAFKA_IBP_VERSION
	DESIRED_KAFKA_IBP_VERSION
	ORGANISATION_ID
	ID
	DESIRED_KAFKA_BILLING_MODEL
	ACTUAL_KAFKA_BILLING_MODEL
	PROMOTION_STATUS
	PROMOTION_DETAILS
	CANARY_SERVICE_ACCOUNT_CLIENT_ID
	CANARY_SERVICE_ACCOUNT_CLIENT_SECRET
)

type KafkaAttribute int

type KafkaRequestBuildOption func(*dbapi.KafkaRequest)

func With(attribute KafkaRequestAttribute, value string) KafkaRequestBuildOption {
	return func(request *dbapi.KafkaRequest) {
		switch attribute {
		case STATUS:
			request.Status = value
		case OWNER:
			request.Owner = value
		case CLUSTER_ID:
			request.ClusterID = value
		case BOOTSTRAP_SERVER_HOST:
			request.BootstrapServerHost = value
		case FAILED_REASON:
			request.FailedReason = value
		case ACTUAL_KAFKA_VERSION:
			request.ActualKafkaVersion = value
		case DESIRED_KAFKA_VERSION:
			request.DesiredKafkaVersion = value
		case INSTANCE_TYPE:
			request.InstanceType = value
		case STORAGE_SIZE:
			request.MaxDataRetentionSize = value
		case SIZE_ID:
			request.SizeId = value
		case NAME:
			request.Name = value
		case NAMESPACE:
			request.Namespace = value
		case REGION:
			request.Region = value
		case CLOUD_PROVIDER:
			request.CloudProvider = value
		case ACTUAL_STRIMZI_VERSION:
			request.ActualStrimziVersion = value
		case DESIRED_STRIMZI_VERSION:
			request.DesiredStrimziVersion = value
		case ACTUAL_KAFKA_IBP_VERSION:
			request.ActualKafkaIBPVersion = value
		case DESIRED_KAFKA_IBP_VERSION:
			request.DesiredKafkaIBPVersion = value
		case ORGANISATION_ID:
			request.OrganisationId = value
		case ID:
			request.Meta.ID = value
		case DESIRED_KAFKA_BILLING_MODEL:
			request.DesiredKafkaBillingModel = value
		case ACTUAL_KAFKA_BILLING_MODEL:
			request.ActualKafkaBillingModel = value
		case PROMOTION_STATUS:
			request.PromotionStatus = dbapi.KafkaPromotionStatus(value)
		case PROMOTION_DETAILS:
			request.PromotionDetails = value
		case CANARY_SERVICE_ACCOUNT_CLIENT_ID:
			request.CanaryServiceAccountClientID = value
		case CANARY_SERVICE_ACCOUNT_CLIENT_SECRET:
			request.CanaryServiceAccountClientSecret = value
		}
	}
}

func WithRoutes(routes api.JSON) KafkaRequestBuildOption {
	return func(request *dbapi.KafkaRequest) {
		request.Routes = routes
	}
}

func WithReauthenticationEnabled(enabled bool) KafkaRequestBuildOption {
	return func(request *dbapi.KafkaRequest) {
		request.ReauthenticationEnabled = enabled
	}
}

func WithDeleted(deleted bool) KafkaRequestBuildOption {
	return func(request *dbapi.KafkaRequest) {
		request.Meta.DeletedAt.Valid = deleted
	}
}

func WithMultiAZ(multiaz bool) KafkaRequestBuildOption {
	return func(request *dbapi.KafkaRequest) {
		request.MultiAZ = multiaz
	}
}

func WithCreatedAt(createdAt time.Time) KafkaRequestBuildOption {
	return func(request *dbapi.KafkaRequest) {
		request.Meta.CreatedAt = createdAt
	}
}

func WithExpiresAt(expiresAt sql.NullTime) KafkaRequestBuildOption {
	return func(request *dbapi.KafkaRequest) {
		request.ExpiresAt = expiresAt
	}
}

func WithPredefinedTestValues() KafkaRequestBuildOption {
	return func(request *dbapi.KafkaRequest) {
		request.Meta = api.Meta{
			DeletedAt: gorm.DeletedAt{Valid: false},
		}
		request.Name = DefaultKafkaRequestName
		request.Namespace = DefaultKafkaRequestName
		request.Region = DefaultKafkaRequestRegion
		request.CloudProvider = DefaultKafkaRequestProvider
		request.MultiAZ = DefaultMultiAz
		request.InstanceType = DefaultInstanceType
		request.SizeId = mocksupportedinstancetypes.DefaultKafkaInstanceSizeId
		request.ClusterID = DefaultClusterID
		request.BootstrapServerHost = DefaultBootstrapServerHost
		request.Owner = user
		request.Status = constants.KafkaRequestStatusReady.String()
		request.MaxDataRetentionSize = mocksupportedinstancetypes.DefaultMaxDataRetentionSize
		request.OrganisationId = DefaultOrganisationId
		request.CanaryServiceAccountClientID = defaultCanaryServiceAccountClientID
		request.CanaryServiceAccountClientSecret = defaultCanaryServiceAccountClientSecret
	}
}

func BuildKafkaRequest(options ...KafkaRequestBuildOption) *dbapi.KafkaRequest {
	kafkaRequest := &dbapi.KafkaRequest{}
	for _, option := range options {
		option(kafkaRequest)
	}
	return kafkaRequest
}

func BuildKafkaRequestMap(modifyFn func(m []map[string]interface{})) []map[string]interface{} {
	m := []map[string]interface{}{
		{
			"region":                DefaultKafkaRequestRegion,
			"cloud_provider":        DefaultKafkaRequestProvider,
			"multi_az":              DefaultMultiAz,
			"name":                  DefaultKafkaRequestName,
			"status":                constants.KafkaRequestStatusReady.String(),
			"owner":                 user,
			"cluster_id":            DefaultClusterID,
			"id":                    "",
			"bootstrap_server_host": "",
			"created_at":            time.Time{},
			"deleted_at":            time.Time{},
			"updated_at":            time.Time{},
		},
	}
	if modifyFn != nil {
		modifyFn(m)
	}
	return m
}

func GetSampleKafkaAllRoutes() []private.KafkaAllOfRoutes {
	return []private.KafkaAllOfRoutes{{Domain: "test.example.com", Router: "test.example.com"}}
}

func GetMalformedRoutes() []byte {
	return []byte("{\"domain\": \"test.example.com\",\"router\": \"test.example.com\"}]")
}

func GetRoutes() []byte {
	return []byte("[{\"domain\": \"test.example.com\",\"router\": \"test.example.com\"}]")
}

func BuildKafkaRequestPayload(modifyFn func(payload *public.KafkaRequestPayload)) *public.KafkaRequestPayload {
	payload := &public.KafkaRequestPayload{
		CloudProvider: DefaultKafkaRequestProvider,
		Name:          DefaultKafkaRequestName,
		Region:        DefaultKafkaRequestRegion,
	}
	if modifyFn != nil {
		modifyFn(payload)
	}
	return payload
}

func BuildPublicKafkaRequest(modifyFn func(kafka *public.KafkaRequest)) *public.KafkaRequest {
	kafka := &public.KafkaRequest{
		Region:        DefaultKafkaRequestRegion,
		CloudProvider: DefaultKafkaRequestProvider,
		Name:          DefaultKafkaRequestName,
		MultiAz:       DefaultMultiAz,
		Status:        constants.KafkaRequestStatusReady.String(),
		Owner:         user,
		CreatedAt:     time.Time{},
		UpdatedAt:     time.Time{},
		Version:       kafkaVersion,
	}
	if modifyFn != nil {
		modifyFn(kafka)
	}
	return kafka
}
