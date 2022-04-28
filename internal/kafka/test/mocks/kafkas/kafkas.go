package mocks

import (
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"gorm.io/gorm"
)

const (
	kafkaRequestRegion   = "us-east-1"
	kafkaRequestProvider = "aws"
	kafkaRequestName     = "test-cluster"
	clusterID            = "test-cluster-id"
	user                 = "test-user"
	multiAz              = true
)

type KafkaRequestAttribute int

const (
	STATUS KafkaRequestAttribute = iota
	OWNER
	CLUSTER_ID
	BOOTSTRAP_SERVER_HOST
	FAILED_REASON
	ACTUAL_KAFKA_VERSION
	INSTANCE_TYPE
	STORAGE_SIZE
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
		case INSTANCE_TYPE:
			request.InstanceType = value
		case STORAGE_SIZE:
			request.KafkaStorageSize = value
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

func BuildKafkaRequest(options ...KafkaRequestBuildOption) *dbapi.KafkaRequest {
	kafkaRequest := &dbapi.KafkaRequest{
		Meta: api.Meta{
			DeletedAt: gorm.DeletedAt{Valid: true},
		},
		Region:        kafkaRequestRegion,
		ClusterID:     clusterID,
		CloudProvider: kafkaRequestProvider,
		Name:          kafkaRequestName,
		MultiAZ:       multiAz,
		Status:        constants.KafkaRequestStatusReady.String(),
		Owner:         user,
	}
	for _, option := range options {
		option(kafkaRequest)
	}
	return kafkaRequest
}

func BuildKafkaRequestMap(modifyFn func(m []map[string]interface{})) []map[string]interface{} {
	m := []map[string]interface{}{
		{
			"region":                kafkaRequestRegion,
			"cloud_provider":        kafkaRequestProvider,
			"multi_az":              multiAz,
			"name":                  kafkaRequestName,
			"status":                constants.KafkaRequestStatusReady.String(),
			"owner":                 user,
			"cluster_id":            clusterID,
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
		CloudProvider: kafkaRequestProvider,
		MultiAz:       multiAz,
		Name:          kafkaRequestName,
		Region:        kafkaRequestRegion,
	}
	if modifyFn != nil {
		modifyFn(payload)
	}
	return payload
}

func BuildPublicKafkaRequest(modifyFn func(kafka *public.KafkaRequest)) *public.KafkaRequest {
	kafka := &public.KafkaRequest{
		Region:        kafkaRequestRegion,
		CloudProvider: kafkaRequestProvider,
		Name:          kafkaRequestName,
		MultiAz:       multiAz,
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
