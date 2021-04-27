// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package services

import (
	"context"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	managedkafka "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/managedkafkas.managedkafka.bf2.org/v1"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	apiErrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"sync"
)

// Ensure, that KafkaServiceMock does implement KafkaService.
// If this is not the case, regenerate this file with moq.
var _ KafkaService = &KafkaServiceMock{}

// KafkaServiceMock is a mock implementation of KafkaService.
//
// 	func TestSomethingThatUsesKafkaService(t *testing.T) {
//
// 		// make and configure a mocked KafkaService
// 		mockedKafkaService := &KafkaServiceMock{
// 			ChangeKafkaCNAMErecordsFunc: func(kafkaRequest *api.KafkaRequest, clusterDNS string, action string) (*route53.ChangeResourceRecordSetsOutput, *apiErrors.ServiceError) {
// 				panic("mock out the ChangeKafkaCNAMErecords method")
// 			},
// 			CountByStatusFunc: func(status []constants.KafkaStatus) ([]KafkaStatusCount, error) {
// 				panic("mock out the CountByStatus method")
// 			},
// 			CreateFunc: func(kafkaRequest *api.KafkaRequest) *apiErrors.ServiceError {
// 				panic("mock out the Create method")
// 			},
// 			DeleteFunc: func(kafkaRequest *api.KafkaRequest) *apiErrors.ServiceError {
// 				panic("mock out the Delete method")
// 			},
// 			DeprovisionExpiredKafkasFunc: func(kafkaAgeInHours int) *apiErrors.ServiceError {
// 				panic("mock out the DeprovisionExpiredKafkas method")
// 			},
// 			DeprovisionKafkaForUsersFunc: func(users []string) *apiErrors.ServiceError {
// 				panic("mock out the DeprovisionKafkaForUsers method")
// 			},
// 			GetFunc: func(ctx context.Context, id string) (*api.KafkaRequest, *apiErrors.ServiceError) {
// 				panic("mock out the Get method")
// 			},
// 			GetByIdFunc: func(id string) (*api.KafkaRequest, *apiErrors.ServiceError) {
// 				panic("mock out the GetById method")
// 			},
// 			GetManagedKafkaByClusterIDFunc: func(clusterID string) ([]managedkafka.ManagedKafka, *apiErrors.ServiceError) {
// 				panic("mock out the GetManagedKafkaByClusterID method")
// 			},
// 			HasAvailableCapacityFunc: func() (bool, *apiErrors.ServiceError) {
// 				panic("mock out the HasAvailableCapacity method")
// 			},
// 			ListFunc: func(ctx context.Context, listArgs *ListArguments) (api.KafkaList, *api.PagingMeta, *apiErrors.ServiceError) {
// 				panic("mock out the List method")
// 			},
// 			ListByStatusFunc: func(status ...constants.KafkaStatus) ([]*api.KafkaRequest, *apiErrors.ServiceError) {
// 				panic("mock out the ListByStatus method")
// 			},
// 			RegisterKafkaDeprovisionJobFunc: func(ctx context.Context, id string) *apiErrors.ServiceError {
// 				panic("mock out the RegisterKafkaDeprovisionJob method")
// 			},
// 			RegisterKafkaJobFunc: func(kafkaRequest *api.KafkaRequest) *apiErrors.ServiceError {
// 				panic("mock out the RegisterKafkaJob method")
// 			},
// 			UpdateFunc: func(kafkaRequest *api.KafkaRequest) *apiErrors.ServiceError {
// 				panic("mock out the Update method")
// 			},
// 			UpdateStatusFunc: func(id string, status constants.KafkaStatus) (bool, *apiErrors.ServiceError) {
// 				panic("mock out the UpdateStatus method")
// 			},
// 		}
//
// 		// use mockedKafkaService in code that requires KafkaService
// 		// and then make assertions.
//
// 	}
type KafkaServiceMock struct {
	// ChangeKafkaCNAMErecordsFunc mocks the ChangeKafkaCNAMErecords method.
	ChangeKafkaCNAMErecordsFunc func(kafkaRequest *api.KafkaRequest, clusterDNS string, action string) (*route53.ChangeResourceRecordSetsOutput, *apiErrors.ServiceError)

	// CountByStatusFunc mocks the CountByStatus method.
	CountByStatusFunc func(status []constants.KafkaStatus) ([]KafkaStatusCount, error)

	// CreateFunc mocks the Create method.
	CreateFunc func(kafkaRequest *api.KafkaRequest) *apiErrors.ServiceError

	// DeleteFunc mocks the Delete method.
	DeleteFunc func(kafkaRequest *api.KafkaRequest) *apiErrors.ServiceError

	// DeprovisionExpiredKafkasFunc mocks the DeprovisionExpiredKafkas method.
	DeprovisionExpiredKafkasFunc func(kafkaAgeInHours int) *apiErrors.ServiceError

	// DeprovisionKafkaForUsersFunc mocks the DeprovisionKafkaForUsers method.
	DeprovisionKafkaForUsersFunc func(users []string) *apiErrors.ServiceError

	// GetFunc mocks the Get method.
	GetFunc func(ctx context.Context, id string) (*api.KafkaRequest, *apiErrors.ServiceError)

	// GetByIdFunc mocks the GetById method.
	GetByIdFunc func(id string) (*api.KafkaRequest, *apiErrors.ServiceError)

	// GetManagedKafkaByClusterIDFunc mocks the GetManagedKafkaByClusterID method.
	GetManagedKafkaByClusterIDFunc func(clusterID string) ([]managedkafka.ManagedKafka, *apiErrors.ServiceError)

	// HasAvailableCapacityFunc mocks the HasAvailableCapacity method.
	HasAvailableCapacityFunc func() (bool, *apiErrors.ServiceError)

	// ListFunc mocks the List method.
	ListFunc func(ctx context.Context, listArgs *ListArguments) (api.KafkaList, *api.PagingMeta, *apiErrors.ServiceError)

	// ListByStatusFunc mocks the ListByStatus method.
	ListByStatusFunc func(status ...constants.KafkaStatus) ([]*api.KafkaRequest, *apiErrors.ServiceError)

	// RegisterKafkaDeprovisionJobFunc mocks the RegisterKafkaDeprovisionJob method.
	RegisterKafkaDeprovisionJobFunc func(ctx context.Context, id string) *apiErrors.ServiceError

	// RegisterKafkaJobFunc mocks the RegisterKafkaJob method.
	RegisterKafkaJobFunc func(kafkaRequest *api.KafkaRequest) *apiErrors.ServiceError

	// UpdateFunc mocks the Update method.
	UpdateFunc func(kafkaRequest *api.KafkaRequest) *apiErrors.ServiceError

	// UpdateStatusFunc mocks the UpdateStatus method.
	UpdateStatusFunc func(id string, status constants.KafkaStatus) (bool, *apiErrors.ServiceError)

	// calls tracks calls to the methods.
	calls struct {
		// ChangeKafkaCNAMErecords holds details about calls to the ChangeKafkaCNAMErecords method.
		ChangeKafkaCNAMErecords []struct {
			// KafkaRequest is the kafkaRequest argument value.
			KafkaRequest *api.KafkaRequest
			// ClusterDNS is the clusterDNS argument value.
			ClusterDNS string
			// Action is the action argument value.
			Action string
		}
		// CountByStatus holds details about calls to the CountByStatus method.
		CountByStatus []struct {
			// Status is the status argument value.
			Status []constants.KafkaStatus
		}
		// Create holds details about calls to the Create method.
		Create []struct {
			// KafkaRequest is the kafkaRequest argument value.
			KafkaRequest *api.KafkaRequest
		}
		// Delete holds details about calls to the Delete method.
		Delete []struct {
			// KafkaRequest is the kafkaRequest argument value.
			KafkaRequest *api.KafkaRequest
		}
		// DeprovisionExpiredKafkas holds details about calls to the DeprovisionExpiredKafkas method.
		DeprovisionExpiredKafkas []struct {
			// KafkaAgeInHours is the kafkaAgeInHours argument value.
			KafkaAgeInHours int
		}
		// DeprovisionKafkaForUsers holds details about calls to the DeprovisionKafkaForUsers method.
		DeprovisionKafkaForUsers []struct {
			// Users is the users argument value.
			Users []string
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
		// GetManagedKafkaByClusterID holds details about calls to the GetManagedKafkaByClusterID method.
		GetManagedKafkaByClusterID []struct {
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
			ListArgs *ListArguments
		}
		// ListByStatus holds details about calls to the ListByStatus method.
		ListByStatus []struct {
			// Status is the status argument value.
			Status []constants.KafkaStatus
		}
		// RegisterKafkaDeprovisionJob holds details about calls to the RegisterKafkaDeprovisionJob method.
		RegisterKafkaDeprovisionJob []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
		}
		// RegisterKafkaJob holds details about calls to the RegisterKafkaJob method.
		RegisterKafkaJob []struct {
			// KafkaRequest is the kafkaRequest argument value.
			KafkaRequest *api.KafkaRequest
		}
		// Update holds details about calls to the Update method.
		Update []struct {
			// KafkaRequest is the kafkaRequest argument value.
			KafkaRequest *api.KafkaRequest
		}
		// UpdateStatus holds details about calls to the UpdateStatus method.
		UpdateStatus []struct {
			// ID is the id argument value.
			ID string
			// Status is the status argument value.
			Status constants.KafkaStatus
		}
	}
	lockChangeKafkaCNAMErecords     sync.RWMutex
	lockCountByStatus               sync.RWMutex
	lockCreate                      sync.RWMutex
	lockDelete                      sync.RWMutex
	lockDeprovisionExpiredKafkas    sync.RWMutex
	lockDeprovisionKafkaForUsers    sync.RWMutex
	lockGet                         sync.RWMutex
	lockGetById                     sync.RWMutex
	lockGetManagedKafkaByClusterID  sync.RWMutex
	lockHasAvailableCapacity        sync.RWMutex
	lockList                        sync.RWMutex
	lockListByStatus                sync.RWMutex
	lockRegisterKafkaDeprovisionJob sync.RWMutex
	lockRegisterKafkaJob            sync.RWMutex
	lockUpdate                      sync.RWMutex
	lockUpdateStatus                sync.RWMutex
}

// ChangeKafkaCNAMErecords calls ChangeKafkaCNAMErecordsFunc.
func (mock *KafkaServiceMock) ChangeKafkaCNAMErecords(kafkaRequest *api.KafkaRequest, clusterDNS string, action string) (*route53.ChangeResourceRecordSetsOutput, *apiErrors.ServiceError) {
	if mock.ChangeKafkaCNAMErecordsFunc == nil {
		panic("KafkaServiceMock.ChangeKafkaCNAMErecordsFunc: method is nil but KafkaService.ChangeKafkaCNAMErecords was just called")
	}
	callInfo := struct {
		KafkaRequest *api.KafkaRequest
		ClusterDNS   string
		Action       string
	}{
		KafkaRequest: kafkaRequest,
		ClusterDNS:   clusterDNS,
		Action:       action,
	}
	mock.lockChangeKafkaCNAMErecords.Lock()
	mock.calls.ChangeKafkaCNAMErecords = append(mock.calls.ChangeKafkaCNAMErecords, callInfo)
	mock.lockChangeKafkaCNAMErecords.Unlock()
	return mock.ChangeKafkaCNAMErecordsFunc(kafkaRequest, clusterDNS, action)
}

// ChangeKafkaCNAMErecordsCalls gets all the calls that were made to ChangeKafkaCNAMErecords.
// Check the length with:
//     len(mockedKafkaService.ChangeKafkaCNAMErecordsCalls())
func (mock *KafkaServiceMock) ChangeKafkaCNAMErecordsCalls() []struct {
	KafkaRequest *api.KafkaRequest
	ClusterDNS   string
	Action       string
} {
	var calls []struct {
		KafkaRequest *api.KafkaRequest
		ClusterDNS   string
		Action       string
	}
	mock.lockChangeKafkaCNAMErecords.RLock()
	calls = mock.calls.ChangeKafkaCNAMErecords
	mock.lockChangeKafkaCNAMErecords.RUnlock()
	return calls
}

// CountByStatus calls CountByStatusFunc.
func (mock *KafkaServiceMock) CountByStatus(status []constants.KafkaStatus) ([]KafkaStatusCount, error) {
	if mock.CountByStatusFunc == nil {
		panic("KafkaServiceMock.CountByStatusFunc: method is nil but KafkaService.CountByStatus was just called")
	}
	callInfo := struct {
		Status []constants.KafkaStatus
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
//     len(mockedKafkaService.CountByStatusCalls())
func (mock *KafkaServiceMock) CountByStatusCalls() []struct {
	Status []constants.KafkaStatus
} {
	var calls []struct {
		Status []constants.KafkaStatus
	}
	mock.lockCountByStatus.RLock()
	calls = mock.calls.CountByStatus
	mock.lockCountByStatus.RUnlock()
	return calls
}

// Create calls CreateFunc.
func (mock *KafkaServiceMock) Create(kafkaRequest *api.KafkaRequest) *apiErrors.ServiceError {
	if mock.CreateFunc == nil {
		panic("KafkaServiceMock.CreateFunc: method is nil but KafkaService.Create was just called")
	}
	callInfo := struct {
		KafkaRequest *api.KafkaRequest
	}{
		KafkaRequest: kafkaRequest,
	}
	mock.lockCreate.Lock()
	mock.calls.Create = append(mock.calls.Create, callInfo)
	mock.lockCreate.Unlock()
	return mock.CreateFunc(kafkaRequest)
}

// CreateCalls gets all the calls that were made to Create.
// Check the length with:
//     len(mockedKafkaService.CreateCalls())
func (mock *KafkaServiceMock) CreateCalls() []struct {
	KafkaRequest *api.KafkaRequest
} {
	var calls []struct {
		KafkaRequest *api.KafkaRequest
	}
	mock.lockCreate.RLock()
	calls = mock.calls.Create
	mock.lockCreate.RUnlock()
	return calls
}

// Delete calls DeleteFunc.
func (mock *KafkaServiceMock) Delete(kafkaRequest *api.KafkaRequest) *apiErrors.ServiceError {
	if mock.DeleteFunc == nil {
		panic("KafkaServiceMock.DeleteFunc: method is nil but KafkaService.Delete was just called")
	}
	callInfo := struct {
		KafkaRequest *api.KafkaRequest
	}{
		KafkaRequest: kafkaRequest,
	}
	mock.lockDelete.Lock()
	mock.calls.Delete = append(mock.calls.Delete, callInfo)
	mock.lockDelete.Unlock()
	return mock.DeleteFunc(kafkaRequest)
}

// DeleteCalls gets all the calls that were made to Delete.
// Check the length with:
//     len(mockedKafkaService.DeleteCalls())
func (mock *KafkaServiceMock) DeleteCalls() []struct {
	KafkaRequest *api.KafkaRequest
} {
	var calls []struct {
		KafkaRequest *api.KafkaRequest
	}
	mock.lockDelete.RLock()
	calls = mock.calls.Delete
	mock.lockDelete.RUnlock()
	return calls
}

// DeprovisionExpiredKafkas calls DeprovisionExpiredKafkasFunc.
func (mock *KafkaServiceMock) DeprovisionExpiredKafkas(kafkaAgeInHours int) *apiErrors.ServiceError {
	if mock.DeprovisionExpiredKafkasFunc == nil {
		panic("KafkaServiceMock.DeprovisionExpiredKafkasFunc: method is nil but KafkaService.DeprovisionExpiredKafkas was just called")
	}
	callInfo := struct {
		KafkaAgeInHours int
	}{
		KafkaAgeInHours: kafkaAgeInHours,
	}
	mock.lockDeprovisionExpiredKafkas.Lock()
	mock.calls.DeprovisionExpiredKafkas = append(mock.calls.DeprovisionExpiredKafkas, callInfo)
	mock.lockDeprovisionExpiredKafkas.Unlock()
	return mock.DeprovisionExpiredKafkasFunc(kafkaAgeInHours)
}

// DeprovisionExpiredKafkasCalls gets all the calls that were made to DeprovisionExpiredKafkas.
// Check the length with:
//     len(mockedKafkaService.DeprovisionExpiredKafkasCalls())
func (mock *KafkaServiceMock) DeprovisionExpiredKafkasCalls() []struct {
	KafkaAgeInHours int
} {
	var calls []struct {
		KafkaAgeInHours int
	}
	mock.lockDeprovisionExpiredKafkas.RLock()
	calls = mock.calls.DeprovisionExpiredKafkas
	mock.lockDeprovisionExpiredKafkas.RUnlock()
	return calls
}

// DeprovisionKafkaForUsers calls DeprovisionKafkaForUsersFunc.
func (mock *KafkaServiceMock) DeprovisionKafkaForUsers(users []string) *apiErrors.ServiceError {
	if mock.DeprovisionKafkaForUsersFunc == nil {
		panic("KafkaServiceMock.DeprovisionKafkaForUsersFunc: method is nil but KafkaService.DeprovisionKafkaForUsers was just called")
	}
	callInfo := struct {
		Users []string
	}{
		Users: users,
	}
	mock.lockDeprovisionKafkaForUsers.Lock()
	mock.calls.DeprovisionKafkaForUsers = append(mock.calls.DeprovisionKafkaForUsers, callInfo)
	mock.lockDeprovisionKafkaForUsers.Unlock()
	return mock.DeprovisionKafkaForUsersFunc(users)
}

// DeprovisionKafkaForUsersCalls gets all the calls that were made to DeprovisionKafkaForUsers.
// Check the length with:
//     len(mockedKafkaService.DeprovisionKafkaForUsersCalls())
func (mock *KafkaServiceMock) DeprovisionKafkaForUsersCalls() []struct {
	Users []string
} {
	var calls []struct {
		Users []string
	}
	mock.lockDeprovisionKafkaForUsers.RLock()
	calls = mock.calls.DeprovisionKafkaForUsers
	mock.lockDeprovisionKafkaForUsers.RUnlock()
	return calls
}

// Get calls GetFunc.
func (mock *KafkaServiceMock) Get(ctx context.Context, id string) (*api.KafkaRequest, *apiErrors.ServiceError) {
	if mock.GetFunc == nil {
		panic("KafkaServiceMock.GetFunc: method is nil but KafkaService.Get was just called")
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
//     len(mockedKafkaService.GetCalls())
func (mock *KafkaServiceMock) GetCalls() []struct {
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
func (mock *KafkaServiceMock) GetById(id string) (*api.KafkaRequest, *apiErrors.ServiceError) {
	if mock.GetByIdFunc == nil {
		panic("KafkaServiceMock.GetByIdFunc: method is nil but KafkaService.GetById was just called")
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
//     len(mockedKafkaService.GetByIdCalls())
func (mock *KafkaServiceMock) GetByIdCalls() []struct {
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

// GetManagedKafkaByClusterID calls GetManagedKafkaByClusterIDFunc.
func (mock *KafkaServiceMock) GetManagedKafkaByClusterID(clusterID string) ([]managedkafka.ManagedKafka, *apiErrors.ServiceError) {
	if mock.GetManagedKafkaByClusterIDFunc == nil {
		panic("KafkaServiceMock.GetManagedKafkaByClusterIDFunc: method is nil but KafkaService.GetManagedKafkaByClusterID was just called")
	}
	callInfo := struct {
		ClusterID string
	}{
		ClusterID: clusterID,
	}
	mock.lockGetManagedKafkaByClusterID.Lock()
	mock.calls.GetManagedKafkaByClusterID = append(mock.calls.GetManagedKafkaByClusterID, callInfo)
	mock.lockGetManagedKafkaByClusterID.Unlock()
	return mock.GetManagedKafkaByClusterIDFunc(clusterID)
}

// GetManagedKafkaByClusterIDCalls gets all the calls that were made to GetManagedKafkaByClusterID.
// Check the length with:
//     len(mockedKafkaService.GetManagedKafkaByClusterIDCalls())
func (mock *KafkaServiceMock) GetManagedKafkaByClusterIDCalls() []struct {
	ClusterID string
} {
	var calls []struct {
		ClusterID string
	}
	mock.lockGetManagedKafkaByClusterID.RLock()
	calls = mock.calls.GetManagedKafkaByClusterID
	mock.lockGetManagedKafkaByClusterID.RUnlock()
	return calls
}

// HasAvailableCapacity calls HasAvailableCapacityFunc.
func (mock *KafkaServiceMock) HasAvailableCapacity() (bool, *apiErrors.ServiceError) {
	if mock.HasAvailableCapacityFunc == nil {
		panic("KafkaServiceMock.HasAvailableCapacityFunc: method is nil but KafkaService.HasAvailableCapacity was just called")
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
//     len(mockedKafkaService.HasAvailableCapacityCalls())
func (mock *KafkaServiceMock) HasAvailableCapacityCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockHasAvailableCapacity.RLock()
	calls = mock.calls.HasAvailableCapacity
	mock.lockHasAvailableCapacity.RUnlock()
	return calls
}

// List calls ListFunc.
func (mock *KafkaServiceMock) List(ctx context.Context, listArgs *ListArguments) (api.KafkaList, *api.PagingMeta, *apiErrors.ServiceError) {
	if mock.ListFunc == nil {
		panic("KafkaServiceMock.ListFunc: method is nil but KafkaService.List was just called")
	}
	callInfo := struct {
		Ctx      context.Context
		ListArgs *ListArguments
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
//     len(mockedKafkaService.ListCalls())
func (mock *KafkaServiceMock) ListCalls() []struct {
	Ctx      context.Context
	ListArgs *ListArguments
} {
	var calls []struct {
		Ctx      context.Context
		ListArgs *ListArguments
	}
	mock.lockList.RLock()
	calls = mock.calls.List
	mock.lockList.RUnlock()
	return calls
}

// ListByStatus calls ListByStatusFunc.
func (mock *KafkaServiceMock) ListByStatus(status ...constants.KafkaStatus) ([]*api.KafkaRequest, *apiErrors.ServiceError) {
	if mock.ListByStatusFunc == nil {
		panic("KafkaServiceMock.ListByStatusFunc: method is nil but KafkaService.ListByStatus was just called")
	}
	callInfo := struct {
		Status []constants.KafkaStatus
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
//     len(mockedKafkaService.ListByStatusCalls())
func (mock *KafkaServiceMock) ListByStatusCalls() []struct {
	Status []constants.KafkaStatus
} {
	var calls []struct {
		Status []constants.KafkaStatus
	}
	mock.lockListByStatus.RLock()
	calls = mock.calls.ListByStatus
	mock.lockListByStatus.RUnlock()
	return calls
}

// RegisterKafkaDeprovisionJob calls RegisterKafkaDeprovisionJobFunc.
func (mock *KafkaServiceMock) RegisterKafkaDeprovisionJob(ctx context.Context, id string) *apiErrors.ServiceError {
	if mock.RegisterKafkaDeprovisionJobFunc == nil {
		panic("KafkaServiceMock.RegisterKafkaDeprovisionJobFunc: method is nil but KafkaService.RegisterKafkaDeprovisionJob was just called")
	}
	callInfo := struct {
		Ctx context.Context
		ID  string
	}{
		Ctx: ctx,
		ID:  id,
	}
	mock.lockRegisterKafkaDeprovisionJob.Lock()
	mock.calls.RegisterKafkaDeprovisionJob = append(mock.calls.RegisterKafkaDeprovisionJob, callInfo)
	mock.lockRegisterKafkaDeprovisionJob.Unlock()
	return mock.RegisterKafkaDeprovisionJobFunc(ctx, id)
}

// RegisterKafkaDeprovisionJobCalls gets all the calls that were made to RegisterKafkaDeprovisionJob.
// Check the length with:
//     len(mockedKafkaService.RegisterKafkaDeprovisionJobCalls())
func (mock *KafkaServiceMock) RegisterKafkaDeprovisionJobCalls() []struct {
	Ctx context.Context
	ID  string
} {
	var calls []struct {
		Ctx context.Context
		ID  string
	}
	mock.lockRegisterKafkaDeprovisionJob.RLock()
	calls = mock.calls.RegisterKafkaDeprovisionJob
	mock.lockRegisterKafkaDeprovisionJob.RUnlock()
	return calls
}

// RegisterKafkaJob calls RegisterKafkaJobFunc.
func (mock *KafkaServiceMock) RegisterKafkaJob(kafkaRequest *api.KafkaRequest) *apiErrors.ServiceError {
	if mock.RegisterKafkaJobFunc == nil {
		panic("KafkaServiceMock.RegisterKafkaJobFunc: method is nil but KafkaService.RegisterKafkaJob was just called")
	}
	callInfo := struct {
		KafkaRequest *api.KafkaRequest
	}{
		KafkaRequest: kafkaRequest,
	}
	mock.lockRegisterKafkaJob.Lock()
	mock.calls.RegisterKafkaJob = append(mock.calls.RegisterKafkaJob, callInfo)
	mock.lockRegisterKafkaJob.Unlock()
	return mock.RegisterKafkaJobFunc(kafkaRequest)
}

// RegisterKafkaJobCalls gets all the calls that were made to RegisterKafkaJob.
// Check the length with:
//     len(mockedKafkaService.RegisterKafkaJobCalls())
func (mock *KafkaServiceMock) RegisterKafkaJobCalls() []struct {
	KafkaRequest *api.KafkaRequest
} {
	var calls []struct {
		KafkaRequest *api.KafkaRequest
	}
	mock.lockRegisterKafkaJob.RLock()
	calls = mock.calls.RegisterKafkaJob
	mock.lockRegisterKafkaJob.RUnlock()
	return calls
}

// Update calls UpdateFunc.
func (mock *KafkaServiceMock) Update(kafkaRequest *api.KafkaRequest) *apiErrors.ServiceError {
	if mock.UpdateFunc == nil {
		panic("KafkaServiceMock.UpdateFunc: method is nil but KafkaService.Update was just called")
	}
	callInfo := struct {
		KafkaRequest *api.KafkaRequest
	}{
		KafkaRequest: kafkaRequest,
	}
	mock.lockUpdate.Lock()
	mock.calls.Update = append(mock.calls.Update, callInfo)
	mock.lockUpdate.Unlock()
	return mock.UpdateFunc(kafkaRequest)
}

// UpdateCalls gets all the calls that were made to Update.
// Check the length with:
//     len(mockedKafkaService.UpdateCalls())
func (mock *KafkaServiceMock) UpdateCalls() []struct {
	KafkaRequest *api.KafkaRequest
} {
	var calls []struct {
		KafkaRequest *api.KafkaRequest
	}
	mock.lockUpdate.RLock()
	calls = mock.calls.Update
	mock.lockUpdate.RUnlock()
	return calls
}

// UpdateStatus calls UpdateStatusFunc.
func (mock *KafkaServiceMock) UpdateStatus(id string, status constants.KafkaStatus) (bool, *apiErrors.ServiceError) {
	if mock.UpdateStatusFunc == nil {
		panic("KafkaServiceMock.UpdateStatusFunc: method is nil but KafkaService.UpdateStatus was just called")
	}
	callInfo := struct {
		ID     string
		Status constants.KafkaStatus
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
//     len(mockedKafkaService.UpdateStatusCalls())
func (mock *KafkaServiceMock) UpdateStatusCalls() []struct {
	ID     string
	Status constants.KafkaStatus
} {
	var calls []struct {
		ID     string
		Status constants.KafkaStatus
	}
	mock.lockUpdateStatus.RLock()
	calls = mock.calls.UpdateStatus
	mock.lockUpdateStatus.RUnlock()
	return calls
}
