// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package services

import (
	"context"
	"github.com/aws/aws-sdk-go/service/route53"
	constants2 "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/constants"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	manageddinosaur "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api/manageddinosaurs.manageddinosaur.bf2.org/v1"
	serviceError "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/services"
	"sync"
)

// Ensure, that DinosaurServiceMock does implement DinosaurService.
// If this is not the case, regenerate this file with moq.
var _ DinosaurService = &DinosaurServiceMock{}

// DinosaurServiceMock is a mock implementation of DinosaurService.
//
// 	func TestSomethingThatUsesDinosaurService(t *testing.T) {
//
// 		// make and configure a mocked DinosaurService
// 		mockedDinosaurService := &DinosaurServiceMock{
// 			ChangeDinosaurCNAMErecordsFunc: func(dinosaurRequest *dbapi.DinosaurRequest, action DinosaurRoutesAction) (*route53.ChangeResourceRecordSetsOutput, *serviceError.ServiceError) {
// 				panic("mock out the ChangeDinosaurCNAMErecords method")
// 			},
// 			CountByStatusFunc: func(status []constants2.DinosaurStatus) ([]DinosaurStatusCount, error) {
// 				panic("mock out the CountByStatus method")
// 			},
// 			DeleteFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *serviceError.ServiceError {
// 				panic("mock out the Delete method")
// 			},
// 			DeprovisionDinosaurForUsersFunc: func(users []string) *serviceError.ServiceError {
// 				panic("mock out the DeprovisionDinosaurForUsers method")
// 			},
// 			DeprovisionExpiredDinosaursFunc: func(dinosaurAgeInHours int) *serviceError.ServiceError {
// 				panic("mock out the DeprovisionExpiredDinosaurs method")
// 			},
// 			GetFunc: func(ctx context.Context, id string) (*dbapi.DinosaurRequest, *serviceError.ServiceError) {
// 				panic("mock out the Get method")
// 			},
// 			GetByIdFunc: func(id string) (*dbapi.DinosaurRequest, *serviceError.ServiceError) {
// 				panic("mock out the GetById method")
// 			},
// 			GetManagedDinosaurByClusterIDFunc: func(clusterID string) ([]manageddinosaur.ManagedDinosaur, *serviceError.ServiceError) {
// 				panic("mock out the GetManagedDinosaurByClusterID method")
// 			},
// 			HasAvailableCapacityFunc: func() (bool, *serviceError.ServiceError) {
// 				panic("mock out the HasAvailableCapacity method")
// 			},
// 			ListFunc: func(ctx context.Context, listArgs *services.ListArguments) (dbapi.DinosaurList, *api.PagingMeta, *serviceError.ServiceError) {
// 				panic("mock out the List method")
// 			},
// 			ListByStatusFunc: func(status ...constants2.DinosaurStatus) ([]*dbapi.DinosaurRequest, *serviceError.ServiceError) {
// 				panic("mock out the ListByStatus method")
// 			},
// 			ListComponentVersionsFunc: func() ([]DinosaurComponentVersions, error) {
// 				panic("mock out the ListComponentVersions method")
// 			},
// 			ListDinosaursWithRoutesNotCreatedFunc: func() ([]*dbapi.DinosaurRequest, *serviceError.ServiceError) {
// 				panic("mock out the ListDinosaursWithRoutesNotCreated method")
// 			},
// 			PrepareDinosaurRequestFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *serviceError.ServiceError {
// 				panic("mock out the PrepareDinosaurRequest method")
// 			},
// 			RegisterDinosaurDeprovisionJobFunc: func(ctx context.Context, id string) *serviceError.ServiceError {
// 				panic("mock out the RegisterDinosaurDeprovisionJob method")
// 			},
// 			RegisterDinosaurJobFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *serviceError.ServiceError {
// 				panic("mock out the RegisterDinosaurJob method")
// 			},
// 			UpdateFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *serviceError.ServiceError {
// 				panic("mock out the Update method")
// 			},
// 			UpdateStatusFunc: func(id string, status constants2.DinosaurStatus) (bool, *serviceError.ServiceError) {
// 				panic("mock out the UpdateStatus method")
// 			},
// 			UpdatesFunc: func(dinosaurRequest *dbapi.DinosaurRequest, values map[string]interface{}) *serviceError.ServiceError {
// 				panic("mock out the Updates method")
// 			},
// 			VerifyAndUpdateDinosaurFunc: func(ctx context.Context, dinosaurRequest *dbapi.DinosaurRequest) *serviceError.ServiceError {
// 				panic("mock out the VerifyAndUpdateDinosaur method")
// 			},
// 			VerifyAndUpdateDinosaurAdminFunc: func(ctx context.Context, dinosaurRequest *dbapi.DinosaurRequest) *serviceError.ServiceError {
// 				panic("mock out the VerifyAndUpdateDinosaurAdmin method")
// 			},
// 		}
//
// 		// use mockedDinosaurService in code that requires DinosaurService
// 		// and then make assertions.
//
// 	}
type DinosaurServiceMock struct {
	// ChangeDinosaurCNAMErecordsFunc mocks the ChangeDinosaurCNAMErecords method.
	ChangeDinosaurCNAMErecordsFunc func(dinosaurRequest *dbapi.DinosaurRequest, action DinosaurRoutesAction) (*route53.ChangeResourceRecordSetsOutput, *serviceError.ServiceError)

	// CountByStatusFunc mocks the CountByStatus method.
	CountByStatusFunc func(status []constants2.DinosaurStatus) ([]DinosaurStatusCount, error)

	// DeleteFunc mocks the Delete method.
	DeleteFunc func(dinosaurRequest *dbapi.DinosaurRequest) *serviceError.ServiceError

	// DeprovisionDinosaurForUsersFunc mocks the DeprovisionDinosaurForUsers method.
	DeprovisionDinosaurForUsersFunc func(users []string) *serviceError.ServiceError

	// DeprovisionExpiredDinosaursFunc mocks the DeprovisionExpiredDinosaurs method.
	DeprovisionExpiredDinosaursFunc func(dinosaurAgeInHours int) *serviceError.ServiceError

	// GetFunc mocks the Get method.
	GetFunc func(ctx context.Context, id string) (*dbapi.DinosaurRequest, *serviceError.ServiceError)

	// GetByIdFunc mocks the GetById method.
	GetByIdFunc func(id string) (*dbapi.DinosaurRequest, *serviceError.ServiceError)

	// GetManagedDinosaurByClusterIDFunc mocks the GetManagedDinosaurByClusterID method.
	GetManagedDinosaurByClusterIDFunc func(clusterID string) ([]manageddinosaur.ManagedDinosaur, *serviceError.ServiceError)

	// HasAvailableCapacityFunc mocks the HasAvailableCapacity method.
	HasAvailableCapacityFunc func() (bool, *serviceError.ServiceError)

	// ListFunc mocks the List method.
	ListFunc func(ctx context.Context, listArgs *services.ListArguments) (dbapi.DinosaurList, *api.PagingMeta, *serviceError.ServiceError)

	// ListByStatusFunc mocks the ListByStatus method.
	ListByStatusFunc func(status ...constants2.DinosaurStatus) ([]*dbapi.DinosaurRequest, *serviceError.ServiceError)

	// ListComponentVersionsFunc mocks the ListComponentVersions method.
	ListComponentVersionsFunc func() ([]DinosaurComponentVersions, error)

	// ListDinosaursWithRoutesNotCreatedFunc mocks the ListDinosaursWithRoutesNotCreated method.
	ListDinosaursWithRoutesNotCreatedFunc func() ([]*dbapi.DinosaurRequest, *serviceError.ServiceError)

	// PrepareDinosaurRequestFunc mocks the PrepareDinosaurRequest method.
	PrepareDinosaurRequestFunc func(dinosaurRequest *dbapi.DinosaurRequest) *serviceError.ServiceError

	// RegisterDinosaurDeprovisionJobFunc mocks the RegisterDinosaurDeprovisionJob method.
	RegisterDinosaurDeprovisionJobFunc func(ctx context.Context, id string) *serviceError.ServiceError

	// RegisterDinosaurJobFunc mocks the RegisterDinosaurJob method.
	RegisterDinosaurJobFunc func(dinosaurRequest *dbapi.DinosaurRequest) *serviceError.ServiceError

	// UpdateFunc mocks the Update method.
	UpdateFunc func(dinosaurRequest *dbapi.DinosaurRequest) *serviceError.ServiceError

	// UpdateStatusFunc mocks the UpdateStatus method.
	UpdateStatusFunc func(id string, status constants2.DinosaurStatus) (bool, *serviceError.ServiceError)

	// UpdatesFunc mocks the Updates method.
	UpdatesFunc func(dinosaurRequest *dbapi.DinosaurRequest, values map[string]interface{}) *serviceError.ServiceError

	// VerifyAndUpdateDinosaurFunc mocks the VerifyAndUpdateDinosaur method.
	VerifyAndUpdateDinosaurFunc func(ctx context.Context, dinosaurRequest *dbapi.DinosaurRequest) *serviceError.ServiceError

	// VerifyAndUpdateDinosaurAdminFunc mocks the VerifyAndUpdateDinosaurAdmin method.
	VerifyAndUpdateDinosaurAdminFunc func(ctx context.Context, dinosaurRequest *dbapi.DinosaurRequest) *serviceError.ServiceError

	// calls tracks calls to the methods.
	calls struct {
		// ChangeDinosaurCNAMErecords holds details about calls to the ChangeDinosaurCNAMErecords method.
		ChangeDinosaurCNAMErecords []struct {
			// DinosaurRequest is the dinosaurRequest argument value.
			DinosaurRequest *dbapi.DinosaurRequest
			// Action is the action argument value.
			Action DinosaurRoutesAction
		}
		// CountByStatus holds details about calls to the CountByStatus method.
		CountByStatus []struct {
			// Status is the status argument value.
			Status []constants2.DinosaurStatus
		}
		// Delete holds details about calls to the Delete method.
		Delete []struct {
			// DinosaurRequest is the dinosaurRequest argument value.
			DinosaurRequest *dbapi.DinosaurRequest
		}
		// DeprovisionDinosaurForUsers holds details about calls to the DeprovisionDinosaurForUsers method.
		DeprovisionDinosaurForUsers []struct {
			// Users is the users argument value.
			Users []string
		}
		// DeprovisionExpiredDinosaurs holds details about calls to the DeprovisionExpiredDinosaurs method.
		DeprovisionExpiredDinosaurs []struct {
			// DinosaurAgeInHours is the dinosaurAgeInHours argument value.
			DinosaurAgeInHours int
		}
		// Get holds details about calls to the Get method.
		Get []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
		}
		// GetById holds details about calls to the GetById method.
		GetById []struct {
			// ID is the id argument value.
			ID string
		}
		// GetManagedDinosaurByClusterID holds details about calls to the GetManagedDinosaurByClusterID method.
		GetManagedDinosaurByClusterID []struct {
			// ClusterID is the clusterID argument value.
			ClusterID string
		}
		// HasAvailableCapacity holds details about calls to the HasAvailableCapacity method.
		HasAvailableCapacity []struct {
		}
		// List holds details about calls to the List method.
		List []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ListArgs is the listArgs argument value.
			ListArgs *services.ListArguments
		}
		// ListByStatus holds details about calls to the ListByStatus method.
		ListByStatus []struct {
			// Status is the status argument value.
			Status []constants2.DinosaurStatus
		}
		// ListComponentVersions holds details about calls to the ListComponentVersions method.
		ListComponentVersions []struct {
		}
		// ListDinosaursWithRoutesNotCreated holds details about calls to the ListDinosaursWithRoutesNotCreated method.
		ListDinosaursWithRoutesNotCreated []struct {
		}
		// PrepareDinosaurRequest holds details about calls to the PrepareDinosaurRequest method.
		PrepareDinosaurRequest []struct {
			// DinosaurRequest is the dinosaurRequest argument value.
			DinosaurRequest *dbapi.DinosaurRequest
		}
		// RegisterDinosaurDeprovisionJob holds details about calls to the RegisterDinosaurDeprovisionJob method.
		RegisterDinosaurDeprovisionJob []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
		}
		// RegisterDinosaurJob holds details about calls to the RegisterDinosaurJob method.
		RegisterDinosaurJob []struct {
			// DinosaurRequest is the dinosaurRequest argument value.
			DinosaurRequest *dbapi.DinosaurRequest
		}
		// Update holds details about calls to the Update method.
		Update []struct {
			// DinosaurRequest is the dinosaurRequest argument value.
			DinosaurRequest *dbapi.DinosaurRequest
		}
		// UpdateStatus holds details about calls to the UpdateStatus method.
		UpdateStatus []struct {
			// ID is the id argument value.
			ID string
			// Status is the status argument value.
			Status constants2.DinosaurStatus
		}
		// Updates holds details about calls to the Updates method.
		Updates []struct {
			// DinosaurRequest is the dinosaurRequest argument value.
			DinosaurRequest *dbapi.DinosaurRequest
			// Values is the values argument value.
			Values map[string]interface{}
		}
		// VerifyAndUpdateDinosaur holds details about calls to the VerifyAndUpdateDinosaur method.
		VerifyAndUpdateDinosaur []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// DinosaurRequest is the dinosaurRequest argument value.
			DinosaurRequest *dbapi.DinosaurRequest
		}
		// VerifyAndUpdateDinosaurAdmin holds details about calls to the VerifyAndUpdateDinosaurAdmin method.
		VerifyAndUpdateDinosaurAdmin []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// DinosaurRequest is the dinosaurRequest argument value.
			DinosaurRequest *dbapi.DinosaurRequest
		}
	}
	lockChangeDinosaurCNAMErecords        sync.RWMutex
	lockCountByStatus                     sync.RWMutex
	lockDelete                            sync.RWMutex
	lockDeprovisionDinosaurForUsers       sync.RWMutex
	lockDeprovisionExpiredDinosaurs       sync.RWMutex
	lockGet                               sync.RWMutex
	lockGetById                           sync.RWMutex
	lockGetManagedDinosaurByClusterID     sync.RWMutex
	lockHasAvailableCapacity              sync.RWMutex
	lockList                              sync.RWMutex
	lockListByStatus                      sync.RWMutex
	lockListComponentVersions             sync.RWMutex
	lockListDinosaursWithRoutesNotCreated sync.RWMutex
	lockPrepareDinosaurRequest            sync.RWMutex
	lockRegisterDinosaurDeprovisionJob    sync.RWMutex
	lockRegisterDinosaurJob               sync.RWMutex
	lockUpdate                            sync.RWMutex
	lockUpdateStatus                      sync.RWMutex
	lockUpdates                           sync.RWMutex
	lockVerifyAndUpdateDinosaur           sync.RWMutex
	lockVerifyAndUpdateDinosaurAdmin      sync.RWMutex
}

// ChangeDinosaurCNAMErecords calls ChangeDinosaurCNAMErecordsFunc.
func (mock *DinosaurServiceMock) ChangeDinosaurCNAMErecords(dinosaurRequest *dbapi.DinosaurRequest, action DinosaurRoutesAction) (*route53.ChangeResourceRecordSetsOutput, *serviceError.ServiceError) {
	if mock.ChangeDinosaurCNAMErecordsFunc == nil {
		panic("DinosaurServiceMock.ChangeDinosaurCNAMErecordsFunc: method is nil but DinosaurService.ChangeDinosaurCNAMErecords was just called")
	}
	callInfo := struct {
		DinosaurRequest *dbapi.DinosaurRequest
		Action          DinosaurRoutesAction
	}{
		DinosaurRequest: dinosaurRequest,
		Action:          action,
	}
	mock.lockChangeDinosaurCNAMErecords.Lock()
	mock.calls.ChangeDinosaurCNAMErecords = append(mock.calls.ChangeDinosaurCNAMErecords, callInfo)
	mock.lockChangeDinosaurCNAMErecords.Unlock()
	return mock.ChangeDinosaurCNAMErecordsFunc(dinosaurRequest, action)
}

// ChangeDinosaurCNAMErecordsCalls gets all the calls that were made to ChangeDinosaurCNAMErecords.
// Check the length with:
//     len(mockedDinosaurService.ChangeDinosaurCNAMErecordsCalls())
func (mock *DinosaurServiceMock) ChangeDinosaurCNAMErecordsCalls() []struct {
	DinosaurRequest *dbapi.DinosaurRequest
	Action          DinosaurRoutesAction
} {
	var calls []struct {
		DinosaurRequest *dbapi.DinosaurRequest
		Action          DinosaurRoutesAction
	}
	mock.lockChangeDinosaurCNAMErecords.RLock()
	calls = mock.calls.ChangeDinosaurCNAMErecords
	mock.lockChangeDinosaurCNAMErecords.RUnlock()
	return calls
}

// CountByStatus calls CountByStatusFunc.
func (mock *DinosaurServiceMock) CountByStatus(status []constants2.DinosaurStatus) ([]DinosaurStatusCount, error) {
	if mock.CountByStatusFunc == nil {
		panic("DinosaurServiceMock.CountByStatusFunc: method is nil but DinosaurService.CountByStatus was just called")
	}
	callInfo := struct {
		Status []constants2.DinosaurStatus
	}{
		Status: status,
	}
	mock.lockCountByStatus.Lock()
	mock.calls.CountByStatus = append(mock.calls.CountByStatus, callInfo)
	mock.lockCountByStatus.Unlock()
	return mock.CountByStatusFunc(status)
}

// CountByStatusCalls gets all the calls that were made to CountByStatus.
// Check the length with:
//     len(mockedDinosaurService.CountByStatusCalls())
func (mock *DinosaurServiceMock) CountByStatusCalls() []struct {
	Status []constants2.DinosaurStatus
} {
	var calls []struct {
		Status []constants2.DinosaurStatus
	}
	mock.lockCountByStatus.RLock()
	calls = mock.calls.CountByStatus
	mock.lockCountByStatus.RUnlock()
	return calls
}

// Delete calls DeleteFunc.
func (mock *DinosaurServiceMock) Delete(dinosaurRequest *dbapi.DinosaurRequest) *serviceError.ServiceError {
	if mock.DeleteFunc == nil {
		panic("DinosaurServiceMock.DeleteFunc: method is nil but DinosaurService.Delete was just called")
	}
	callInfo := struct {
		DinosaurRequest *dbapi.DinosaurRequest
	}{
		DinosaurRequest: dinosaurRequest,
	}
	mock.lockDelete.Lock()
	mock.calls.Delete = append(mock.calls.Delete, callInfo)
	mock.lockDelete.Unlock()
	return mock.DeleteFunc(dinosaurRequest)
}

// DeleteCalls gets all the calls that were made to Delete.
// Check the length with:
//     len(mockedDinosaurService.DeleteCalls())
func (mock *DinosaurServiceMock) DeleteCalls() []struct {
	DinosaurRequest *dbapi.DinosaurRequest
} {
	var calls []struct {
		DinosaurRequest *dbapi.DinosaurRequest
	}
	mock.lockDelete.RLock()
	calls = mock.calls.Delete
	mock.lockDelete.RUnlock()
	return calls
}

// DeprovisionDinosaurForUsers calls DeprovisionDinosaurForUsersFunc.
func (mock *DinosaurServiceMock) DeprovisionDinosaurForUsers(users []string) *serviceError.ServiceError {
	if mock.DeprovisionDinosaurForUsersFunc == nil {
		panic("DinosaurServiceMock.DeprovisionDinosaurForUsersFunc: method is nil but DinosaurService.DeprovisionDinosaurForUsers was just called")
	}
	callInfo := struct {
		Users []string
	}{
		Users: users,
	}
	mock.lockDeprovisionDinosaurForUsers.Lock()
	mock.calls.DeprovisionDinosaurForUsers = append(mock.calls.DeprovisionDinosaurForUsers, callInfo)
	mock.lockDeprovisionDinosaurForUsers.Unlock()
	return mock.DeprovisionDinosaurForUsersFunc(users)
}

// DeprovisionDinosaurForUsersCalls gets all the calls that were made to DeprovisionDinosaurForUsers.
// Check the length with:
//     len(mockedDinosaurService.DeprovisionDinosaurForUsersCalls())
func (mock *DinosaurServiceMock) DeprovisionDinosaurForUsersCalls() []struct {
	Users []string
} {
	var calls []struct {
		Users []string
	}
	mock.lockDeprovisionDinosaurForUsers.RLock()
	calls = mock.calls.DeprovisionDinosaurForUsers
	mock.lockDeprovisionDinosaurForUsers.RUnlock()
	return calls
}

// DeprovisionExpiredDinosaurs calls DeprovisionExpiredDinosaursFunc.
func (mock *DinosaurServiceMock) DeprovisionExpiredDinosaurs(dinosaurAgeInHours int) *serviceError.ServiceError {
	if mock.DeprovisionExpiredDinosaursFunc == nil {
		panic("DinosaurServiceMock.DeprovisionExpiredDinosaursFunc: method is nil but DinosaurService.DeprovisionExpiredDinosaurs was just called")
	}
	callInfo := struct {
		DinosaurAgeInHours int
	}{
		DinosaurAgeInHours: dinosaurAgeInHours,
	}
	mock.lockDeprovisionExpiredDinosaurs.Lock()
	mock.calls.DeprovisionExpiredDinosaurs = append(mock.calls.DeprovisionExpiredDinosaurs, callInfo)
	mock.lockDeprovisionExpiredDinosaurs.Unlock()
	return mock.DeprovisionExpiredDinosaursFunc(dinosaurAgeInHours)
}

// DeprovisionExpiredDinosaursCalls gets all the calls that were made to DeprovisionExpiredDinosaurs.
// Check the length with:
//     len(mockedDinosaurService.DeprovisionExpiredDinosaursCalls())
func (mock *DinosaurServiceMock) DeprovisionExpiredDinosaursCalls() []struct {
	DinosaurAgeInHours int
} {
	var calls []struct {
		DinosaurAgeInHours int
	}
	mock.lockDeprovisionExpiredDinosaurs.RLock()
	calls = mock.calls.DeprovisionExpiredDinosaurs
	mock.lockDeprovisionExpiredDinosaurs.RUnlock()
	return calls
}

// Get calls GetFunc.
func (mock *DinosaurServiceMock) Get(ctx context.Context, id string) (*dbapi.DinosaurRequest, *serviceError.ServiceError) {
	if mock.GetFunc == nil {
		panic("DinosaurServiceMock.GetFunc: method is nil but DinosaurService.Get was just called")
	}
	callInfo := struct {
		Ctx context.Context
		ID  string
	}{
		Ctx: ctx,
		ID:  id,
	}
	mock.lockGet.Lock()
	mock.calls.Get = append(mock.calls.Get, callInfo)
	mock.lockGet.Unlock()
	return mock.GetFunc(ctx, id)
}

// GetCalls gets all the calls that were made to Get.
// Check the length with:
//     len(mockedDinosaurService.GetCalls())
func (mock *DinosaurServiceMock) GetCalls() []struct {
	Ctx context.Context
	ID  string
} {
	var calls []struct {
		Ctx context.Context
		ID  string
	}
	mock.lockGet.RLock()
	calls = mock.calls.Get
	mock.lockGet.RUnlock()
	return calls
}

// GetById calls GetByIdFunc.
func (mock *DinosaurServiceMock) GetById(id string) (*dbapi.DinosaurRequest, *serviceError.ServiceError) {
	if mock.GetByIdFunc == nil {
		panic("DinosaurServiceMock.GetByIdFunc: method is nil but DinosaurService.GetById was just called")
	}
	callInfo := struct {
		ID string
	}{
		ID: id,
	}
	mock.lockGetById.Lock()
	mock.calls.GetById = append(mock.calls.GetById, callInfo)
	mock.lockGetById.Unlock()
	return mock.GetByIdFunc(id)
}

// GetByIdCalls gets all the calls that were made to GetById.
// Check the length with:
//     len(mockedDinosaurService.GetByIdCalls())
func (mock *DinosaurServiceMock) GetByIdCalls() []struct {
	ID string
} {
	var calls []struct {
		ID string
	}
	mock.lockGetById.RLock()
	calls = mock.calls.GetById
	mock.lockGetById.RUnlock()
	return calls
}

// GetManagedDinosaurByClusterID calls GetManagedDinosaurByClusterIDFunc.
func (mock *DinosaurServiceMock) GetManagedDinosaurByClusterID(clusterID string) ([]manageddinosaur.ManagedDinosaur, *serviceError.ServiceError) {
	if mock.GetManagedDinosaurByClusterIDFunc == nil {
		panic("DinosaurServiceMock.GetManagedDinosaurByClusterIDFunc: method is nil but DinosaurService.GetManagedDinosaurByClusterID was just called")
	}
	callInfo := struct {
		ClusterID string
	}{
		ClusterID: clusterID,
	}
	mock.lockGetManagedDinosaurByClusterID.Lock()
	mock.calls.GetManagedDinosaurByClusterID = append(mock.calls.GetManagedDinosaurByClusterID, callInfo)
	mock.lockGetManagedDinosaurByClusterID.Unlock()
	return mock.GetManagedDinosaurByClusterIDFunc(clusterID)
}

// GetManagedDinosaurByClusterIDCalls gets all the calls that were made to GetManagedDinosaurByClusterID.
// Check the length with:
//     len(mockedDinosaurService.GetManagedDinosaurByClusterIDCalls())
func (mock *DinosaurServiceMock) GetManagedDinosaurByClusterIDCalls() []struct {
	ClusterID string
} {
	var calls []struct {
		ClusterID string
	}
	mock.lockGetManagedDinosaurByClusterID.RLock()
	calls = mock.calls.GetManagedDinosaurByClusterID
	mock.lockGetManagedDinosaurByClusterID.RUnlock()
	return calls
}

// HasAvailableCapacity calls HasAvailableCapacityFunc.
func (mock *DinosaurServiceMock) HasAvailableCapacity() (bool, *serviceError.ServiceError) {
	if mock.HasAvailableCapacityFunc == nil {
		panic("DinosaurServiceMock.HasAvailableCapacityFunc: method is nil but DinosaurService.HasAvailableCapacity was just called")
	}
	callInfo := struct {
	}{}
	mock.lockHasAvailableCapacity.Lock()
	mock.calls.HasAvailableCapacity = append(mock.calls.HasAvailableCapacity, callInfo)
	mock.lockHasAvailableCapacity.Unlock()
	return mock.HasAvailableCapacityFunc()
}

// HasAvailableCapacityCalls gets all the calls that were made to HasAvailableCapacity.
// Check the length with:
//     len(mockedDinosaurService.HasAvailableCapacityCalls())
func (mock *DinosaurServiceMock) HasAvailableCapacityCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockHasAvailableCapacity.RLock()
	calls = mock.calls.HasAvailableCapacity
	mock.lockHasAvailableCapacity.RUnlock()
	return calls
}

// List calls ListFunc.
func (mock *DinosaurServiceMock) List(ctx context.Context, listArgs *services.ListArguments) (dbapi.DinosaurList, *api.PagingMeta, *serviceError.ServiceError) {
	if mock.ListFunc == nil {
		panic("DinosaurServiceMock.ListFunc: method is nil but DinosaurService.List was just called")
	}
	callInfo := struct {
		Ctx      context.Context
		ListArgs *services.ListArguments
	}{
		Ctx:      ctx,
		ListArgs: listArgs,
	}
	mock.lockList.Lock()
	mock.calls.List = append(mock.calls.List, callInfo)
	mock.lockList.Unlock()
	return mock.ListFunc(ctx, listArgs)
}

// ListCalls gets all the calls that were made to List.
// Check the length with:
//     len(mockedDinosaurService.ListCalls())
func (mock *DinosaurServiceMock) ListCalls() []struct {
	Ctx      context.Context
	ListArgs *services.ListArguments
} {
	var calls []struct {
		Ctx      context.Context
		ListArgs *services.ListArguments
	}
	mock.lockList.RLock()
	calls = mock.calls.List
	mock.lockList.RUnlock()
	return calls
}

// ListByStatus calls ListByStatusFunc.
func (mock *DinosaurServiceMock) ListByStatus(status ...constants2.DinosaurStatus) ([]*dbapi.DinosaurRequest, *serviceError.ServiceError) {
	if mock.ListByStatusFunc == nil {
		panic("DinosaurServiceMock.ListByStatusFunc: method is nil but DinosaurService.ListByStatus was just called")
	}
	callInfo := struct {
		Status []constants2.DinosaurStatus
	}{
		Status: status,
	}
	mock.lockListByStatus.Lock()
	mock.calls.ListByStatus = append(mock.calls.ListByStatus, callInfo)
	mock.lockListByStatus.Unlock()
	return mock.ListByStatusFunc(status...)
}

// ListByStatusCalls gets all the calls that were made to ListByStatus.
// Check the length with:
//     len(mockedDinosaurService.ListByStatusCalls())
func (mock *DinosaurServiceMock) ListByStatusCalls() []struct {
	Status []constants2.DinosaurStatus
} {
	var calls []struct {
		Status []constants2.DinosaurStatus
	}
	mock.lockListByStatus.RLock()
	calls = mock.calls.ListByStatus
	mock.lockListByStatus.RUnlock()
	return calls
}

// ListComponentVersions calls ListComponentVersionsFunc.
func (mock *DinosaurServiceMock) ListComponentVersions() ([]DinosaurComponentVersions, error) {
	if mock.ListComponentVersionsFunc == nil {
		panic("DinosaurServiceMock.ListComponentVersionsFunc: method is nil but DinosaurService.ListComponentVersions was just called")
	}
	callInfo := struct {
	}{}
	mock.lockListComponentVersions.Lock()
	mock.calls.ListComponentVersions = append(mock.calls.ListComponentVersions, callInfo)
	mock.lockListComponentVersions.Unlock()
	return mock.ListComponentVersionsFunc()
}

// ListComponentVersionsCalls gets all the calls that were made to ListComponentVersions.
// Check the length with:
//     len(mockedDinosaurService.ListComponentVersionsCalls())
func (mock *DinosaurServiceMock) ListComponentVersionsCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockListComponentVersions.RLock()
	calls = mock.calls.ListComponentVersions
	mock.lockListComponentVersions.RUnlock()
	return calls
}

// ListDinosaursWithRoutesNotCreated calls ListDinosaursWithRoutesNotCreatedFunc.
func (mock *DinosaurServiceMock) ListDinosaursWithRoutesNotCreated() ([]*dbapi.DinosaurRequest, *serviceError.ServiceError) {
	if mock.ListDinosaursWithRoutesNotCreatedFunc == nil {
		panic("DinosaurServiceMock.ListDinosaursWithRoutesNotCreatedFunc: method is nil but DinosaurService.ListDinosaursWithRoutesNotCreated was just called")
	}
	callInfo := struct {
	}{}
	mock.lockListDinosaursWithRoutesNotCreated.Lock()
	mock.calls.ListDinosaursWithRoutesNotCreated = append(mock.calls.ListDinosaursWithRoutesNotCreated, callInfo)
	mock.lockListDinosaursWithRoutesNotCreated.Unlock()
	return mock.ListDinosaursWithRoutesNotCreatedFunc()
}

// ListDinosaursWithRoutesNotCreatedCalls gets all the calls that were made to ListDinosaursWithRoutesNotCreated.
// Check the length with:
//     len(mockedDinosaurService.ListDinosaursWithRoutesNotCreatedCalls())
func (mock *DinosaurServiceMock) ListDinosaursWithRoutesNotCreatedCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockListDinosaursWithRoutesNotCreated.RLock()
	calls = mock.calls.ListDinosaursWithRoutesNotCreated
	mock.lockListDinosaursWithRoutesNotCreated.RUnlock()
	return calls
}

// PrepareDinosaurRequest calls PrepareDinosaurRequestFunc.
func (mock *DinosaurServiceMock) PrepareDinosaurRequest(dinosaurRequest *dbapi.DinosaurRequest) *serviceError.ServiceError {
	if mock.PrepareDinosaurRequestFunc == nil {
		panic("DinosaurServiceMock.PrepareDinosaurRequestFunc: method is nil but DinosaurService.PrepareDinosaurRequest was just called")
	}
	callInfo := struct {
		DinosaurRequest *dbapi.DinosaurRequest
	}{
		DinosaurRequest: dinosaurRequest,
	}
	mock.lockPrepareDinosaurRequest.Lock()
	mock.calls.PrepareDinosaurRequest = append(mock.calls.PrepareDinosaurRequest, callInfo)
	mock.lockPrepareDinosaurRequest.Unlock()
	return mock.PrepareDinosaurRequestFunc(dinosaurRequest)
}

// PrepareDinosaurRequestCalls gets all the calls that were made to PrepareDinosaurRequest.
// Check the length with:
//     len(mockedDinosaurService.PrepareDinosaurRequestCalls())
func (mock *DinosaurServiceMock) PrepareDinosaurRequestCalls() []struct {
	DinosaurRequest *dbapi.DinosaurRequest
} {
	var calls []struct {
		DinosaurRequest *dbapi.DinosaurRequest
	}
	mock.lockPrepareDinosaurRequest.RLock()
	calls = mock.calls.PrepareDinosaurRequest
	mock.lockPrepareDinosaurRequest.RUnlock()
	return calls
}

// RegisterDinosaurDeprovisionJob calls RegisterDinosaurDeprovisionJobFunc.
func (mock *DinosaurServiceMock) RegisterDinosaurDeprovisionJob(ctx context.Context, id string) *serviceError.ServiceError {
	if mock.RegisterDinosaurDeprovisionJobFunc == nil {
		panic("DinosaurServiceMock.RegisterDinosaurDeprovisionJobFunc: method is nil but DinosaurService.RegisterDinosaurDeprovisionJob was just called")
	}
	callInfo := struct {
		Ctx context.Context
		ID  string
	}{
		Ctx: ctx,
		ID:  id,
	}
	mock.lockRegisterDinosaurDeprovisionJob.Lock()
	mock.calls.RegisterDinosaurDeprovisionJob = append(mock.calls.RegisterDinosaurDeprovisionJob, callInfo)
	mock.lockRegisterDinosaurDeprovisionJob.Unlock()
	return mock.RegisterDinosaurDeprovisionJobFunc(ctx, id)
}

// RegisterDinosaurDeprovisionJobCalls gets all the calls that were made to RegisterDinosaurDeprovisionJob.
// Check the length with:
//     len(mockedDinosaurService.RegisterDinosaurDeprovisionJobCalls())
func (mock *DinosaurServiceMock) RegisterDinosaurDeprovisionJobCalls() []struct {
	Ctx context.Context
	ID  string
} {
	var calls []struct {
		Ctx context.Context
		ID  string
	}
	mock.lockRegisterDinosaurDeprovisionJob.RLock()
	calls = mock.calls.RegisterDinosaurDeprovisionJob
	mock.lockRegisterDinosaurDeprovisionJob.RUnlock()
	return calls
}

// RegisterDinosaurJob calls RegisterDinosaurJobFunc.
func (mock *DinosaurServiceMock) RegisterDinosaurJob(dinosaurRequest *dbapi.DinosaurRequest) *serviceError.ServiceError {
	if mock.RegisterDinosaurJobFunc == nil {
		panic("DinosaurServiceMock.RegisterDinosaurJobFunc: method is nil but DinosaurService.RegisterDinosaurJob was just called")
	}
	callInfo := struct {
		DinosaurRequest *dbapi.DinosaurRequest
	}{
		DinosaurRequest: dinosaurRequest,
	}
	mock.lockRegisterDinosaurJob.Lock()
	mock.calls.RegisterDinosaurJob = append(mock.calls.RegisterDinosaurJob, callInfo)
	mock.lockRegisterDinosaurJob.Unlock()
	return mock.RegisterDinosaurJobFunc(dinosaurRequest)
}

// RegisterDinosaurJobCalls gets all the calls that were made to RegisterDinosaurJob.
// Check the length with:
//     len(mockedDinosaurService.RegisterDinosaurJobCalls())
func (mock *DinosaurServiceMock) RegisterDinosaurJobCalls() []struct {
	DinosaurRequest *dbapi.DinosaurRequest
} {
	var calls []struct {
		DinosaurRequest *dbapi.DinosaurRequest
	}
	mock.lockRegisterDinosaurJob.RLock()
	calls = mock.calls.RegisterDinosaurJob
	mock.lockRegisterDinosaurJob.RUnlock()
	return calls
}

// Update calls UpdateFunc.
func (mock *DinosaurServiceMock) Update(dinosaurRequest *dbapi.DinosaurRequest) *serviceError.ServiceError {
	if mock.UpdateFunc == nil {
		panic("DinosaurServiceMock.UpdateFunc: method is nil but DinosaurService.Update was just called")
	}
	callInfo := struct {
		DinosaurRequest *dbapi.DinosaurRequest
	}{
		DinosaurRequest: dinosaurRequest,
	}
	mock.lockUpdate.Lock()
	mock.calls.Update = append(mock.calls.Update, callInfo)
	mock.lockUpdate.Unlock()
	return mock.UpdateFunc(dinosaurRequest)
}

// UpdateCalls gets all the calls that were made to Update.
// Check the length with:
//     len(mockedDinosaurService.UpdateCalls())
func (mock *DinosaurServiceMock) UpdateCalls() []struct {
	DinosaurRequest *dbapi.DinosaurRequest
} {
	var calls []struct {
		DinosaurRequest *dbapi.DinosaurRequest
	}
	mock.lockUpdate.RLock()
	calls = mock.calls.Update
	mock.lockUpdate.RUnlock()
	return calls
}

// UpdateStatus calls UpdateStatusFunc.
func (mock *DinosaurServiceMock) UpdateStatus(id string, status constants2.DinosaurStatus) (bool, *serviceError.ServiceError) {
	if mock.UpdateStatusFunc == nil {
		panic("DinosaurServiceMock.UpdateStatusFunc: method is nil but DinosaurService.UpdateStatus was just called")
	}
	callInfo := struct {
		ID     string
		Status constants2.DinosaurStatus
	}{
		ID:     id,
		Status: status,
	}
	mock.lockUpdateStatus.Lock()
	mock.calls.UpdateStatus = append(mock.calls.UpdateStatus, callInfo)
	mock.lockUpdateStatus.Unlock()
	return mock.UpdateStatusFunc(id, status)
}

// UpdateStatusCalls gets all the calls that were made to UpdateStatus.
// Check the length with:
//     len(mockedDinosaurService.UpdateStatusCalls())
func (mock *DinosaurServiceMock) UpdateStatusCalls() []struct {
	ID     string
	Status constants2.DinosaurStatus
} {
	var calls []struct {
		ID     string
		Status constants2.DinosaurStatus
	}
	mock.lockUpdateStatus.RLock()
	calls = mock.calls.UpdateStatus
	mock.lockUpdateStatus.RUnlock()
	return calls
}

// Updates calls UpdatesFunc.
func (mock *DinosaurServiceMock) Updates(dinosaurRequest *dbapi.DinosaurRequest, values map[string]interface{}) *serviceError.ServiceError {
	if mock.UpdatesFunc == nil {
		panic("DinosaurServiceMock.UpdatesFunc: method is nil but DinosaurService.Updates was just called")
	}
	callInfo := struct {
		DinosaurRequest *dbapi.DinosaurRequest
		Values          map[string]interface{}
	}{
		DinosaurRequest: dinosaurRequest,
		Values:          values,
	}
	mock.lockUpdates.Lock()
	mock.calls.Updates = append(mock.calls.Updates, callInfo)
	mock.lockUpdates.Unlock()
	return mock.UpdatesFunc(dinosaurRequest, values)
}

// UpdatesCalls gets all the calls that were made to Updates.
// Check the length with:
//     len(mockedDinosaurService.UpdatesCalls())
func (mock *DinosaurServiceMock) UpdatesCalls() []struct {
	DinosaurRequest *dbapi.DinosaurRequest
	Values          map[string]interface{}
} {
	var calls []struct {
		DinosaurRequest *dbapi.DinosaurRequest
		Values          map[string]interface{}
	}
	mock.lockUpdates.RLock()
	calls = mock.calls.Updates
	mock.lockUpdates.RUnlock()
	return calls
}

// VerifyAndUpdateDinosaur calls VerifyAndUpdateDinosaurFunc.
func (mock *DinosaurServiceMock) VerifyAndUpdateDinosaur(ctx context.Context, dinosaurRequest *dbapi.DinosaurRequest) *serviceError.ServiceError {
	if mock.VerifyAndUpdateDinosaurFunc == nil {
		panic("DinosaurServiceMock.VerifyAndUpdateDinosaurFunc: method is nil but DinosaurService.VerifyAndUpdateDinosaur was just called")
	}
	callInfo := struct {
		Ctx             context.Context
		DinosaurRequest *dbapi.DinosaurRequest
	}{
		Ctx:             ctx,
		DinosaurRequest: dinosaurRequest,
	}
	mock.lockVerifyAndUpdateDinosaur.Lock()
	mock.calls.VerifyAndUpdateDinosaur = append(mock.calls.VerifyAndUpdateDinosaur, callInfo)
	mock.lockVerifyAndUpdateDinosaur.Unlock()
	return mock.VerifyAndUpdateDinosaurFunc(ctx, dinosaurRequest)
}

// VerifyAndUpdateDinosaurCalls gets all the calls that were made to VerifyAndUpdateDinosaur.
// Check the length with:
//     len(mockedDinosaurService.VerifyAndUpdateDinosaurCalls())
func (mock *DinosaurServiceMock) VerifyAndUpdateDinosaurCalls() []struct {
	Ctx             context.Context
	DinosaurRequest *dbapi.DinosaurRequest
} {
	var calls []struct {
		Ctx             context.Context
		DinosaurRequest *dbapi.DinosaurRequest
	}
	mock.lockVerifyAndUpdateDinosaur.RLock()
	calls = mock.calls.VerifyAndUpdateDinosaur
	mock.lockVerifyAndUpdateDinosaur.RUnlock()
	return calls
}

// VerifyAndUpdateDinosaurAdmin calls VerifyAndUpdateDinosaurAdminFunc.
func (mock *DinosaurServiceMock) VerifyAndUpdateDinosaurAdmin(ctx context.Context, dinosaurRequest *dbapi.DinosaurRequest) *serviceError.ServiceError {
	if mock.VerifyAndUpdateDinosaurAdminFunc == nil {
		panic("DinosaurServiceMock.VerifyAndUpdateDinosaurAdminFunc: method is nil but DinosaurService.VerifyAndUpdateDinosaurAdmin was just called")
	}
	callInfo := struct {
		Ctx             context.Context
		DinosaurRequest *dbapi.DinosaurRequest
	}{
		Ctx:             ctx,
		DinosaurRequest: dinosaurRequest,
	}
	mock.lockVerifyAndUpdateDinosaurAdmin.Lock()
	mock.calls.VerifyAndUpdateDinosaurAdmin = append(mock.calls.VerifyAndUpdateDinosaurAdmin, callInfo)
	mock.lockVerifyAndUpdateDinosaurAdmin.Unlock()
	return mock.VerifyAndUpdateDinosaurAdminFunc(ctx, dinosaurRequest)
}

// VerifyAndUpdateDinosaurAdminCalls gets all the calls that were made to VerifyAndUpdateDinosaurAdmin.
// Check the length with:
//     len(mockedDinosaurService.VerifyAndUpdateDinosaurAdminCalls())
func (mock *DinosaurServiceMock) VerifyAndUpdateDinosaurAdminCalls() []struct {
	Ctx             context.Context
	DinosaurRequest *dbapi.DinosaurRequest
} {
	var calls []struct {
		Ctx             context.Context
		DinosaurRequest *dbapi.DinosaurRequest
	}
	mock.lockVerifyAndUpdateDinosaurAdmin.RLock()
	calls = mock.calls.VerifyAndUpdateDinosaurAdmin
	mock.lockVerifyAndUpdateDinosaurAdmin.RUnlock()
	return calls
}
