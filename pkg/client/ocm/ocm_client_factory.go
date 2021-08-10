package ocm

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/pkg/errors"
)

type ClientType string

const (
	ClusterManagementClientType ClientType = "ocm"
	AMSClientType               ClientType = "ams"
)

type OCMClientFactory struct {
	clientContainer     map[ClientType]Client
	connectionContainer map[ClientType]*sdk.Connection
}

func NewOcmClientProvider(config *OCMConfig) *OCMClientFactory {
	clientContainer := map[ClientType]Client{}
	connectionContainer := map[ClientType]*sdk.Connection{}

	amsConnection, _, err := NewOCMConnection(config, config.AmsUrl)
	if err != nil {
		logger.Logger.Error(errors.Wrap(err, "failed to create AMS client"))
	}

	connectionContainer[AMSClientType] = amsConnection
	clientContainer[AMSClientType] = NewClient(amsConnection)

	clusterManagerConnection, _, err := NewOCMConnection(config, config.BaseURL)
	if err != nil {
		logger.Logger.Error(err)
	}

	connectionContainer[ClusterManagementClientType] = clusterManagerConnection
	clientContainer[ClusterManagementClientType] = NewClient(clusterManagerConnection)

	return &OCMClientFactory{
		clientContainer:     clientContainer,
		connectionContainer: connectionContainer,
	}
}

func (p *OCMClientFactory) GetClient(clientType ClientType) Client {
	return p.clientContainer[clientType]
}

func (p *OCMClientFactory) GetConnection(clientType ClientType) *sdk.Connection {
	return p.connectionContainer[clientType]
}
