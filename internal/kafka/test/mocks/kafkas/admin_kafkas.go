package mocks

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/admin/private"
)

func BuildAdminKafkaRequest(modifyFn func(kafka *private.Kafka)) *private.Kafka {
	kafka := &private.Kafka{
		ClusterId:     clusterID,
		Region:        kafkaRequestRegion,
		CloudProvider: kafkaRequestProvider,
		Status:        constants.KafkaRequestStatusReady.String(),
		MultiAz:       multiAz,
		Owner:         user,
		AccountNumber: "mock-ebs-0", // value relies on mock from here: pkg/services/account/account_mock.go
		Name:          kafkaRequestName,
	}
	if modifyFn != nil {
		modifyFn(kafka)
	}
	return kafka
}
