// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package services

import (
	"github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"sync"
)

// Ensure, that SyncsetServiceMock does implement SyncsetService.
// If this is not the case, regenerate this file with moq.
var _ SyncsetService = &SyncsetServiceMock{}

// SyncsetServiceMock is a mock implementation of SyncsetService.
//
//     func TestSomethingThatUsesSyncsetService(t *testing.T) {
//
//         // make and configure a mocked SyncsetService
//         mockedSyncsetService := &SyncsetServiceMock{
//             CreateFunc: func(syncsetBuilder *v1.SyncsetBuilder, syncsetId string, clusterId string) (*v1.Syncset, *errors.ServiceError) {
// 	               panic("mock out the Create method")
//             },
//             DeleteFunc: func(syncsetId string, clusterId string) *errors.ServiceError {
// 	               panic("mock out the Delete method")
//             },
//         }
//
//         // use mockedSyncsetService in code that requires SyncsetService
//         // and then make assertions.
//
//     }
type SyncsetServiceMock struct {
	// CreateFunc mocks the Create method.
	CreateFunc func(syncsetBuilder *v1.SyncsetBuilder, syncsetId string, clusterId string) (*v1.Syncset, *errors.ServiceError)

	// DeleteFunc mocks the Delete method.
	DeleteFunc func(syncsetId string, clusterId string) *errors.ServiceError

	// calls tracks calls to the methods.
	calls struct {
		// Create holds details about calls to the Create method.
		Create []struct {
			// SyncsetBuilder is the syncsetBuilder argument value.
			SyncsetBuilder *v1.SyncsetBuilder
			// SyncsetId is the syncsetId argument value.
			SyncsetId string
			// ClusterId is the clusterId argument value.
			ClusterId string
		}
		// Delete holds details about calls to the Delete method.
		Delete []struct {
			// SyncsetId is the syncsetId argument value.
			SyncsetId string
			// ClusterId is the clusterId argument value.
			ClusterId string
		}
	}
	lockCreate sync.RWMutex
	lockDelete sync.RWMutex
}

// Create calls CreateFunc.
func (mock *SyncsetServiceMock) Create(syncsetBuilder *v1.SyncsetBuilder, syncsetId string, clusterId string) (*v1.Syncset, *errors.ServiceError) {
	if mock.CreateFunc == nil {
		panic("SyncsetServiceMock.CreateFunc: method is nil but SyncsetService.Create was just called")
	}
	callInfo := struct {
		SyncsetBuilder *v1.SyncsetBuilder
		SyncsetId      string
		ClusterId      string
	}{
		SyncsetBuilder: syncsetBuilder,
		SyncsetId:      syncsetId,
		ClusterId:      clusterId,
	}
	mock.lockCreate.Lock()
	mock.calls.Create = append(mock.calls.Create, callInfo)
	mock.lockCreate.Unlock()
	return mock.CreateFunc(syncsetBuilder, syncsetId, clusterId)
}

// CreateCalls gets all the calls that were made to Create.
// Check the length with:
//     len(mockedSyncsetService.CreateCalls())
func (mock *SyncsetServiceMock) CreateCalls() []struct {
	SyncsetBuilder *v1.SyncsetBuilder
	SyncsetId      string
	ClusterId      string
} {
	var calls []struct {
		SyncsetBuilder *v1.SyncsetBuilder
		SyncsetId      string
		ClusterId      string
	}
	mock.lockCreate.RLock()
	calls = mock.calls.Create
	mock.lockCreate.RUnlock()
	return calls
}

// Delete calls DeleteFunc.
func (mock *SyncsetServiceMock) Delete(syncsetId string, clusterId string) *errors.ServiceError {
	if mock.DeleteFunc == nil {
		panic("SyncsetServiceMock.DeleteFunc: method is nil but SyncsetService.Delete was just called")
	}
	callInfo := struct {
		SyncsetId string
		ClusterId string
	}{
		SyncsetId: syncsetId,
		ClusterId: clusterId,
	}
	mock.lockDelete.Lock()
	mock.calls.Delete = append(mock.calls.Delete, callInfo)
	mock.lockDelete.Unlock()
	return mock.DeleteFunc(syncsetId, clusterId)
}

// DeleteCalls gets all the calls that were made to Delete.
// Check the length with:
//     len(mockedSyncsetService.DeleteCalls())
func (mock *SyncsetServiceMock) DeleteCalls() []struct {
	SyncsetId string
	ClusterId string
} {
	var calls []struct {
		SyncsetId string
		ClusterId string
	}
	mock.lockDelete.RLock()
	calls = mock.calls.Delete
	mock.lockDelete.RUnlock()
	return calls
}
