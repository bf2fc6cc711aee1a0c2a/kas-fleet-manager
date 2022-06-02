package sso

import (
	"context"
	"fmt"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/redhatsso"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/golang/glog"
	serviceaccountsclient "github.com/redhat-developer/app-services-sdk-go/serviceaccounts/apiv1internal/client"
)

var _ keycloakServiceInternal = &redhatssoService{}

type redhatssoService struct {
	client redhatsso.SSOClient
}

func (r *redhatssoService) RegisterClientInSSO(accessToken string, clusterId string, clusterOathCallbackURI string) (string, *errors.ServiceError) {
	// TODO
	return "", errors.New(errors.ErrorGeneral, "RegisterClientInSSO Not implemented")
}

func (r *redhatssoService) DeRegisterClientInSSO(accessToken string, clientId string) *errors.ServiceError {
	glog.V(5).Infof("Attempting to deregister client with id: %s in SSO", clientId)
	err := r.client.DeleteServiceAccount(accessToken, clientId)
	if err != nil {
		return errors.NewWithCause(errors.ErrorFailedToDeleteSSOClient, err, "failed to delete the sso client")
	}
	glog.V(5).Infof("Kafka Client %s deleted successfully", clientId)
	return nil
}

func (r *redhatssoService) GetConfig() *keycloak.KeycloakConfig {
	return r.client.GetConfig()
}

func (r *redhatssoService) GetRealmConfig() *keycloak.KeycloakRealmConfig {
	return r.client.GetRealmConfig()
}

func (r *redhatssoService) IsKafkaClientExist(accessToken string, clientId string) *errors.ServiceError {
	glog.V(5).Infof("Checking if kafka client with id %s exists", clientId)
	_, found, err := r.client.GetServiceAccount(accessToken, clientId)
	if err != nil {
		return errors.NewWithCause(errors.ErrorFailedToGetSSOClient, err, "failed to get sso client with id: %s", clientId)
	}

	if !found {
		return errors.New(errors.ErrorNotFound, "sso client with id: %s not found", clientId)
	}
	glog.V(5).Infof("kafka client with id %s found", clientId)
	return nil
}

func (r *redhatssoService) CreateServiceAccount(accessToken string, serviceAccountRequest *api.ServiceAccountRequest, ctx context.Context) (*api.ServiceAccount, *errors.ServiceError) {
	glog.V(5).Infof("Attempting to create service account")
	serviceAccount, err := r.client.CreateServiceAccount(accessToken, serviceAccountRequest.Name, serviceAccountRequest.Description)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorFailedToCreateServiceAccount, err, "failed to create service account")
	}
	glog.V(5).Infof("Service account created successfully")
	return convertServiceAccountDataToAPIServiceAccount(&serviceAccount), nil
}

func (r *redhatssoService) DeleteServiceAccount(accessToken string, ctx context.Context, clientId string) *errors.ServiceError {
	glog.V(5).Infof("Attempting to delete service account")
	err := r.client.DeleteServiceAccount(accessToken, clientId)
	if err != nil { //5xx
		return errors.NewWithCause(errors.ErrorFailedToDeleteServiceAccount, err, "failed to delete service account")
	}
	glog.V(5).Infof("deleted service account clientId = %s ", clientId)
	return nil
}

func (r *redhatssoService) ResetServiceAccountCredentials(accessToken string, ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError) {
	glog.V(5).Infof("Attempting to reset service account credentials")
	serviceAccount, err := r.client.RegenerateClientSecret(accessToken, clientId)
	if err != nil { //5xx
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to reset service account credentials")
	}
	glog.V(5).Infof("Service account credentials successfully reset")
	return convertServiceAccountDataToAPIServiceAccount(&serviceAccount), nil
}

func (r *redhatssoService) ListServiceAcc(accessToken string, ctx context.Context, first int, max int) ([]api.ServiceAccount, *errors.ServiceError) {
	glog.V(5).Infof("Attempting to list all service accounts")
	accounts, err := r.client.GetServiceAccounts(accessToken, first, max)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to collect service accounts")
	}
	var res []api.ServiceAccount

	for _, account := range accounts {
		res = append(res, *convertServiceAccountDataToAPIServiceAccount(&account))
	}
	glog.V(5).Infof("Service accounts listed successfully")
	return res, nil
}

func (r *redhatssoService) RegisterKasFleetshardOperatorServiceAccount(accessToken string, agentClusterId string) (*api.ServiceAccount, *errors.ServiceError) {
	return r.registerAgentServiceAccount(accessToken, agentClusterId)
}

func (r *redhatssoService) registerAgentServiceAccount(accessToken string, agentClusterId string) (*api.ServiceAccount, *errors.ServiceError) {
	glog.V(5).Infof("Attempting to register service account")
	svcData, err := r.client.CreateServiceAccount(accessToken, agentClusterId, fmt.Sprintf("service account for agent on cluster %s", agentClusterId))
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to create agent service account")
	}
	glog.V(5).Infof("Service account registered successfully")
	return convertServiceAccountDataToAPIServiceAccount(&svcData), nil
}

func (r *redhatssoService) DeRegisterKasFleetshardOperatorServiceAccount(accessToken string, agentClusterId string) *errors.ServiceError {
	glog.V(5).Infof("Attempting to deregister service account")
	if _, found, err := r.client.GetServiceAccount(accessToken, agentClusterId); err != nil {
		return errors.NewWithCause(errors.ErrorFailedToDeleteServiceAccount, err, "Failed to delete service account: %s", agentClusterId)
	} else {
		if !found {
			// if the account to be deleted does not exists, we simply exit with no errors
			glog.V(5).Infof("Service account not found")
			return nil
		}
	}

	err := r.client.DeleteServiceAccount(accessToken, agentClusterId)
	if err != nil {
		return errors.NewWithCause(errors.ErrorFailedToDeleteServiceAccount, err, "Failed to delete service account: %s", agentClusterId)
	}
	glog.V(5).Infof("Service account deregistered successfully")
	return nil
}

func (r *redhatssoService) GetServiceAccountById(accessToken string, ctx context.Context, id string) (*api.ServiceAccount, *errors.ServiceError) {
	return r.GetServiceAccountByClientId(accessToken, ctx, id)
}

func (r *redhatssoService) GetServiceAccountByClientId(accessToken string, ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError) {
	glog.V(5).Infof("Attempting to retrieve service account by client id : %s", clientId)
	serviceAccount, found, err := r.client.GetServiceAccount(accessToken, clientId)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "error retrieving service account with clientId %s", clientId)
	}

	if !found {
		return nil, errors.NewWithCause(errors.ErrorServiceAccountNotFound, err, "service account not found clientId %s", clientId)
	}
	glog.V(5).Infof("Service account with id %s retrieved", clientId)
	return convertServiceAccountDataToAPIServiceAccount(serviceAccount), nil
}

func (r *redhatssoService) RegisterConnectorFleetshardOperatorServiceAccount(accessToken string, agentClusterId string) (*api.ServiceAccount, *errors.ServiceError) {
	return r.registerAgentServiceAccount(accessToken, agentClusterId)
}

func (r *redhatssoService) DeRegisterConnectorFleetshardOperatorServiceAccount(accessToken string, agentClusterId string) *errors.ServiceError {
	glog.V(5).Infof("Attempting to DeRegister Connector Fleetshard Operator Service Account")
	if _, found, err := r.client.GetServiceAccount(accessToken, agentClusterId); err != nil {
		return errors.NewWithCause(errors.ErrorFailedToDeleteServiceAccount, err, "Failed to delete service account: %s", agentClusterId)
	} else {
		if !found {
			// If the account does not exists, simply exit without errors
			glog.V(5).Infof("Client does not exist")
			return nil
		}
	}

	err := r.client.DeleteServiceAccount(accessToken, agentClusterId)
	if err != nil {
		return errors.NewWithCause(errors.ErrorFailedToDeleteServiceAccount, err, "Failed to delete service account: %s", agentClusterId)
	}
	glog.V(5).Infof("Successfully deregistered Connector Fleetshard Operator Service Account")
	return nil
}

func (r *redhatssoService) GetKafkaClientSecret(accessToken string, clientId string) (string, *errors.ServiceError) {
	glog.V(5).Infof("Attempting to get kafka client secret")
	serviceAccount, found, err := r.client.GetServiceAccount(accessToken, clientId)
	if err != nil {
		return "", errors.NewWithCause(errors.ErrorFailedToGetSSOClientSecret, err, "failed to get sso client secret")
	}
	if found {
		//if client is found re-generate the client secret.
		glog.V(5).Infof("Kafka client not found regenerating client secret")
		svcData, seErr := r.client.RegenerateClientSecret(accessToken, shared.SafeString(serviceAccount.Id))
		if seErr != nil {
			return "", errors.NewWithCause(errors.ErrorFailedToGetSSOClientSecret, err, "failed to get sso client secret")
		}
		return shared.SafeString(svcData.Secret), nil
	}

	return "", errors.NewWithCause(errors.ErrorFailedToGetSSOClientSecret, err, "failed to get sso client secret")
}

func (r *redhatssoService) CreateServiceAccountInternal(accessToken string, request CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError) {
	svcData, err := r.client.CreateServiceAccount(accessToken, request.ClientId, request.Description)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorFailedToCreateServiceAccount, err, "failed to create service account")
	}
	return convertServiceAccountDataToAPIServiceAccount(&svcData), nil
}

func (r *redhatssoService) DeleteServiceAccountInternal(accessToken string, clientId string) *errors.ServiceError {
	err := r.client.DeleteServiceAccount(accessToken, clientId)
	if err != nil {
		return errors.NewWithCause(errors.ErrorFailedToDeleteServiceAccount, err, "Failed to delete service account: %s", clientId)
	}
	return nil
}

//// utility functions
func convertServiceAccountDataToAPIServiceAccount(data *serviceaccountsclient.ServiceAccountData) *api.ServiceAccount {
	return &api.ServiceAccount{
		ID:           shared.SafeString(data.Id),
		ClientID:     shared.SafeString(data.ClientId),
		ClientSecret: shared.SafeString(data.Secret),
		Name:         shared.SafeString(data.Name),
		CreatedBy:    shared.SafeString(data.CreatedBy),
		Description:  shared.SafeString(data.Description),
		CreatedAt:    time.Unix(shared.SafeInt64(data.CreatedAt), 0),
	}
}
