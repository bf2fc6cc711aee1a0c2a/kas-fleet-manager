package sso

import (
	"context"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/redhatsso"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/golang/glog"
	serviceaccountsclient "github.com/redhat-developer/app-services-sdk-go/serviceaccounts/apiv1internal/client"
	"time"
)

var _ keycloakServiceInternal = &redhatssoService{}

type redhatssoService struct {
	client redhatsso.SSOClient
}

func (r *redhatssoService) RegisterKafkaClientInSSO(accessToken string, kafkaNamespace string, orgId string) (string, *errors.ServiceError) {
	svcData, err := r.client.CreateServiceAccount(accessToken, kafkaNamespace, fmt.Sprintf("%s:%s", orgId, kafkaNamespace))
	if err != nil {
		return "", errors.NewWithCause(errors.ErrorFailedToCreateSSOClient, err, "failed to register Kafka Client in SSO")
	}
	if svcData.Secret != nil {
		return *svcData.Secret, nil
	} else {
		return "", errors.NewWithCause(errors.ErrorGeneral, err, "failed to retrieve secret while registering Kafka Client in SSO")
	}
}

func (r *redhatssoService) RegisterOSDClusterClientInSSO(accessToken string, clusterId string, clusterOathCallbackURI string) (string, *errors.ServiceError) {
	// TODO
	return "", errors.New(errors.ErrorGeneral, "Not implemented")
}

func (r *redhatssoService) DeRegisterClientInSSO(accessToken string, clientId string) *errors.ServiceError {
	err := r.client.DeleteServiceAccount(accessToken, clientId)
	if err != nil {
		return errors.NewWithCause(errors.ErrorFailedToDeleteSSOClient, err, "failed to delete the sso client")
	}
	glog.V(5).Infof("Kafka Client %s deleted successfully", clientId)
	return nil
}

func (r *redhatssoService) GetConfig() *keycloak.KeycloakConfig {
	return nil
}

func (r *redhatssoService) GetRealmConfig() *keycloak.KeycloakRealmConfig {
	return nil
}

func (r *redhatssoService) IsKafkaClientExist(accessToken string, clientId string) *errors.ServiceError {
	_, err := r.client.GetServiceAccount(accessToken, clientId)
	if err != nil {
		return errors.NewWithCause(errors.ErrorFailedToGetSSOClient, err, "failed to get sso client with id: %s", clientId)
	}
	return nil
}

func (r *redhatssoService) CreateServiceAccount(accessToken string, serviceAccountRequest *api.ServiceAccountRequest, ctx context.Context) (*api.ServiceAccount, *errors.ServiceError) {
	serviceAccount, err := r.client.CreateServiceAccount(accessToken, serviceAccountRequest.Name, serviceAccountRequest.Description)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorFailedToCreateServiceAccount, err, "failed to create service account")
	}

	return convertServiceAccountDataToAPIServiceAccount(&serviceAccount), nil
}

func (r *redhatssoService) DeleteServiceAccount(accessToken string, ctx context.Context, clientId string) *errors.ServiceError {
	err := r.client.DeleteServiceAccount(accessToken, clientId)
	if err != nil { //5xx
		return errors.NewWithCause(errors.ErrorFailedToDeleteServiceAccount, err, "failed to delete service account")
	}
	glog.V(5).Infof("deleted service account clientId = %s ", clientId)
	return nil
}

func (r *redhatssoService) ResetServiceAccountCredentials(accessToken string, ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError) {
	serviceAccount, err := r.client.RegenerateClientSecret(accessToken, clientId)
	if err != nil { //5xx
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to reset service account credentials")
	}
	return convertServiceAccountDataToAPIServiceAccount(&serviceAccount), nil
}

func (r *redhatssoService) ListServiceAcc(accessToken string, ctx context.Context, first int, max int) ([]api.ServiceAccount, *errors.ServiceError) {
	// TODO
	return nil, errors.New(errors.ErrorGeneral, "Not implemented")
}

func (r *redhatssoService) RegisterKasFleetshardOperatorServiceAccount(accessToken string, agentClusterId string) (*api.ServiceAccount, *errors.ServiceError) {
	svcData, err := r.client.CreateServiceAccount(accessToken, agentClusterId, agentClusterId)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to update agent service account")
	}

	return convertServiceAccountDataToAPIServiceAccount(&svcData), nil
}
func (r *redhatssoService) DeRegisterKasFleetshardOperatorServiceAccount(accessToken string, agentClusterId string) *errors.ServiceError {
	err := r.client.DeleteServiceAccount(accessToken, agentClusterId)
	if err != nil {
		return errors.NewWithCause(errors.ErrorFailedToDeleteServiceAccount, err, "Failed to delete service account: %s", agentClusterId)
	}
	return nil
}
func (r *redhatssoService) GetServiceAccountById(accessToken string, ctx context.Context, id string) (*api.ServiceAccount, *errors.ServiceError) {
	return r.GetServiceAccountByClientId(accessToken, ctx, id)
}
func (r *redhatssoService) GetServiceAccountByClientId(accessToken string, ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError) {
	serviceAccount, err := r.client.GetServiceAccount(accessToken, clientId)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorServiceAccountNotFound, err, "service account not found clientId %s", clientId)
	}
	return convertServiceAccountDataToAPIServiceAccount(&serviceAccount), nil
}
func (r *redhatssoService) RegisterConnectorFleetshardOperatorServiceAccount(accessToken string, agentClusterId string) (*api.ServiceAccount, *errors.ServiceError) {
	// TODO
	return nil, errors.GeneralError("not implemented")
}
func (r *redhatssoService) DeRegisterConnectorFleetshardOperatorServiceAccount(accessToken string, agentClusterId string) *errors.ServiceError {
	err := r.client.DeleteServiceAccount(accessToken, agentClusterId)
	if err != nil {
		return errors.NewWithCause(errors.ErrorFailedToDeleteServiceAccount, err, "Failed to delete service account: %s", agentClusterId)
	}
	return nil
}
func (r *redhatssoService) GetKafkaClientSecret(accessToken string, clientId string) (string, *errors.ServiceError) {
	serviceAccount, err := r.client.GetServiceAccount(accessToken, clientId)
	if err != nil {
		return "", errors.NewWithCause(errors.ErrorFailedToGetSSOClientSecret, err, "failed to get sso client secret")
	}

	return *serviceAccount.Secret, nil
}
func (r *redhatssoService) CreateServiceAccountInternal(accessToken string, request CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError) {
	return nil, nil
}
func (r *redhatssoService) DeleteServiceAccountInternal(accessToken string, clientId string) *errors.ServiceError {
	return nil
}

//// utility functions
func convertServiceAccountDataToAPIServiceAccount(data *serviceaccountsclient.ServiceAccountData) *api.ServiceAccount {
	return &api.ServiceAccount{
		ID:           shared.SafeString(data.Id),
		ClientID:     shared.SafeString(data.ClientId),
		ClientSecret: shared.SafeString(data.Secret),
		Name:         shared.SafeString(data.Name),
		Owner:        shared.SafeString(data.OwnerId), // TODO: owner is deprecated. Check with SSM team about a new OPENAPI for createdBy
		Description:  shared.SafeString(data.Description),
		CreatedAt:    time.UnixMilli(shared.SafeInt64(data.CreatedAt)),
	}
}
