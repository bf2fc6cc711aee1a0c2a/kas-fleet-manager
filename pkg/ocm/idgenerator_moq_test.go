// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package ocm

import (
	"sync"
)

// Ensure, that IDGeneratorMock does implement IDGenerator.
// If this is not the case, regenerate this file with moq.
var _ IDGenerator = &IDGeneratorMock{}

// IDGeneratorMock is a mock implementation of IDGenerator.
//
// 	func TestSomethingThatUsesIDGenerator(t *testing.T) {
//
// 		// make and configure a mocked IDGenerator
// 		mockedIDGenerator := &IDGeneratorMock{
// 			GenerateFunc: func() string {
// 				panic("mock out the Generate method")
// 			},
// 		}
//
// 		// use mockedIDGenerator in code that requires IDGenerator
// 		// and then make assertions.
//
// 	}
type IDGeneratorMock struct {
	// GenerateFunc mocks the Generate method.
	GenerateFunc func() string

	// calls tracks calls to the methods.
	calls struct {
		// Generate holds details about calls to the Generate method.
		Generate []struct {
		}
	}
	lockGenerate sync.RWMutex
}

// Generate calls GenerateFunc.
func (mock *IDGeneratorMock) Generate() string {
	if mock.GenerateFunc == nil {
		panic("IDGeneratorMock.GenerateFunc: method is nil but IDGenerator.Generate was just called")
	}
	callInfo := struct {
	}{}
	mock.lockGenerate.Lock()
	mock.calls.Generate = append(mock.calls.Generate, callInfo)
	mock.lockGenerate.Unlock()
	return mock.GenerateFunc()
}

// GenerateCalls gets all the calls that were made to Generate.
// Check the length with:
//     len(mockedIDGenerator.GenerateCalls())
func (mock *IDGeneratorMock) GenerateCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockGenerate.RLock()
	calls = mock.calls.Generate
	mock.lockGenerate.RUnlock()
	return calls
}
