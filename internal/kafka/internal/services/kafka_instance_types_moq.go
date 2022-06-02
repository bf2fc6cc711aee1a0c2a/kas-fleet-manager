// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package services

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"sync"
)

// Ensure, that SupportedKafkaInstanceTypesServiceMock does implement SupportedKafkaInstanceTypesService.
// If this is not the case, regenerate this file with moq.
var _ SupportedKafkaInstanceTypesService = &SupportedKafkaInstanceTypesServiceMock{}

// SupportedKafkaInstanceTypesServiceMock is a mock implementation of SupportedKafkaInstanceTypesService.
//
//     func TestSomethingThatUsesSupportedKafkaInstanceTypesService(t *testing.T) {
//
//         // make and configure a mocked SupportedKafkaInstanceTypesService
//         mockedSupportedKafkaInstanceTypesService := &SupportedKafkaInstanceTypesServiceMock{
//             GetSupportedKafkaInstanceTypesByRegionFunc: func(providerId string, regionId string) ([]config.KafkaInstanceType, *errors.ServiceError) {
// 	               panic("mock out the GetSupportedKafkaInstanceTypesByRegion method")
//             },
//         }
//
//         // use mockedSupportedKafkaInstanceTypesService in code that requires SupportedKafkaInstanceTypesService
//         // and then make assertions.
//
//     }
type SupportedKafkaInstanceTypesServiceMock struct {
	// GetSupportedKafkaInstanceTypesByRegionFunc mocks the GetSupportedKafkaInstanceTypesByRegion method.
	GetSupportedKafkaInstanceTypesByRegionFunc func(providerId string, regionId string) ([]config.KafkaInstanceType, *errors.ServiceError)

	// calls tracks calls to the methods.
	calls struct {
		// GetSupportedKafkaInstanceTypesByRegion holds details about calls to the GetSupportedKafkaInstanceTypesByRegion method.
		GetSupportedKafkaInstanceTypesByRegion []struct {
			// ProviderId is the providerId argument value.
			ProviderId string
			// RegionId is the regionId argument value.
			RegionId string
		}
	}
	lockGetSupportedKafkaInstanceTypesByRegion sync.RWMutex
}

// GetSupportedKafkaInstanceTypesByRegion calls GetSupportedKafkaInstanceTypesByRegionFunc.
func (mock *SupportedKafkaInstanceTypesServiceMock) GetSupportedKafkaInstanceTypesByRegion(providerId string, regionId string) ([]config.KafkaInstanceType, *errors.ServiceError) {
	if mock.GetSupportedKafkaInstanceTypesByRegionFunc == nil {
		panic("SupportedKafkaInstanceTypesServiceMock.GetSupportedKafkaInstanceTypesByRegionFunc: method is nil but SupportedKafkaInstanceTypesService.GetSupportedKafkaInstanceTypesByRegion was just called")
	}
	callInfo := struct {
		ProviderId string
		RegionId   string
	}{
		ProviderId: providerId,
		RegionId:   regionId,
	}
	mock.lockGetSupportedKafkaInstanceTypesByRegion.Lock()
	mock.calls.GetSupportedKafkaInstanceTypesByRegion = append(mock.calls.GetSupportedKafkaInstanceTypesByRegion, callInfo)
	mock.lockGetSupportedKafkaInstanceTypesByRegion.Unlock()
	return mock.GetSupportedKafkaInstanceTypesByRegionFunc(providerId, regionId)
}

// GetSupportedKafkaInstanceTypesByRegionCalls gets all the calls that were made to GetSupportedKafkaInstanceTypesByRegion.
// Check the length with:
//     len(mockedSupportedKafkaInstanceTypesService.GetSupportedKafkaInstanceTypesByRegionCalls())
func (mock *SupportedKafkaInstanceTypesServiceMock) GetSupportedKafkaInstanceTypesByRegionCalls() []struct {
	ProviderId string
	RegionId   string
} {
	var calls []struct {
		ProviderId string
		RegionId   string
	}
	mock.lockGetSupportedKafkaInstanceTypesByRegion.RLock()
	calls = mock.calls.GetSupportedKafkaInstanceTypesByRegion
	mock.lockGetSupportedKafkaInstanceTypesByRegion.RUnlock()
	return calls
}
