package sso

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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

type redhatssoError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

const (
	ServiceAccountLimitExceeded = "service_account_limit_exceeded"
	ServiceAccountNotFound      = "service_account_not_found"
	ServiceAccountUserNotFound  = "service_account_user_not_found"
	ServiceAccountAccessInvalid = "service_account_access_invalid"
)

func parseRedhatssoError(e error) (*redhatssoError, error) {
	if e == nil {
		return nil, fmt.Errorf("passed in error is nil")
	}

	if openApiError, ok := e.(serviceaccountsclient.GenericOpenAPIError); ok {
		if openApiError.Body() != nil {
			rhErr := redhatssoError{}
			if err := json.Unmarshal(openApiError.Body(), &rhErr); err != nil {
				// Unable to get error description
				return nil, err
			}
			return &rhErr, nil
		}
	}

	return nil, fmt.Errorf("not an OpenAPI error")
}

func getErrorDescription(e error, defaultDesc string) string {
	if rhErr, err := parseRedhatssoError(e); err == nil {
		return rhErr.ErrorDescription
	}

	return defaultDesc
}

func (r *redhatssoService) RegisterClientInSSO(accessToken string, clusterId string, clusterOathCallbackURI string) (string, *errors.ServiceError) {
	// TODO
	return "", errors.New(errors.ErrorGeneral, "RegisterClientInSSO Not implemented")
}

func (r *redhatssoService) DeRegisterClientInSSO(accessToken string, clientId string) *errors.ServiceError {
	glog.V(5).Infof("Deregistering client with id: %s", clientId)
	err := r.client.DeleteServiceAccount(accessToken, clientId)
	if err != nil {
		errorDetail := getErrorDescription(err, "failed to delete the sso client")
		glog.V(5).Infof("Error deregistering client with id %s: %s", clientId, errorDetail)
		return errors.NewWithCause(errors.ErrorFailedToDeleteSSOClient, err, errorDetail)
	}
	glog.V(5).Infof("Deregistered client with id: %s", clientId)
	return nil
}

func (r *redhatssoService) GetConfig() *keycloak.KeycloakConfig {
	return r.client.GetConfig()
}

func (r *redhatssoService) GetRealmConfig() *keycloak.KeycloakRealmConfig {
	return r.client.GetRealmConfig()
}

func (r *redhatssoService) IsKafkaClientExist(accessToken string, clientId string) *errors.ServiceError {
	glog.V(5).Infof("Checking if client with id: %s exists", clientId)
	_, found, err := r.client.GetServiceAccount(accessToken, clientId)
	if err != nil {
		errorDetail := getErrorDescription(err, fmt.Sprintf("failed to get sso client with id: %s", clientId))
		glog.V(5).Infof("Failure checking kafka client existence: %s", errorDetail)
		return errors.NewWithCause(errors.ErrorFailedToGetSSOClient, err, errorDetail)
	}

	if !found {
		return errors.New(errors.ErrorNotFound, "sso client with id: %s not found", clientId)
	}
	glog.V(5).Infof("sso client with id: %s found", clientId)
	return nil
}

func (r *redhatssoService) CreateServiceAccount(accessToken string, serviceAccountRequest *api.ServiceAccountRequest, ctx context.Context) (*api.ServiceAccount, *errors.ServiceError) {
	glog.V(5).Infof("Creating service account with name: %s", serviceAccountRequest.Name)
	serviceAccount, err := r.client.CreateServiceAccount(accessToken, serviceAccountRequest.Name, serviceAccountRequest.Description)
	if err != nil {
		if rhErr, err1 := parseRedhatssoError(err); err1 == nil {
			switch rhErr.Error {
			case ServiceAccountLimitExceeded:
				glog.V(5).Infof("Failure creating service account: max allowed number:%d of service accounts for user has reached", r.GetConfig().MaxAllowedServiceAccounts)
				return nil, errors.MaxLimitForServiceAccountReached("Max allowed number:%d of service accounts for user has reached", r.GetConfig().MaxAllowedServiceAccounts)
			case ServiceAccountAccessInvalid:
				glog.V(5).Infof("Failure creating service account: account access invalid")
				return nil, errors.NewWithCause(errors.ErrorForbidden, err, "failed to create agent service account")
			}
		}

		errorDetail := getErrorDescription(err, "failed to create service account")
		glog.V(5).Infof("Failure creating service account: %s", errorDetail)
		return nil, errors.NewWithCause(errors.ErrorFailedToCreateServiceAccount, err, errorDetail)
	}
	glog.V(5).Infof("Service account %s created", serviceAccountRequest.Name)
	return convertServiceAccountDataToAPIServiceAccount(&serviceAccount), nil
}

func (r *redhatssoService) DeleteServiceAccount(accessToken string, ctx context.Context, clientId string) *errors.ServiceError {
	glog.V(5).Infof("Deleting service account with id: %s", clientId)
	err := r.client.DeleteServiceAccount(accessToken, clientId)
	if err != nil { //5xx
		if rhErr, err1 := parseRedhatssoError(err); err1 == nil {
			switch rhErr.Error {
			case ServiceAccountNotFound:
				glog.V(5).Infof("service account not found %s", clientId)
				return errors.NewWithCause(errors.ErrorServiceAccountNotFound, err, "service account not found %s", clientId)
			case ServiceAccountAccessInvalid:
				glog.V(5).Infof("service account not found %s", err.Error())
				return errors.NewWithCause(errors.ErrorForbidden, err, "service account not found %s", clientId)
			default:
				glog.V(5).Infof("failed to delete service account: %s", err.Error())
				return errors.NewWithCause(errors.ErrorFailedToDeleteServiceAccount, err, "failed to delete service account")
			}
		}
		glog.V(5).Infof("failed to delete service account: %s", err.Error())
		return errors.NewWithCause(errors.ErrorFailedToDeleteServiceAccount, err, "failed to delete service account")
	}
	glog.V(5).Infof("service account with id: %s deleted", clientId)
	return nil
}

func (r *redhatssoService) ResetServiceAccountCredentials(accessToken string, ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError) {
	glog.V(5).Infof("Regenerating service account credentials for client with id: %s", clientId)
	serviceAccount, err := r.client.RegenerateClientSecret(accessToken, clientId)
	if err != nil { //5xx
		if rhErr, err1 := parseRedhatssoError(err); err1 == nil {
			switch rhErr.Error {
			case ServiceAccountNotFound:
				glog.V(5).Infof("service account not found %s", clientId)
				return nil, errors.NewWithCause(errors.ErrorServiceAccountNotFound, err, "service account not found %s", clientId)
			case ServiceAccountAccessInvalid:
				glog.V(5).Infof("service account access invalid %s", err.Error())
				return nil, errors.NewWithCause(errors.ErrorForbidden, err, "failed to reset service account credentials")
			}
		}
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to reset service account credentials")
	}
	glog.V(5).Infof("service account credentials regenerated for client with id: %s", clientId)
	return convertServiceAccountDataToAPIServiceAccount(&serviceAccount), nil
}

func (r *redhatssoService) ListServiceAcc(accessToken string, ctx context.Context, first int, max int) ([]api.ServiceAccount, *errors.ServiceError) {
	glog.V(5).Infof("Listing service accounts")
	accounts, err := r.client.GetServiceAccounts(accessToken, first, max)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to collect service accounts")
	}
	var res []api.ServiceAccount

	for i := range accounts {
		res = append(res, *convertServiceAccountDataToAPIServiceAccount(&accounts[i]))
	}
	return res, nil
}

func (r *redhatssoService) RegisterKasFleetshardOperatorServiceAccount(accessToken string, agentClusterId string) (*api.ServiceAccount, *errors.ServiceError) {
	return r.registerAgentServiceAccount(accessToken, agentClusterId)
}

func (r *redhatssoService) registerAgentServiceAccount(accessToken string, agentClusterId string) (*api.ServiceAccount, *errors.ServiceError) {
	glog.V(5).Infof("Registering agent service account with cluster: %s", agentClusterId)
	ret, err := r.CreateServiceAccount(accessToken, &api.ServiceAccountRequest{
		Name:        agentClusterId,
		Description: fmt.Sprintf("service account for agent on cluster %s", agentClusterId),
	}, context.Background())

	if err != nil {
		return nil, err
	}

	glog.V(5).Infof("Agent service account registered with cluster: %s", agentClusterId)
	return ret, nil
}

func (r *redhatssoService) DeRegisterKasFleetshardOperatorServiceAccount(accessToken string, agentClusterId string) *errors.ServiceError {
	glog.V(5).Infof("Deregistering kas Fleetshard agent service account with cluster: %s", agentClusterId)
	if _, found, err := r.client.GetServiceAccount(accessToken, agentClusterId); err != nil {
		return errors.NewWithCause(errors.ErrorFailedToGetSSOClient, err, "failed to get sso client with id: %s", agentClusterId)
	} else {
		if !found {
			// if the account to be deleted does not exists, we simply exit with no errors
			glog.V(5).Infof("kas Fleetshard agent service account not found")
			return nil
		}
	}

	err := r.DeleteServiceAccount(accessToken, context.Background(), agentClusterId)
	if err != nil {
		return err
	}
	glog.V(5).Infof("Kas Fleetshard agent service account deregistered with cluster: %s", agentClusterId)
	return nil
}

func (r *redhatssoService) GetServiceAccountById(accessToken string, ctx context.Context, id string) (*api.ServiceAccount, *errors.ServiceError) {
	return r.GetServiceAccountByClientId(accessToken, ctx, id)
}

func (r *redhatssoService) GetServiceAccountByClientId(accessToken string, ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError) {
	glog.V(5).Infof("Getting service account by client id: %s", clientId)
	serviceAccount, found, err := r.client.GetServiceAccount(accessToken, clientId)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "error retrieving service account with clientId %s", clientId)
	}

	if !found {
		return nil, errors.NewWithCause(errors.ErrorServiceAccountNotFound, err, "service account not found clientId %s", clientId)
	}
	glog.V(5).Infof("service account with client id: %s returned", clientId)
	return convertServiceAccountDataToAPIServiceAccount(serviceAccount), nil
}

func (r *redhatssoService) RegisterConnectorFleetshardOperatorServiceAccount(accessToken string, agentClusterId string) (*api.ServiceAccount, *errors.ServiceError) {
	return r.registerAgentServiceAccount(accessToken, agentClusterId)
}

func (r *redhatssoService) DeRegisterConnectorFleetshardOperatorServiceAccount(accessToken string, agentClusterId string) *errors.ServiceError {
	glog.V(5).Infof("DeRegistering Connector Fleetshard Operator Service Account with cluster: %s", agentClusterId)
	if _, found, err := r.client.GetServiceAccount(accessToken, agentClusterId); err != nil {
		return errors.NewWithCause(errors.ErrorFailedToGetSSOClient, err, "failed to get sso client with id: %s", agentClusterId)
	} else {
		if !found {
			// If the account does not exists, simply exit without errors
			glog.V(5).Infof("Connector Fleetshard agent service account not found")
			return nil
		}
	}

	err := r.client.DeleteServiceAccount(accessToken, agentClusterId)
	if err != nil {
		return errors.NewWithCause(errors.ErrorFailedToDeleteServiceAccount, err, "Failed to delete service account: %s", agentClusterId)
	}
	glog.V(5).Infof("Connector Fleetshard Operator Service Account DeRegistered with cluster: %s", agentClusterId)
	return nil
}

func (r *redhatssoService) GetKafkaClientSecret(accessToken string, clientId string) (string, *errors.ServiceError) {
	glog.V(5).Infof("Getting client secret for client id: %s", clientId)
	serviceAccount, found, err := r.client.GetServiceAccount(accessToken, clientId)
	if err != nil {
		return "", errors.NewWithCause(errors.ErrorFailedToGetSSOClient, err, "failed to get sso client with id: %s", clientId)
	}
	if found {
		//if client is found re-generate the client secret.
		svcData, seErr := r.client.RegenerateClientSecret(accessToken, shared.SafeString(serviceAccount.Id))
		if seErr != nil {
			return "", errors.NewWithCause(errors.ErrorFailedToGetSSOClientSecret, err, "failed to get sso client secret")
		}
		return shared.SafeString(svcData.Secret), nil
	}

	return "", errors.NewWithCause(errors.ErrorFailedToGetSSOClientSecret, err, "failed to get sso client secret")
}

func (r *redhatssoService) CreateServiceAccountInternal(accessToken string, request CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError) {
	req := api.ServiceAccountRequest{
		Name:        request.ClientId,
		Description: request.Description,
	}
	return r.CreateServiceAccount(accessToken, &req, context.Background())
}

func (r *redhatssoService) DeleteServiceAccountInternal(accessToken string, clientId string) *errors.ServiceError {
	// This is code can be removed once the migrated canary & kas-fleetshard are removed. Performance impact would be less as the number of canary & kas-fleetshard are low.
	// Once the existing canary or kas-fleetshard are deleted. This logic can be removed.
	if strings.HasPrefix(clientId, "canary") || strings.HasPrefix(clientId, "kas-fleetshard-agent") {
		first := 0
		max := 100
		for {
			accounts, err := r.client.GetServiceAccounts(accessToken, first, max)
			if err != nil {
				glog.Errorf("failed to get internal service account: %s", err)
				return nil
			}
			if len(accounts) == 0 {
				return nil
			}
			for _, account := range accounts {
				if clientId == shared.SafeString(account.ClientId) {
					return r.DeleteServiceAccount(accessToken, context.Background(), shared.SafeString(account.Id))
				}
			}
			first = first + max
		}
	}
	return r.DeleteServiceAccount(accessToken, context.Background(), clientId)
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
