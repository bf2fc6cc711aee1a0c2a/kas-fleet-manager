// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package services

import (
	"context"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	apiErrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"sync"
)

// Ensure, that ConnectorsServiceMock does implement ConnectorsService.
// If this is not the case, regenerate this file with moq.
var _ ConnectorsService = &ConnectorsServiceMock{}

// ConnectorsServiceMock is a mock implementation of ConnectorsService.
//
// 	func TestSomethingThatUsesConnectorsService(t *testing.T) {
//
// 		// make and configure a mocked ConnectorsService
// 		mockedConnectorsService := &ConnectorsServiceMock{
// 			CreateFunc: func(ctx context.Context, resource *api.Connector) *apiErrors.ServiceError {
// 				panic("mock out the Create method")
// 			},
// 			DeleteFunc: func(ctx context.Context, kid string, id string) *apiErrors.ServiceError {
// 				panic("mock out the Delete method")
// 			},
// 			ForEachInStatusFunc: func(statuses []string, f func(*api.Connector) *apiErrors.ServiceError) *apiErrors.ServiceError {
// 				panic("mock out the ForEachInStatus method")
// 			},
// 			GetFunc: func(ctx context.Context, kid string, id string, tid string) (*api.Connector, *apiErrors.ServiceError) {
// 				panic("mock out the Get method")
// 			},
// 			ListFunc: func(ctx context.Context, kid string, listArgs *ListArguments, tid string) (api.ConnectorList, *api.PagingMeta, *apiErrors.ServiceError) {
// 				panic("mock out the List method")
// 			},
// 			UpdateFunc: func(ctx context.Context, resource *api.Connector) *apiErrors.ServiceError {
// 				panic("mock out the Update method")
// 			},
// 		}
//
// 		// use mockedConnectorsService in code that requires ConnectorsService
// 		// and then make assertions.
//
// 	}
type ConnectorsServiceMock struct {
	// CreateFunc mocks the Create method.
	CreateFunc func(ctx context.Context, resource *api.Connector) *apiErrors.ServiceError

	// DeleteFunc mocks the Delete method.
	DeleteFunc func(ctx context.Context, kid string, id string) *apiErrors.ServiceError

	// ForEachInStatusFunc mocks the ForEachInStatus method.
	ForEachInStatusFunc func(statuses []string, f func(*api.Connector) *apiErrors.ServiceError) *apiErrors.ServiceError

	// GetFunc mocks the Get method.
	GetFunc func(ctx context.Context, kid string, id string, tid string) (*api.Connector, *apiErrors.ServiceError)

	// ListFunc mocks the List method.
	ListFunc func(ctx context.Context, kid string, listArgs *ListArguments, tid string) (api.ConnectorList, *api.PagingMeta, *apiErrors.ServiceError)

	// UpdateFunc mocks the Update method.
	UpdateFunc func(ctx context.Context, resource *api.Connector) *apiErrors.ServiceError

	// calls tracks calls to the methods.
	calls struct {
		// Create holds details about calls to the Create method.
		Create []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Resource is the resource argument value.
			Resource *api.Connector
		}
		// Delete holds details about calls to the Delete method.
		Delete []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Kid is the kid argument value.
			Kid string
			// ID is the id argument value.
			ID string
		}
		// ForEachInStatus holds details about calls to the ForEachInStatus method.
		ForEachInStatus []struct {
			// Statuses is the statuses argument value.
			Statuses []string
			// F is the f argument value.
			F func(*api.Connector) *apiErrors.ServiceError
		}
		// Get holds details about calls to the Get method.
		Get []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Kid is the kid argument value.
			Kid string
			// ID is the id argument value.
			ID string
			// Tid is the tid argument value.
			Tid string
		}
		// List holds details about calls to the List method.
		List []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Kid is the kid argument value.
			Kid string
			// ListArgs is the listArgs argument value.
			ListArgs *ListArguments
			// Tid is the tid argument value.
			Tid string
		}
		// Update holds details about calls to the Update method.
		Update []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Resource is the resource argument value.
			Resource *api.Connector
		}
	}
	lockCreate          sync.RWMutex
	lockDelete          sync.RWMutex
	lockForEachInStatus sync.RWMutex
	lockGet             sync.RWMutex
	lockList            sync.RWMutex
	lockUpdate          sync.RWMutex
}

// Create calls CreateFunc.
func (mock *ConnectorsServiceMock) Create(ctx context.Context, resource *api.Connector) *apiErrors.ServiceError {
	if mock.CreateFunc == nil {
		panic("ConnectorsServiceMock.CreateFunc: method is nil but ConnectorsService.Create was just called")
	}
	callInfo := struct {
		Ctx      context.Context
		Resource *api.Connector
	}{
		Ctx:      ctx,
		Resource: resource,
	}
	mock.lockCreate.Lock()
	mock.calls.Create = append(mock.calls.Create, callInfo)
	mock.lockCreate.Unlock()
	return mock.CreateFunc(ctx, resource)
}

// CreateCalls gets all the calls that were made to Create.
// Check the length with:
//     len(mockedConnectorsService.CreateCalls())
func (mock *ConnectorsServiceMock) CreateCalls() []struct {
	Ctx      context.Context
	Resource *api.Connector
} {
	var calls []struct {
		Ctx      context.Context
		Resource *api.Connector
	}
	mock.lockCreate.RLock()
	calls = mock.calls.Create
	mock.lockCreate.RUnlock()
	return calls
}

// Delete calls DeleteFunc.
func (mock *ConnectorsServiceMock) Delete(ctx context.Context, kid string, id string) *apiErrors.ServiceError {
	if mock.DeleteFunc == nil {
		panic("ConnectorsServiceMock.DeleteFunc: method is nil but ConnectorsService.Delete was just called")
	}
	callInfo := struct {
		Ctx context.Context
		Kid string
		ID  string
	}{
		Ctx: ctx,
		Kid: kid,
		ID:  id,
	}
	mock.lockDelete.Lock()
	mock.calls.Delete = append(mock.calls.Delete, callInfo)
	mock.lockDelete.Unlock()
	return mock.DeleteFunc(ctx, kid, id)
}

// DeleteCalls gets all the calls that were made to Delete.
// Check the length with:
//     len(mockedConnectorsService.DeleteCalls())
func (mock *ConnectorsServiceMock) DeleteCalls() []struct {
	Ctx context.Context
	Kid string
	ID  string
} {
	var calls []struct {
		Ctx context.Context
		Kid string
		ID  string
	}
	mock.lockDelete.RLock()
	calls = mock.calls.Delete
	mock.lockDelete.RUnlock()
	return calls
}

// ForEachInStatus calls ForEachInStatusFunc.
func (mock *ConnectorsServiceMock) ForEachInStatus(statuses []string, f func(*api.Connector) *apiErrors.ServiceError) *apiErrors.ServiceError {
	if mock.ForEachInStatusFunc == nil {
		panic("ConnectorsServiceMock.ForEachInStatusFunc: method is nil but ConnectorsService.ForEachInStatus was just called")
	}
	callInfo := struct {
		Statuses []string
		F        func(*api.Connector) *apiErrors.ServiceError
	}{
		Statuses: statuses,
		F:        f,
	}
	mock.lockForEachInStatus.Lock()
	mock.calls.ForEachInStatus = append(mock.calls.ForEachInStatus, callInfo)
	mock.lockForEachInStatus.Unlock()
	return mock.ForEachInStatusFunc(statuses, f)
}

// ForEachInStatusCalls gets all the calls that were made to ForEachInStatus.
// Check the length with:
//     len(mockedConnectorsService.ForEachInStatusCalls())
func (mock *ConnectorsServiceMock) ForEachInStatusCalls() []struct {
	Statuses []string
	F        func(*api.Connector) *apiErrors.ServiceError
} {
	var calls []struct {
		Statuses []string
		F        func(*api.Connector) *apiErrors.ServiceError
	}
	mock.lockForEachInStatus.RLock()
	calls = mock.calls.ForEachInStatus
	mock.lockForEachInStatus.RUnlock()
	return calls
}

// Get calls GetFunc.
func (mock *ConnectorsServiceMock) Get(ctx context.Context, kid string, id string, tid string) (*api.Connector, *apiErrors.ServiceError) {
	if mock.GetFunc == nil {
		panic("ConnectorsServiceMock.GetFunc: method is nil but ConnectorsService.Get was just called")
	}
	callInfo := struct {
		Ctx context.Context
		Kid string
		ID  string
		Tid string
	}{
		Ctx: ctx,
		Kid: kid,
		ID:  id,
		Tid: tid,
	}
	mock.lockGet.Lock()
	mock.calls.Get = append(mock.calls.Get, callInfo)
	mock.lockGet.Unlock()
	return mock.GetFunc(ctx, kid, id, tid)
}

// GetCalls gets all the calls that were made to Get.
// Check the length with:
//     len(mockedConnectorsService.GetCalls())
func (mock *ConnectorsServiceMock) GetCalls() []struct {
	Ctx context.Context
	Kid string
	ID  string
	Tid string
} {
	var calls []struct {
		Ctx context.Context
		Kid string
		ID  string
		Tid string
	}
	mock.lockGet.RLock()
	calls = mock.calls.Get
	mock.lockGet.RUnlock()
	return calls
}

// List calls ListFunc.
func (mock *ConnectorsServiceMock) List(ctx context.Context, kid string, listArgs *ListArguments, tid string) (api.ConnectorList, *api.PagingMeta, *apiErrors.ServiceError) {
	if mock.ListFunc == nil {
		panic("ConnectorsServiceMock.ListFunc: method is nil but ConnectorsService.List was just called")
	}
	callInfo := struct {
		Ctx      context.Context
		Kid      string
		ListArgs *ListArguments
		Tid      string
	}{
		Ctx:      ctx,
		Kid:      kid,
		ListArgs: listArgs,
		Tid:      tid,
	}
	mock.lockList.Lock()
	mock.calls.List = append(mock.calls.List, callInfo)
	mock.lockList.Unlock()
	return mock.ListFunc(ctx, kid, listArgs, tid)
}

// ListCalls gets all the calls that were made to List.
// Check the length with:
//     len(mockedConnectorsService.ListCalls())
func (mock *ConnectorsServiceMock) ListCalls() []struct {
	Ctx      context.Context
	Kid      string
	ListArgs *ListArguments
	Tid      string
} {
	var calls []struct {
		Ctx      context.Context
		Kid      string
		ListArgs *ListArguments
		Tid      string
	}
	mock.lockList.RLock()
	calls = mock.calls.List
	mock.lockList.RUnlock()
	return calls
}

// Update calls UpdateFunc.
func (mock *ConnectorsServiceMock) Update(ctx context.Context, resource *api.Connector) *apiErrors.ServiceError {
	if mock.UpdateFunc == nil {
		panic("ConnectorsServiceMock.UpdateFunc: method is nil but ConnectorsService.Update was just called")
	}
	callInfo := struct {
		Ctx      context.Context
		Resource *api.Connector
	}{
		Ctx:      ctx,
		Resource: resource,
	}
	mock.lockUpdate.Lock()
	mock.calls.Update = append(mock.calls.Update, callInfo)
	mock.lockUpdate.Unlock()
	return mock.UpdateFunc(ctx, resource)
}

// UpdateCalls gets all the calls that were made to Update.
// Check the length with:
//     len(mockedConnectorsService.UpdateCalls())
func (mock *ConnectorsServiceMock) UpdateCalls() []struct {
	Ctx      context.Context
	Resource *api.Connector
} {
	var calls []struct {
		Ctx      context.Context
		Resource *api.Connector
	}
	mock.lockUpdate.RLock()
	calls = mock.calls.Update
	mock.lockUpdate.RUnlock()
	return calls
}
