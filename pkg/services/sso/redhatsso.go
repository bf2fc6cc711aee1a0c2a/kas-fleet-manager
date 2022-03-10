package sso

import (
	"context"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/redhatsso"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
)

type redhatssoServiceProxy struct {
	client  redhatsso.SSOClient
	service redhatssoService
}

var _ services.KeycloakService = &redhatssoServiceProxy{}

func (r *redhatssoServiceProxy) retrieveToken() (string, *errors.ServiceError) {
	accessToken, tokenErr := r.client.GetToken()
	if tokenErr != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, tokenErr, "error getting access token")
	}
	return accessToken, nil
}

func (r *redhatssoServiceProxy) RegisterKafkaClientInSSO(kafkaNamespace string, orgId string) (string, *errors.ServiceError) {
	if token, err := r.retrieveToken(); err != nil {
		return "", err
	} else {
		return r.service.RegisterKafkaClientInSSO(token, kafkaNamespace, orgId)
	}
}

func (r *redhatssoServiceProxy) RegisterOSDClusterClientInSSO(clusterId string, clusterOathCallbackURI string) (string, *errors.ServiceError) {
	if token, err := r.retrieveToken(); err != nil {
		return "", err
	} else {
		return r.service.RegisterOSDClusterClientInSSO(token, clusterId, clusterOathCallbackURI)
	}
}

func (r *redhatssoServiceProxy) DeRegisterClientInSSO(clientId string) *errors.ServiceError {
	if token, err := r.retrieveToken(); err != nil {
		return err
	} else {
		return r.service.DeRegisterClientInSSO(token, clientId)
	}
}

func (r *redhatssoServiceProxy) GetConfig() *keycloak.KeycloakConfig {
	return r.service.GetConfig()
}

func (r *redhatssoServiceProxy) GetRealmConfig() *keycloak.KeycloakRealmConfig {
	return r.service.GetRealmConfig()
}

func (r *redhatssoServiceProxy) IsKafkaClientExist(clientId string) *errors.ServiceError {
	if token, err := r.retrieveToken(); err != nil {
		return err
	} else {
		return r.service.IsKafkaClientExist(token, clientId)
	}
}

func (r *redhatssoServiceProxy) CreateServiceAccount(serviceAccountRequest *api.ServiceAccountRequest, ctx context.Context) (*api.ServiceAccount, *errors.ServiceError) {
	if token, err := r.retrieveToken(); err != nil {
		return nil, err
	} else {
		return r.service.CreateServiceAccount(token, serviceAccountRequest, ctx)
	}
}

func (r *redhatssoServiceProxy) DeleteServiceAccount(ctx context.Context, clientId string) *errors.ServiceError {
	if token, err := r.retrieveToken(); err != nil {
		return err
	} else {
		return r.service.DeleteServiceAccount(token, ctx, clientId)
	}
}

func (r *redhatssoServiceProxy) ResetServiceAccountCredentials(ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError) {
	if token, err := r.retrieveToken(); err != nil {
		return nil, err
	} else {
		return r.service.ResetServiceAccountCredentials(token, ctx, clientId)
	}
}

func (r *redhatssoServiceProxy) ListServiceAcc(ctx context.Context, first int, max int) ([]api.ServiceAccount, *errors.ServiceError) {
	if token, err := r.retrieveToken(); err != nil {
		return nil, err
	} else {
		return r.service.ListServiceAcc(token, ctx, first, max)
	}
}

func (r *redhatssoServiceProxy) RegisterKasFleetshardOperatorServiceAccount(agentClusterId string) (*api.ServiceAccount, *errors.ServiceError) {
	if token, err := r.retrieveToken(); err != nil {
		return nil, err
	} else {
		return r.service.RegisterKasFleetshardOperatorServiceAccount(token, agentClusterId)
	}
}

func (r *redhatssoServiceProxy) DeRegisterKasFleetshardOperatorServiceAccount(agentClusterId string) *errors.ServiceError {
	if token, err := r.retrieveToken(); err != nil {
		return err
	} else {
		return r.service.DeRegisterKasFleetshardOperatorServiceAccount(token, agentClusterId)
	}
}

func (r *redhatssoServiceProxy) GetServiceAccountById(ctx context.Context, id string) (*api.ServiceAccount, *errors.ServiceError) {
	if token, err := r.retrieveToken(); err != nil {
		return nil, err
	} else {
		return r.service.GetServiceAccountById(token, ctx, id)
	}
}

func (r *redhatssoServiceProxy) GetServiceAccountByClientId(ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError) {
	if token, err := r.retrieveToken(); err != nil {
		return nil, err
	} else {
		return r.service.GetServiceAccountByClientId(token, ctx, clientId)
	}
}

func (r *redhatssoServiceProxy) RegisterConnectorFleetshardOperatorServiceAccount(agentClusterId string) (*api.ServiceAccount, *errors.ServiceError) {
	if token, err := r.retrieveToken(); err != nil {
		return nil, err
	} else {
		return r.service.RegisterConnectorFleetshardOperatorServiceAccount(token, agentClusterId)
	}
}
func (r *redhatssoServiceProxy) DeRegisterConnectorFleetshardOperatorServiceAccount(agentClusterId string) *errors.ServiceError {
	if token, err := r.retrieveToken(); err != nil {
		return err
	} else {
		return r.service.DeRegisterConnectorFleetshardOperatorServiceAccount(token, agentClusterId)
	}
}
func (r *redhatssoServiceProxy) GetKafkaClientSecret(clientId string) (string, *errors.ServiceError) {
	if token, err := r.retrieveToken(); err != nil {
		return "", err
	} else {
		return r.service.GetKafkaClientSecret(token, clientId)
	}
}
func (r *redhatssoServiceProxy) CreateServiceAccountInternal(request services.CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError) {
	if token, err := r.retrieveToken(); err != nil {
		return nil, err
	} else {
		return r.service.CreateServiceAccountInternal(token, request)
	}
}
func (r *redhatssoServiceProxy) DeleteServiceAccountInternal(clientId string) *errors.ServiceError {
	if token, err := r.retrieveToken(); err != nil {
		return err
	} else {
		return r.service.DeleteServiceAccountInternal(token, clientId)
	}
}
