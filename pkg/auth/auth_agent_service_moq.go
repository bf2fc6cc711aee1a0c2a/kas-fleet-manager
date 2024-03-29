// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package auth

import (
	"sync"
)

// Ensure, that AuthAgentServiceMock does implement AuthAgentService.
// If this is not the case, regenerate this file with moq.
var _ AuthAgentService = &AuthAgentServiceMock{}

// AuthAgentServiceMock is a mock implementation of AuthAgentService.
//
//	func TestSomethingThatUsesAuthAgentService(t *testing.T) {
//
//		// make and configure a mocked AuthAgentService
//		mockedAuthAgentService := &AuthAgentServiceMock{
//			GetClientIDFunc: func(clusterID string) (string, error) {
//				panic("mock out the GetClientID method")
//			},
//		}
//
//		// use mockedAuthAgentService in code that requires AuthAgentService
//		// and then make assertions.
//
//	}
type AuthAgentServiceMock struct {
	// GetClientIDFunc mocks the GetClientID method.
	GetClientIDFunc func(clusterID string) (string, error)

	// calls tracks calls to the methods.
	calls struct {
		// GetClientID holds details about calls to the GetClientID method.
		GetClientID []struct {
			// ClusterID is the clusterID argument value.
			ClusterID string
		}
	}
	lockGetClientID sync.RWMutex
}

// GetClientID calls GetClientIDFunc.
func (mock *AuthAgentServiceMock) GetClientID(clusterID string) (string, error) {
	if mock.GetClientIDFunc == nil {
		panic("AuthAgentServiceMock.GetClientIDFunc: method is nil but AuthAgentService.GetClientID was just called")
	}
	callInfo := struct {
		ClusterID string
	}{
		ClusterID: clusterID,
	}
	mock.lockGetClientID.Lock()
	mock.calls.GetClientID = append(mock.calls.GetClientID, callInfo)
	mock.lockGetClientID.Unlock()
	return mock.GetClientIDFunc(clusterID)
}

// GetClientIDCalls gets all the calls that were made to GetClientID.
// Check the length with:
//
//	len(mockedAuthAgentService.GetClientIDCalls())
func (mock *AuthAgentServiceMock) GetClientIDCalls() []struct {
	ClusterID string
} {
	var calls []struct {
		ClusterID string
	}
	mock.lockGetClientID.RLock()
	calls = mock.calls.GetClientID
	mock.lockGetClientID.RUnlock()
	return calls
}
