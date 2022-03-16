// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package services

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"sync"
)

// Ensure, that QuotaServiceMock does implement QuotaService.
// If this is not the case, regenerate this file with moq.
var _ QuotaService = &QuotaServiceMock{}

// QuotaServiceMock is a mock implementation of QuotaService.
//
//     func TestSomethingThatUsesQuotaService(t *testing.T) {
//
//         // make and configure a mocked QuotaService
//         mockedQuotaService := &QuotaServiceMock{
//             CheckIfQuotaIsDefinedForInstanceTypeFunc: func(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (bool, *errors.ServiceError) {
// 	               panic("mock out the CheckIfQuotaIsDefinedForInstanceType method")
//             },
//             DeleteQuotaFunc: func(subscriptionId string) *errors.ServiceError {
// 	               panic("mock out the DeleteQuota method")
//             },
//             ReserveQuotaFunc: func(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType, sizeRequired int, kafkaConfig *config.KafkaConfig) (string, *errors.ServiceError) {
// 	               panic("mock out the ReserveQuota method")
//             },
//         }
//
//         // use mockedQuotaService in code that requires QuotaService
//         // and then make assertions.
//
//     }
type QuotaServiceMock struct {
	// CheckIfQuotaIsDefinedForInstanceTypeFunc mocks the CheckIfQuotaIsDefinedForInstanceType method.
	CheckIfQuotaIsDefinedForInstanceTypeFunc func(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (bool, *errors.ServiceError)

	// DeleteQuotaFunc mocks the DeleteQuota method.
	DeleteQuotaFunc func(subscriptionId string) *errors.ServiceError

	// ReserveQuotaFunc mocks the ReserveQuota method.
	ReserveQuotaFunc func(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType, sizeRequired int, kafkaConfig *config.KafkaConfig) (string, *errors.ServiceError)

	// calls tracks calls to the methods.
	calls struct {
		// CheckIfQuotaIsDefinedForInstanceType holds details about calls to the CheckIfQuotaIsDefinedForInstanceType method.
		CheckIfQuotaIsDefinedForInstanceType []struct {
			// Kafka is the kafka argument value.
			Kafka *dbapi.KafkaRequest
			// InstanceType is the instanceType argument value.
			InstanceType types.KafkaInstanceType
		}
		// DeleteQuota holds details about calls to the DeleteQuota method.
		DeleteQuota []struct {
			// SubscriptionId is the subscriptionId argument value.
			SubscriptionId string
		}
		// ReserveQuota holds details about calls to the ReserveQuota method.
		ReserveQuota []struct {
			// Kafka is the kafka argument value.
			Kafka *dbapi.KafkaRequest
			// InstanceType is the instanceType argument value.
			InstanceType types.KafkaInstanceType
			// SizeRequired is the sizeRequired argument value.
			SizeRequired int
			// KafkaConfig is the kafkaConfig argument value.
			KafkaConfig *config.KafkaConfig
		}
	}
	lockCheckIfQuotaIsDefinedForInstanceType sync.RWMutex
	lockDeleteQuota                          sync.RWMutex
	lockReserveQuota                         sync.RWMutex
}

// CheckIfQuotaIsDefinedForInstanceType calls CheckIfQuotaIsDefinedForInstanceTypeFunc.
func (mock *QuotaServiceMock) CheckIfQuotaIsDefinedForInstanceType(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (bool, *errors.ServiceError) {
	if mock.CheckIfQuotaIsDefinedForInstanceTypeFunc == nil {
		panic("QuotaServiceMock.CheckIfQuotaIsDefinedForInstanceTypeFunc: method is nil but QuotaService.CheckIfQuotaIsDefinedForInstanceType was just called")
	}
	callInfo := struct {
		Kafka        *dbapi.KafkaRequest
		InstanceType types.KafkaInstanceType
	}{
		Kafka:        kafka,
		InstanceType: instanceType,
	}
	mock.lockCheckIfQuotaIsDefinedForInstanceType.Lock()
	mock.calls.CheckIfQuotaIsDefinedForInstanceType = append(mock.calls.CheckIfQuotaIsDefinedForInstanceType, callInfo)
	mock.lockCheckIfQuotaIsDefinedForInstanceType.Unlock()
	return mock.CheckIfQuotaIsDefinedForInstanceTypeFunc(kafka, instanceType)
}

// CheckIfQuotaIsDefinedForInstanceTypeCalls gets all the calls that were made to CheckIfQuotaIsDefinedForInstanceType.
// Check the length with:
//     len(mockedQuotaService.CheckIfQuotaIsDefinedForInstanceTypeCalls())
func (mock *QuotaServiceMock) CheckIfQuotaIsDefinedForInstanceTypeCalls() []struct {
	Kafka        *dbapi.KafkaRequest
	InstanceType types.KafkaInstanceType
} {
	var calls []struct {
		Kafka        *dbapi.KafkaRequest
		InstanceType types.KafkaInstanceType
	}
	mock.lockCheckIfQuotaIsDefinedForInstanceType.RLock()
	calls = mock.calls.CheckIfQuotaIsDefinedForInstanceType
	mock.lockCheckIfQuotaIsDefinedForInstanceType.RUnlock()
	return calls
}

// DeleteQuota calls DeleteQuotaFunc.
func (mock *QuotaServiceMock) DeleteQuota(subscriptionId string) *errors.ServiceError {
	if mock.DeleteQuotaFunc == nil {
		panic("QuotaServiceMock.DeleteQuotaFunc: method is nil but QuotaService.DeleteQuota was just called")
	}
	callInfo := struct {
		SubscriptionId string
	}{
		SubscriptionId: subscriptionId,
	}
	mock.lockDeleteQuota.Lock()
	mock.calls.DeleteQuota = append(mock.calls.DeleteQuota, callInfo)
	mock.lockDeleteQuota.Unlock()
	return mock.DeleteQuotaFunc(subscriptionId)
}

// DeleteQuotaCalls gets all the calls that were made to DeleteQuota.
// Check the length with:
//     len(mockedQuotaService.DeleteQuotaCalls())
func (mock *QuotaServiceMock) DeleteQuotaCalls() []struct {
	SubscriptionId string
} {
	var calls []struct {
		SubscriptionId string
	}
	mock.lockDeleteQuota.RLock()
	calls = mock.calls.DeleteQuota
	mock.lockDeleteQuota.RUnlock()
	return calls
}

// ReserveQuota calls ReserveQuotaFunc.
func (mock *QuotaServiceMock) ReserveQuota(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType, sizeRequired int, kafkaConfig *config.KafkaConfig) (string, *errors.ServiceError) {
	if mock.ReserveQuotaFunc == nil {
		panic("QuotaServiceMock.ReserveQuotaFunc: method is nil but QuotaService.ReserveQuota was just called")
	}
	callInfo := struct {
		Kafka        *dbapi.KafkaRequest
		InstanceType types.KafkaInstanceType
		SizeRequired int
		KafkaConfig  *config.KafkaConfig
	}{
		Kafka:        kafka,
		InstanceType: instanceType,
		SizeRequired: sizeRequired,
		KafkaConfig:  kafkaConfig,
	}
	mock.lockReserveQuota.Lock()
	mock.calls.ReserveQuota = append(mock.calls.ReserveQuota, callInfo)
	mock.lockReserveQuota.Unlock()
	return mock.ReserveQuotaFunc(kafka, instanceType, sizeRequired, kafkaConfig)
}

// ReserveQuotaCalls gets all the calls that were made to ReserveQuota.
// Check the length with:
//     len(mockedQuotaService.ReserveQuotaCalls())
func (mock *QuotaServiceMock) ReserveQuotaCalls() []struct {
	Kafka        *dbapi.KafkaRequest
	InstanceType types.KafkaInstanceType
	SizeRequired int
	KafkaConfig  *config.KafkaConfig
} {
	var calls []struct {
		Kafka        *dbapi.KafkaRequest
		InstanceType types.KafkaInstanceType
		SizeRequired int
		KafkaConfig  *config.KafkaConfig
	}
	mock.lockReserveQuota.RLock()
	calls = mock.calls.ReserveQuota
	mock.lockReserveQuota.RUnlock()
	return calls
}
