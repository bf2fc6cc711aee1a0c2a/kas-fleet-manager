package services

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/auth"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/client/keycloak"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
)

const (
	rhOrgId = "rh-org-id"
)

//go:generate moq -out keycloakservice_moq.go . KeycloakService
type KeycloakService interface {
	RegisterKafkaClientInSSO(kafkaNamespace string, orgId string) (string, *errors.ServiceError)
	DeRegisterKafkaClientInSSO(kafkaNamespace string) *errors.ServiceError
	GetSecretForRegisteredKafkaClient(kafkaClusterName string) (string, *errors.ServiceError)
	GetConfig() *config.KeycloakConfig
	IsKafkaClientExist(clientId string) *errors.ServiceError
	CreateServiceAccount(serviceAccountRequest *api.ServiceAccountRequest, ctx context.Context) (*api.ServiceAccount, *errors.ServiceError)
	DeleteServiceAccount(ctx context.Context, clientId string) *errors.ServiceError
	ResetServiceAccountCredentials(ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError)
	ListServiceAcc(ctx context.Context, first int, max int) ([]api.ServiceAccount, *errors.ServiceError)
}

type keycloakService struct {
	kcClient keycloak.KcClient
}

var _ KeycloakService = &keycloakService{}

func NewKeycloakService(config *config.KeycloakConfig) *keycloakService {
	client := keycloak.NewClient(config)
	return &keycloakService{
		kcClient: client,
	}
}

func (kc *keycloakService) RegisterKafkaClientInSSO(kafkaClusterName string, orgId string) (string, *errors.ServiceError) {
	accessToken, _ := kc.kcClient.GetToken()
	internalClientId, err := kc.kcClient.IsClientExist(kafkaClusterName, accessToken)
	if err != nil {
		return "", errors.GeneralError("failed to check the sso client exists:%v", err)
	}
	if internalClientId != "" {
		secretValue, _ := kc.kcClient.GetClientSecret(internalClientId, accessToken)
		return secretValue, nil
	}

	rhOrgIdAttributes := map[string]string{
		rhOrgId: orgId,
	}
	c := keycloak.ClientRepresentation{
		ClientID:                     kafkaClusterName,
		Name:                         kafkaClusterName,
		ServiceAccountsEnabled:       true,
		AuthorizationServicesEnabled: true,
		StandardFlowEnabled:          false,
		Attributes:                   rhOrgIdAttributes,
	}
	clientConfig := kc.kcClient.ClientConfig(c)
	internalClient, err := kc.kcClient.CreateClient(clientConfig, accessToken)
	if err != nil {
		return "", errors.FailedToCreateSSOClient("failed to create the sso client")
	}
	secretValue, err := kc.kcClient.GetClientSecret(internalClient, accessToken)
	if err != nil {
		return "", errors.FailedToGetSSOClientSecret("failed to get the sso client secret")
	}
	return secretValue, nil
}

func (kc *keycloakService) GetSecretForRegisteredKafkaClient(kafkaClusterName string) (string, *errors.ServiceError) {
	accessToken, _ := kc.kcClient.GetToken()
	internalClientId, err := kc.kcClient.IsClientExist(kafkaClusterName, accessToken)
	if err != nil {
		return "", errors.FailedToGetSSOClient("failed to get the sso client")
	}
	if internalClientId != "" {
		secretValue, _ := kc.kcClient.GetClientSecret(internalClientId, accessToken)
		return secretValue, nil
	}
	return "", nil
}

func (kc *keycloakService) DeRegisterKafkaClientInSSO(kafkaClusterName string) *errors.ServiceError {
	accessToken, _ := kc.kcClient.GetToken()
	internalClientID, _ := kc.kcClient.IsClientExist(kafkaClusterName, accessToken)
	if internalClientID == "" {
		return nil
	}
	err := kc.kcClient.DeleteClient(internalClientID, accessToken)
	if err != nil {
		return errors.FailedToDeleteSSOClient("failed to delete the sso client")
	}
	return nil
}

func (kc *keycloakService) GetConfig() *config.KeycloakConfig {
	return kc.kcClient.GetConfig()
}

func (kc keycloakService) IsKafkaClientExist(clientId string) *errors.ServiceError {
	accessToken, _ := kc.kcClient.GetToken()
	_, err := kc.kcClient.IsClientExist(clientId, accessToken)
	if err != nil {
		return errors.FailedToGetSSOClient("failed to get the sso client")
	}
	return nil
}

func (kc *keycloakService) CreateServiceAccount(serviceAccountRequest *api.ServiceAccountRequest, ctx context.Context) (*api.ServiceAccount, *errors.ServiceError) {
	var serviceAcc api.ServiceAccount
	accessToken, _ := kc.kcClient.GetToken()
	orgId := auth.GetOrgIdFromContext(ctx)
	rhAccountID := map[string][]string{
		"rh-ord-id": {orgId},
	}
	rhOrgIdAttributes := map[string]string{
		rhOrgId: orgId,
	}
	protocolMapper := kc.kcClient.CreateProtocolMapperConfig(rhOrgId)

	c := keycloak.ClientRepresentation{
		ClientID:               kc.buildServiceAccountIdentifier(),
		Name:                   serviceAccountRequest.Name,
		Description:            serviceAccountRequest.Description,
		ServiceAccountsEnabled: true,
		StandardFlowEnabled:    false,
		ProtocolMappers:        protocolMapper,
		Attributes:             rhOrgIdAttributes,
	}
	clientConfig := kc.kcClient.ClientConfig(c)
	internalClient, err := kc.kcClient.CreateClient(clientConfig, accessToken)
	if err != nil {
		return nil, errors.FailedToCreateServiceAccount("failed to create the service account")
	}
	clientSecret, err := kc.kcClient.GetClientSecret(internalClient, accessToken)
	if err != nil {
		return nil, errors.FailedToGetSSOClientSecret("failed to get service account secret")
	}
	serviceAccountUser, error := kc.kcClient.GetClientServiceAccount(accessToken, internalClient)
	if error != nil {
		return nil, errors.FailedToGetServiceAccount("failed fetch the service account user")
	}
	serviceAccountUser.Attributes = &rhAccountID
	serAccUser := *serviceAccountUser
	updateErr := kc.kcClient.UpdateServiceAccountUser(accessToken, serAccUser)
	if updateErr != nil {
		return nil, errors.GeneralError("failed add attributes to service account user:%v", err)
	}
	serviceAcc.ID = internalClient
	serviceAcc.Name = c.Name
	serviceAcc.ClientID = c.ClientID
	serviceAcc.Description = c.Description
	serviceAcc.ClientSecret = clientSecret
	return &serviceAcc, nil
}

func (kc *keycloakService) buildServiceAccountIdentifier() string {
	return "srvc-acct-" + NewUUID()
}

func (kc *keycloakService) ListServiceAcc(ctx context.Context, first int, max int) ([]api.ServiceAccount, *errors.ServiceError) {
	accessToken, _ := kc.kcClient.GetToken()
	orgId := auth.GetOrgIdFromContext(ctx)
	var sa []api.ServiceAccount
	clients, err := kc.kcClient.GetClients(accessToken, first, max)
	if err != nil {
		return nil, errors.GeneralError("failed to check the sso client exists:%v", err)
	}
	for _, client := range clients {
		acc := api.ServiceAccount{}
		attributes := client.Attributes
		att := *attributes
		if att["rh-org-id"] == orgId && strings.HasPrefix(safeString(client.ClientID), "srvc-acct"){
			acc.ID = *client.ID
			acc.ClientID = *client.ClientID
			acc.Name = safeString(client.Name)
			acc.Description = safeString(client.Description)
			sa = append(sa, acc)
		}
	}
	return sa, nil
}

func (kc *keycloakService) DeleteServiceAccount(ctx context.Context, id string) *errors.ServiceError {
	accessToken, _ := kc.kcClient.GetToken()
	orgId := auth.GetOrgIdFromContext(ctx)
	c, err := kc.kcClient.GetClientById(id, accessToken)
	if err != nil {
		return errors.GeneralError("failed to check the sso client exists:%v", err)
	}
	if kc.kcClient.IsSameOrg(c, orgId) {
		err = kc.kcClient.DeleteClient(id, accessToken)
		if err != nil {
			return errors.FailedToDeleteServiceAccount("failed to delete service account")
		}
		return nil
	} else {
		return errors.Forbidden("can not delete sso client due to permission error")
	}
}

func (kc *keycloakService) ResetServiceAccountCredentials(ctx context.Context, id string) (*api.ServiceAccount, *errors.ServiceError) {
	accessToken, _ := kc.kcClient.GetToken()
	orgId := auth.GetOrgIdFromContext(ctx)
	c, err := kc.kcClient.GetClientById(id, accessToken)
	if err != nil {
		return nil, errors.FailedToGetServiceAccount("failed to check the service account exists")
	}
	if kc.kcClient.IsSameOrg(c, orgId) {
		credRep, error := kc.kcClient.RegenerateClientSecret(accessToken, id)
		if error != nil {
			return nil, errors.GeneralError("failed to regenerate service account secret:%v", err)
		}
		value := *credRep.Value
		return &api.ServiceAccount{
			ID:           *c.ID,
			ClientID:     *c.ClientID,
			ClientSecret: value,
			Name:         safeString(c.Name),
			Description:  safeString(c.Description),
		}, nil
	} else {
		return nil, errors.Forbidden("can not regenerate service account secret due to permission error")
	}
}

func NewUUID() string {
	return uuid.New().String()
}
