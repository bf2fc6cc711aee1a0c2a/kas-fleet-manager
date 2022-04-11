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

func BuildKafkaRequest(modifyFn func(kafkaRequest *dbapi.KafkaRequest)) *dbapi.KafkaRequest {
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
	if modifyFn != nil {
		modifyFn(kafkaRequest)
	}
	return kafkaRequest
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
