// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package redhatsso

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	serviceaccountsclient "github.com/redhat-developer/app-services-sdk-core/app-services-sdk-go/serviceaccountmgmt/apiv1/client"
	"sync"
)

// Ensure, that SSOClientMock does implement SSOClient.
// If this is not the case, regenerate this file with moq.
var _ SSOClient = &SSOClientMock{}

// SSOClientMock is a mock implementation of SSOClient.
//
//	func TestSomethingThatUsesSSOClient(t *testing.T) {
//
//		// make and configure a mocked SSOClient
//		mockedSSOClient := &SSOClientMock{
//			CreateServiceAccountFunc: func(accessToken string, name string, description string) (serviceaccountsclient.ServiceAccountData, error) {
//				panic("mock out the CreateServiceAccount method")
//			},
//			DeleteServiceAccountFunc: func(accessToken string, clientId string) error {
//				panic("mock out the DeleteServiceAccount method")
//			},
//			GetConfigFunc: func() *keycloak.KeycloakConfig {
//				panic("mock out the GetConfig method")
//			},
//			GetRealmConfigFunc: func() *keycloak.KeycloakRealmConfig {
//				panic("mock out the GetRealmConfig method")
//			},
//			GetServiceAccountFunc: func(accessToken string, clientId string) (*serviceaccountsclient.ServiceAccountData, bool, error) {
//				panic("mock out the GetServiceAccount method")
//			},
//			GetServiceAccountsFunc: func(accessToken string, first int, max int) ([]serviceaccountsclient.ServiceAccountData, error) {
//				panic("mock out the GetServiceAccounts method")
//			},
//			GetTokenFunc: func() (string, error) {
//				panic("mock out the GetToken method")
//			},
//			RegenerateClientSecretFunc: func(accessToken string, id string) (serviceaccountsclient.ServiceAccountData, error) {
//				panic("mock out the RegenerateClientSecret method")
//			},
//			UpdateServiceAccountFunc: func(accessToken string, clientId string, name string, description string) (serviceaccountsclient.ServiceAccountData, error) {
//				panic("mock out the UpdateServiceAccount method")
//			},
//		}
//
//		// use mockedSSOClient in code that requires SSOClient
//		// and then make assertions.
//
//	}
type SSOClientMock struct {
	// CreateServiceAccountFunc mocks the CreateServiceAccount method.
	CreateServiceAccountFunc func(accessToken string, name string, description string) (serviceaccountsclient.ServiceAccountData, error)

	// DeleteServiceAccountFunc mocks the DeleteServiceAccount method.
	DeleteServiceAccountFunc func(accessToken string, clientId string) error

	// GetConfigFunc mocks the GetConfig method.
	GetConfigFunc func() *keycloak.KeycloakConfig

	// GetRealmConfigFunc mocks the GetRealmConfig method.
	GetRealmConfigFunc func() *keycloak.KeycloakRealmConfig

	// GetServiceAccountFunc mocks the GetServiceAccount method.
	GetServiceAccountFunc func(accessToken string, clientId string) (*serviceaccountsclient.ServiceAccountData, bool, error)

	// GetServiceAccountsFunc mocks the GetServiceAccounts method.
	GetServiceAccountsFunc func(accessToken string, first int, max int) ([]serviceaccountsclient.ServiceAccountData, error)

	// GetTokenFunc mocks the GetToken method.
	GetTokenFunc func() (string, error)

	// RegenerateClientSecretFunc mocks the RegenerateClientSecret method.
	RegenerateClientSecretFunc func(accessToken string, id string) (serviceaccountsclient.ServiceAccountData, error)

	// UpdateServiceAccountFunc mocks the UpdateServiceAccount method.
	UpdateServiceAccountFunc func(accessToken string, clientId string, name string, description string) (serviceaccountsclient.ServiceAccountData, error)

	// calls tracks calls to the methods.
	calls struct {
		// CreateServiceAccount holds details about calls to the CreateServiceAccount method.
		CreateServiceAccount []struct {
			// AccessToken is the accessToken argument value.
			AccessToken string
			// Name is the name argument value.
			Name string
			// Description is the description argument value.
			Description string
		}
		// DeleteServiceAccount holds details about calls to the DeleteServiceAccount method.
		DeleteServiceAccount []struct {
			// AccessToken is the accessToken argument value.
			AccessToken string
			// ClientId is the clientId argument value.
			ClientId string
		}
		// GetConfig holds details about calls to the GetConfig method.
		GetConfig []struct {
		}
		// GetRealmConfig holds details about calls to the GetRealmConfig method.
		GetRealmConfig []struct {
		}
		// GetServiceAccount holds details about calls to the GetServiceAccount method.
		GetServiceAccount []struct {
			// AccessToken is the accessToken argument value.
			AccessToken string
			// ClientId is the clientId argument value.
			ClientId string
		}
		// GetServiceAccounts holds details about calls to the GetServiceAccounts method.
		GetServiceAccounts []struct {
			// AccessToken is the accessToken argument value.
			AccessToken string
			// First is the first argument value.
			First int
			// Max is the max argument value.
			Max int
		}
		// GetToken holds details about calls to the GetToken method.
		GetToken []struct {
		}
		// RegenerateClientSecret holds details about calls to the RegenerateClientSecret method.
		RegenerateClientSecret []struct {
			// AccessToken is the accessToken argument value.
			AccessToken string
			// ID is the id argument value.
			ID string
		}
		// UpdateServiceAccount holds details about calls to the UpdateServiceAccount method.
		UpdateServiceAccount []struct {
			// AccessToken is the accessToken argument value.
			AccessToken string
			// ClientId is the clientId argument value.
			ClientId string
			// Name is the name argument value.
			Name string
			// Description is the description argument value.
			Description string
		}
	}
	lockCreateServiceAccount   sync.RWMutex
	lockDeleteServiceAccount   sync.RWMutex
	lockGetConfig              sync.RWMutex
	lockGetRealmConfig         sync.RWMutex
	lockGetServiceAccount      sync.RWMutex
	lockGetServiceAccounts     sync.RWMutex
	lockGetToken               sync.RWMutex
	lockRegenerateClientSecret sync.RWMutex
	lockUpdateServiceAccount   sync.RWMutex
}

// CreateServiceAccount calls CreateServiceAccountFunc.
func (mock *SSOClientMock) CreateServiceAccount(accessToken string, name string, description string) (serviceaccountsclient.ServiceAccountData, error) {
	if mock.CreateServiceAccountFunc == nil {
		panic("SSOClientMock.CreateServiceAccountFunc: method is nil but SSOClient.CreateServiceAccount was just called")
	}
	callInfo := struct {
		AccessToken string
		Name        string
		Description string
	}{
		AccessToken: accessToken,
		Name:        name,
		Description: description,
	}
	mock.lockCreateServiceAccount.Lock()
	mock.calls.CreateServiceAccount = append(mock.calls.CreateServiceAccount, callInfo)
	mock.lockCreateServiceAccount.Unlock()
	return mock.CreateServiceAccountFunc(accessToken, name, description)
}

// CreateServiceAccountCalls gets all the calls that were made to CreateServiceAccount.
// Check the length with:
//
//	len(mockedSSOClient.CreateServiceAccountCalls())
func (mock *SSOClientMock) CreateServiceAccountCalls() []struct {
	AccessToken string
	Name        string
	Description string
} {
	var calls []struct {
		AccessToken string
		Name        string
		Description string
	}
	mock.lockCreateServiceAccount.RLock()
	calls = mock.calls.CreateServiceAccount
	mock.lockCreateServiceAccount.RUnlock()
	return calls
}

// DeleteServiceAccount calls DeleteServiceAccountFunc.
func (mock *SSOClientMock) DeleteServiceAccount(accessToken string, clientId string) error {
	if mock.DeleteServiceAccountFunc == nil {
		panic("SSOClientMock.DeleteServiceAccountFunc: method is nil but SSOClient.DeleteServiceAccount was just called")
	}
	callInfo := struct {
		AccessToken string
		ClientId    string
	}{
		AccessToken: accessToken,
		ClientId:    clientId,
	}
	mock.lockDeleteServiceAccount.Lock()
	mock.calls.DeleteServiceAccount = append(mock.calls.DeleteServiceAccount, callInfo)
	mock.lockDeleteServiceAccount.Unlock()
	return mock.DeleteServiceAccountFunc(accessToken, clientId)
}

// DeleteServiceAccountCalls gets all the calls that were made to DeleteServiceAccount.
// Check the length with:
//
//	len(mockedSSOClient.DeleteServiceAccountCalls())
func (mock *SSOClientMock) DeleteServiceAccountCalls() []struct {
	AccessToken string
	ClientId    string
} {
	var calls []struct {
		AccessToken string
		ClientId    string
	}
	mock.lockDeleteServiceAccount.RLock()
	calls = mock.calls.DeleteServiceAccount
	mock.lockDeleteServiceAccount.RUnlock()
	return calls
}

// GetConfig calls GetConfigFunc.
func (mock *SSOClientMock) GetConfig() *keycloak.KeycloakConfig {
	if mock.GetConfigFunc == nil {
		panic("SSOClientMock.GetConfigFunc: method is nil but SSOClient.GetConfig was just called")
	}
	callInfo := struct {
	}{}
	mock.lockGetConfig.Lock()
	mock.calls.GetConfig = append(mock.calls.GetConfig, callInfo)
	mock.lockGetConfig.Unlock()
	return mock.GetConfigFunc()
}

// GetConfigCalls gets all the calls that were made to GetConfig.
// Check the length with:
//
//	len(mockedSSOClient.GetConfigCalls())
func (mock *SSOClientMock) GetConfigCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockGetConfig.RLock()
	calls = mock.calls.GetConfig
	mock.lockGetConfig.RUnlock()
	return calls
}

// GetRealmConfig calls GetRealmConfigFunc.
func (mock *SSOClientMock) GetRealmConfig() *keycloak.KeycloakRealmConfig {
	if mock.GetRealmConfigFunc == nil {
		panic("SSOClientMock.GetRealmConfigFunc: method is nil but SSOClient.GetRealmConfig was just called")
	}
	callInfo := struct {
	}{}
	mock.lockGetRealmConfig.Lock()
	mock.calls.GetRealmConfig = append(mock.calls.GetRealmConfig, callInfo)
	mock.lockGetRealmConfig.Unlock()
	return mock.GetRealmConfigFunc()
}

// GetRealmConfigCalls gets all the calls that were made to GetRealmConfig.
// Check the length with:
//
//	len(mockedSSOClient.GetRealmConfigCalls())
func (mock *SSOClientMock) GetRealmConfigCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockGetRealmConfig.RLock()
	calls = mock.calls.GetRealmConfig
	mock.lockGetRealmConfig.RUnlock()
	return calls
}

// GetServiceAccount calls GetServiceAccountFunc.
func (mock *SSOClientMock) GetServiceAccount(accessToken string, clientId string) (*serviceaccountsclient.ServiceAccountData, bool, error) {
	if mock.GetServiceAccountFunc == nil {
		panic("SSOClientMock.GetServiceAccountFunc: method is nil but SSOClient.GetServiceAccount was just called")
	}
	callInfo := struct {
		AccessToken string
		ClientId    string
	}{
		AccessToken: accessToken,
		ClientId:    clientId,
	}
	mock.lockGetServiceAccount.Lock()
	mock.calls.GetServiceAccount = append(mock.calls.GetServiceAccount, callInfo)
	mock.lockGetServiceAccount.Unlock()
	return mock.GetServiceAccountFunc(accessToken, clientId)
}

// GetServiceAccountCalls gets all the calls that were made to GetServiceAccount.
// Check the length with:
//
//	len(mockedSSOClient.GetServiceAccountCalls())
func (mock *SSOClientMock) GetServiceAccountCalls() []struct {
	AccessToken string
	ClientId    string
} {
	var calls []struct {
		AccessToken string
		ClientId    string
	}
	mock.lockGetServiceAccount.RLock()
	calls = mock.calls.GetServiceAccount
	mock.lockGetServiceAccount.RUnlock()
	return calls
}

// GetServiceAccounts calls GetServiceAccountsFunc.
func (mock *SSOClientMock) GetServiceAccounts(accessToken string, first int, max int) ([]serviceaccountsclient.ServiceAccountData, error) {
	if mock.GetServiceAccountsFunc == nil {
		panic("SSOClientMock.GetServiceAccountsFunc: method is nil but SSOClient.GetServiceAccounts was just called")
	}
	callInfo := struct {
		AccessToken string
		First       int
		Max         int
	}{
		AccessToken: accessToken,
		First:       first,
		Max:         max,
	}
	mock.lockGetServiceAccounts.Lock()
	mock.calls.GetServiceAccounts = append(mock.calls.GetServiceAccounts, callInfo)
	mock.lockGetServiceAccounts.Unlock()
	return mock.GetServiceAccountsFunc(accessToken, first, max)
}

// GetServiceAccountsCalls gets all the calls that were made to GetServiceAccounts.
// Check the length with:
//
//	len(mockedSSOClient.GetServiceAccountsCalls())
func (mock *SSOClientMock) GetServiceAccountsCalls() []struct {
	AccessToken string
	First       int
	Max         int
} {
	var calls []struct {
		AccessToken string
		First       int
		Max         int
	}
	mock.lockGetServiceAccounts.RLock()
	calls = mock.calls.GetServiceAccounts
	mock.lockGetServiceAccounts.RUnlock()
	return calls
}

// GetToken calls GetTokenFunc.
func (mock *SSOClientMock) GetToken() (string, error) {
	if mock.GetTokenFunc == nil {
		panic("SSOClientMock.GetTokenFunc: method is nil but SSOClient.GetToken was just called")
	}
	callInfo := struct {
	}{}
	mock.lockGetToken.Lock()
	mock.calls.GetToken = append(mock.calls.GetToken, callInfo)
	mock.lockGetToken.Unlock()
	return mock.GetTokenFunc()
}

// GetTokenCalls gets all the calls that were made to GetToken.
// Check the length with:
//
//	len(mockedSSOClient.GetTokenCalls())
func (mock *SSOClientMock) GetTokenCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockGetToken.RLock()
	calls = mock.calls.GetToken
	mock.lockGetToken.RUnlock()
	return calls
}

// RegenerateClientSecret calls RegenerateClientSecretFunc.
func (mock *SSOClientMock) RegenerateClientSecret(accessToken string, id string) (serviceaccountsclient.ServiceAccountData, error) {
	if mock.RegenerateClientSecretFunc == nil {
		panic("SSOClientMock.RegenerateClientSecretFunc: method is nil but SSOClient.RegenerateClientSecret was just called")
	}
	callInfo := struct {
		AccessToken string
		ID          string
	}{
		AccessToken: accessToken,
		ID:          id,
	}
	mock.lockRegenerateClientSecret.Lock()
	mock.calls.RegenerateClientSecret = append(mock.calls.RegenerateClientSecret, callInfo)
	mock.lockRegenerateClientSecret.Unlock()
	return mock.RegenerateClientSecretFunc(accessToken, id)
}

// RegenerateClientSecretCalls gets all the calls that were made to RegenerateClientSecret.
// Check the length with:
//
//	len(mockedSSOClient.RegenerateClientSecretCalls())
func (mock *SSOClientMock) RegenerateClientSecretCalls() []struct {
	AccessToken string
	ID          string
} {
	var calls []struct {
		AccessToken string
		ID          string
	}
	mock.lockRegenerateClientSecret.RLock()
	calls = mock.calls.RegenerateClientSecret
	mock.lockRegenerateClientSecret.RUnlock()
	return calls
}

// UpdateServiceAccount calls UpdateServiceAccountFunc.
func (mock *SSOClientMock) UpdateServiceAccount(accessToken string, clientId string, name string, description string) (serviceaccountsclient.ServiceAccountData, error) {
	if mock.UpdateServiceAccountFunc == nil {
		panic("SSOClientMock.UpdateServiceAccountFunc: method is nil but SSOClient.UpdateServiceAccount was just called")
	}
	callInfo := struct {
		AccessToken string
		ClientId    string
		Name        string
		Description string
	}{
		AccessToken: accessToken,
		ClientId:    clientId,
		Name:        name,
		Description: description,
	}
	mock.lockUpdateServiceAccount.Lock()
	mock.calls.UpdateServiceAccount = append(mock.calls.UpdateServiceAccount, callInfo)
	mock.lockUpdateServiceAccount.Unlock()
	return mock.UpdateServiceAccountFunc(accessToken, clientId, name, description)
}

// UpdateServiceAccountCalls gets all the calls that were made to UpdateServiceAccount.
// Check the length with:
//
//	len(mockedSSOClient.UpdateServiceAccountCalls())
func (mock *SSOClientMock) UpdateServiceAccountCalls() []struct {
	AccessToken string
	ClientId    string
	Name        string
	Description string
} {
	var calls []struct {
		AccessToken string
		ClientId    string
		Name        string
		Description string
	}
	mock.lockUpdateServiceAccount.RLock()
	calls = mock.calls.UpdateServiceAccount
	mock.lockUpdateServiceAccount.RUnlock()
	return calls
}
