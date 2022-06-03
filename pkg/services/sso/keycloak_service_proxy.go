package sso

import (
	"context"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/golang/glog"
	"github.com/openshift-online/ocm-sdk-go/authentication"
)

type tokenProvider interface {
	GetToken() (string, error)
}

type keycloakServiceProxy struct {
	accessTokenProvider tokenProvider
	service             keycloakServiceInternal
}

var _ KeycloakService = &keycloakServiceProxy{}
var _ OSDKeycloakService = &keycloakServiceProxy{}

func (r *keycloakServiceProxy) DeRegisterClientInSSO(clientId string) *errors.ServiceError {
	if token, err := r.retrieveToken(); err != nil {
		return err
	} else {
		glog.V(5).Infof("Attempting to deregister client with id: %s in SSO", clientId)
		return r.service.DeRegisterClientInSSO(token, clientId)
	}
}

func (r *keycloakServiceProxy) RegisterClientInSSO(clusterId string, clusterOathCallbackURI string) (string, *errors.ServiceError) {
	if token, err := r.retrieveToken(); err != nil {
		return "", err
	} else {
		glog.V(5).Infof("Attempting to register client")
		return r.service.RegisterClientInSSO(token, clusterId, clusterOathCallbackURI)
	}
}

func (r *keycloakServiceProxy) GetConfig() *keycloak.KeycloakConfig {
	return r.service.GetConfig()
}

func (r *keycloakServiceProxy) GetRealmConfig() *keycloak.KeycloakRealmConfig {
	return r.service.GetRealmConfig()
}

func (r *keycloakServiceProxy) IsKafkaClientExist(clientId string) *errors.ServiceError {
	if token, err := r.retrieveToken(); err != nil {
		return err
	} else {
		glog.V(5).Infof("Checking if kafka client with id %s exists", clientId)
		return r.service.IsKafkaClientExist(token, clientId)
	}
}

func (r *keycloakServiceProxy) CreateServiceAccount(serviceAccountRequest *api.ServiceAccountRequest, ctx context.Context) (*api.ServiceAccount, *errors.ServiceError) {
	if token, err := tokenForServiceAPIHandler(ctx, r); err != nil {
		return nil, err
	} else {
		glog.V(5).Infof("Attempting to create service account")
		return r.service.CreateServiceAccount(token, serviceAccountRequest, ctx)
	}
}

func (r *keycloakServiceProxy) DeleteServiceAccount(ctx context.Context, clientId string) *errors.ServiceError {
	if token, err := tokenForServiceAPIHandler(ctx, r); err != nil {
		return err
	} else {
		glog.V(5).Infof("Attempting to delete service account with clientId: %s", clientId)
		return r.service.DeleteServiceAccount(token, ctx, clientId)
	}
}

func (r *keycloakServiceProxy) ResetServiceAccountCredentials(ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError) {
	if token, err := tokenForServiceAPIHandler(ctx, r); err != nil {
		return nil, err
	} else {
		glog.V(5).Infof("Attempting to reset service account credentials")
		return r.service.ResetServiceAccountCredentials(token, ctx, clientId)
	}
}

func (r *keycloakServiceProxy) ListServiceAcc(ctx context.Context, first int, max int) ([]api.ServiceAccount, *errors.ServiceError) {
	if token, err := tokenForServiceAPIHandler(ctx, r); err != nil {
		return nil, err
	} else {
		glog.V(5).Infof("Attempting to list all service accounts")
		return r.service.ListServiceAcc(token, ctx, first, max)
	}
}

func (r *keycloakServiceProxy) RegisterKasFleetshardOperatorServiceAccount(agentClusterId string) (*api.ServiceAccount, *errors.ServiceError) {
	if token, err := r.retrieveToken(); err != nil {
		return nil, err
	} else {
		glog.V(5).Infof("Attempting to register service account")
		return r.service.RegisterKasFleetshardOperatorServiceAccount(token, agentClusterId)
	}
}

func (r *keycloakServiceProxy) DeRegisterKasFleetshardOperatorServiceAccount(agentClusterId string) *errors.ServiceError {
	if token, err := r.retrieveToken(); err != nil {
		return err
	} else {
		glog.V(5).Infof("Attempting to deregister service account")
		return r.service.DeRegisterKasFleetshardOperatorServiceAccount(token, agentClusterId)
	}
}

func (r *keycloakServiceProxy) GetServiceAccountById(ctx context.Context, id string) (*api.ServiceAccount, *errors.ServiceError) {
	if token, err := tokenForServiceAPIHandler(ctx, r); err != nil {
		return nil, err
	} else {
		glog.V(5).Infof("Attempting to retrieve service account by id : %s", id)
		return r.service.GetServiceAccountById(token, ctx, id)
	}
}

func (r *keycloakServiceProxy) GetServiceAccountByClientId(ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError) {
	if token, err := tokenForServiceAPIHandler(ctx, r); err != nil {
		return nil, err
	} else {
		glog.V(5).Infof("Attempting to retrieve service account by client id : %s", clientId)
		return r.service.GetServiceAccountByClientId(token, ctx, clientId)
	}
}

func (r *keycloakServiceProxy) RegisterConnectorFleetshardOperatorServiceAccount(agentClusterId string) (*api.ServiceAccount, *errors.ServiceError) {
	if token, err := r.retrieveToken(); err != nil {
		return nil, err
	} else {
		glog.V(5).Infof("Attempting to Register Connector Fleetshard Operator Service Account")
		return r.service.RegisterConnectorFleetshardOperatorServiceAccount(token, agentClusterId)
	}
}
func (r *keycloakServiceProxy) DeRegisterConnectorFleetshardOperatorServiceAccount(agentClusterId string) *errors.ServiceError {
	if token, err := r.retrieveToken(); err != nil {
		return err
	} else {
		glog.V(5).Infof("Attempting to DeRegister Connector Fleetshard Operator Service Account")
		return r.service.DeRegisterConnectorFleetshardOperatorServiceAccount(token, agentClusterId)
	}
}
func (r *keycloakServiceProxy) GetKafkaClientSecret(clientId string) (string, *errors.ServiceError) {
	if token, err := r.retrieveToken(); err != nil {
		return "", err
	} else {
		glog.V(5).Infof("Attempting to get kafka client secret")
		return r.service.GetKafkaClientSecret(token, clientId)
	}
}
func (r *keycloakServiceProxy) CreateServiceAccountInternal(request CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError) {
	if token, err := r.retrieveToken(); err != nil {
		return nil, err
	} else {
		return r.service.CreateServiceAccountInternal(token, request)
	}
}
func (r *keycloakServiceProxy) DeleteServiceAccountInternal(clientId string) *errors.ServiceError {
	if token, err := r.retrieveToken(); err != nil {
		return err
	} else {
		return r.service.DeleteServiceAccountInternal(token, clientId)
	}
}

// Utility functions

func (r *keycloakServiceProxy) retrieveToken() (string, *errors.ServiceError) {
	glog.V(5).Infof("Attempting to retrieve token")
	accessToken, tokenErr := r.accessTokenProvider.GetToken()
	if tokenErr != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, tokenErr, "error getting access token")
	}
	glog.V(5).Infof("Token retrieved successfully")
	return accessToken, nil
}

func retrieveUserToken(ctx context.Context) (string, *errors.ServiceError) {
	glog.V(5).Infof("Attempting to retrieve user token")
	userToken, err := authentication.TokenFromContext(ctx)
	if err != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, err, "error getting access token")
	}
	token := userToken.Raw
	glog.V(5).Infof("Token retrieved successfully")
	return token, nil
}

func tokenForServiceAPIHandler(ctx context.Context, r *keycloakServiceProxy) (string, *errors.ServiceError) {
	glog.V(5).Infof("Attempting to retrieve token for Service API Handler")
	var token string
	var err *errors.ServiceError
	if r.GetConfig().SelectSSOProvider == keycloak.REDHAT_SSO {
		token, err = retrieveUserToken(ctx)
	} else {
		token, err = r.retrieveToken()
	}
	if err != nil {
		return "", err
	}
	glog.V(5).Infof("Token retrieved successfully")
	return token, nil
}
