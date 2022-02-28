package mocks

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
)

var (
	ingress                      = "2Mi"
	egress                       = "3Mi"
	maxConnections         int32 = 1000
	maxDataRetention             = "1000Gi"
	maxPartitions          int32 = 1000
	maxDataRetentionPeriod       = "P14D"
	maxConnectionAttempts  int32 = 50
	conditionsType               = "Ready"
	conditionsStatus             = "False"
	conditionsReason             = "Installing"
	kafkaVersion                 = "2.8.0"
	strimziVersion               = "2.8.0"
	ibpVersion                   = "2.8"
	routeName                    = "bootstrap"
	prefix                       = ""
	router                       = "elb.test1.kafka.example.com"
)

func BuildPrivateDataPlaneKafkaStatus(modifyFn func(status map[string]private.DataPlaneKafkaStatus)) map[string]private.DataPlaneKafkaStatus {
	status := map[string]private.DataPlaneKafkaStatus{}
	status[constants.KafkaRequestStatusReady.String()] = private.DataPlaneKafkaStatus{
		Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{
			{
				Type:   conditionsType,
				Status: conditionsStatus,
				Reason: conditionsReason,
			},
		},
		Capacity: private.DataPlaneKafkaStatusCapacity{
			IngressThroughputPerSec:     &ingress,
			EgressThroughputPerSec:      &egress,
			TotalMaxConnections:         &maxConnections,
			MaxDataRetentionSize:        &maxDataRetention,
			MaxPartitions:               &maxPartitions,
			MaxDataRetentionPeriod:      &maxDataRetentionPeriod,
			MaxConnectionAttemptsPerSec: &maxConnectionAttempts,
		},
		Versions: private.DataPlaneKafkaStatusVersions{
			Kafka:    kafkaVersion,
			Strimzi:  strimziVersion,
			KafkaIbp: ibpVersion,
		},
		Routes: &[]private.DataPlaneKafkaStatusRoutes{
			{
				Name:   routeName,
				Prefix: prefix,
				Router: router,
			},
		},
	}
	if modifyFn != nil {
		modifyFn(status)
	}
	return status
}

func BuildDbDataPlaneKafkaStatus(modifyFn func(status *dbapi.DataPlaneKafkaStatus)) *dbapi.DataPlaneKafkaStatus {
	status := &dbapi.DataPlaneKafkaStatus{
		KafkaClusterId: constants.KafkaRequestStatusReady.String(),
		Conditions: []dbapi.DataPlaneKafkaStatusCondition{
			{
				Type:   conditionsType,
				Status: conditionsStatus,
				Reason: conditionsReason,
			},
		},
		Routes: []dbapi.DataPlaneKafkaRouteRequest{
			{
				Name:   routeName,
				Prefix: prefix,
				Router: router,
			},
		},
		KafkaVersion:    kafkaVersion,
		StrimziVersion:  strimziVersion,
		KafkaIBPVersion: ibpVersion,
	}
	if modifyFn != nil {
		modifyFn(status)
	}
	return status
}
