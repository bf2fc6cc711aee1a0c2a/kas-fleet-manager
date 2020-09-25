// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package ocm

import (
	"sync"

	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

// Ensure, that ClientMock does implement Client.
// If this is not the case, regenerate this file with moq.
var _ Client = &ClientMock{}

// ClientMock is a mock implementation of Client.
//
//     func TestSomethingThatUsesClient(t *testing.T) {
//
//         // make and configure a mocked Client
//         mockedClient := &ClientMock{
//             CreateClusterFunc: func(cluster *v1.Cluster) (*v1.Cluster, error) {
// 	               panic("mock out the CreateCluster method")
//             },
//         }
//
//         // use mockedClient in code that requires Client
//         // and then make assertions.
//
//     }
type ClientMock struct {
	// CreateClusterFunc mocks the CreateCluster method.
	CreateClusterFunc func(cluster *v1.Cluster) (*v1.Cluster, error)

	// calls tracks calls to the methods.
	calls struct {
		// CreateCluster holds details about calls to the CreateCluster method.
		CreateCluster []struct {
			// Cluster is the cluster argument value.
			Cluster *v1.Cluster
		}
	}
	lockCreateCluster sync.RWMutex
}

// CreateCluster calls CreateClusterFunc.
func (mock *ClientMock) CreateCluster(cluster *v1.Cluster) (*v1.Cluster, error) {
	if mock.CreateClusterFunc == nil {
		panic("ClientMock.CreateClusterFunc: method is nil but Client.CreateCluster was just called")
	}
	callInfo := struct {
		Cluster *v1.Cluster
	}{
		Cluster: cluster,
	}
	mock.lockCreateCluster.Lock()
	mock.calls.CreateCluster = append(mock.calls.CreateCluster, callInfo)
	mock.lockCreateCluster.Unlock()
	return mock.CreateClusterFunc(cluster)
}

// CreateClusterCalls gets all the calls that were made to CreateCluster.
// Check the length with:
//     len(mockedClient.CreateClusterCalls())
func (mock *ClientMock) CreateClusterCalls() []struct {
	Cluster *v1.Cluster
} {
	var calls []struct {
		Cluster *v1.Cluster
	}
	mock.lockCreateCluster.RLock()
	calls = mock.calls.CreateCluster
	mock.lockCreateCluster.RUnlock()
	return calls
}
