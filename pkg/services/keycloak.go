package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Nerzal/gocloak/v8"
	"github.com/golang/glog"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/google/uuid"
)

const (
	rhOrgId            = "rh-org-id"
	rhUserId           = "rh-user-id"
	username           = "username"
	created_at         = "created_at"
	clusterId          = "kas-fleetshard-operator-cluster-id"
	connectorClusterId = "connector-fleetshard-operator-cluster-id"
)

//go:generate moq -out keycloakservice_moq.go . KeycloakService
type KeycloakService interface {
	RegisterKafkaClientInSSO(kafkaNamespace string, orgId string) (string, *errors.ServiceError)
	RegisterOSDClusterClientInSSO(clusterId string, clusterOathCallbackURI string) (string, *errors.ServiceError)
	DeRegisterClientInSSO(kafkaNamespace string) *errors.ServiceError
	GetConfig() *config.KeycloakConfig
	GetRealmConfig() *config.KeycloakRealmConfig
	IsKafkaClientExist(clientId string) *errors.ServiceError
	CreateServiceAccount(serviceAccountRequest *api.ServiceAccountRequest, ctx context.Context) (*api.ServiceAccount, *errors.ServiceError)
	DeleteServiceAccount(ctx context.Context, clientId string) *errors.ServiceError
	ResetServiceAccountCredentials(ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError)
	ListServiceAcc(ctx context.Context, first int, max int) ([]api.ServiceAccount, *errors.ServiceError)
	RegisterKasFleetshardOperatorServiceAccount(agentClusterId string, roleName string) (*api.ServiceAccount, *errors.ServiceError)
	DeRegisterKasFleetshardOperatorServiceAccount(agentClusterId string) *errors.ServiceError
	GetServiceAccountById(ctx context.Context, id string) (*api.ServiceAccount, *errors.ServiceError)
	RegisterConnectorFleetshardOperatorServiceAccount(agentClusterId string, roleName string) (*api.ServiceAccount, *errors.ServiceError)
	GetKafkaClientSecret(clientId string) (string, *errors.ServiceError)
}

type keycloakService struct {
	kcClient keycloak.KcClient
}

var _ KeycloakService = &keycloakService{}

func NewKeycloakService(config *config.KeycloakConfig, realmConfig *config.KeycloakRealmConfig) *keycloakService {
	client := keycloak.NewClient(config, realmConfig)
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
		AuthorizationServicesEnabled: false,
		StandardFlowEnabled:          false,
		Attributes:                   rhOrgIdAttributes,
	}
	clientConfig := kc.kcClient.ClientConfig(c)
	internalClient, err := kc.kcClient.CreateClient(clientConfig, accessToken)
	if err != nil {
		return "", errors.FailedToCreateSSOClient("failed to create the sso client: %v", err)
	}
	secretValue, err := kc.kcClient.GetClientSecret(internalClient, accessToken)
	if err != nil {
		return "", errors.FailedToGetSSOClientSecret("failed to get the sso client secret: %v", err)
	}
	return secretValue, nil
}

func (kc *keycloakService) RegisterOSDClusterClientInSSO(clusterId string, clusterOathCallbackURI string) (string, *errors.ServiceError) {
	accessToken, tokenErr := kc.kcClient.GetToken()
	if tokenErr != nil {
		return "", errors.GeneralError("failed to get token for the sso client: %v", tokenErr)
	}

	internalClientId, err := kc.kcClient.IsClientExist(clusterId, accessToken)
	if err != nil {
		return "", errors.GeneralError("failed to check the sso client exists: %v", err)
	}

	if internalClientId != "" {
		secretValue, _ := kc.kcClient.GetClientSecret(internalClientId, accessToken)
		return secretValue, nil
	}

	c := keycloak.ClientRepresentation{
		ClientID:                     clusterId,
		Name:                         clusterId,
		ServiceAccountsEnabled:       false,
		AuthorizationServicesEnabled: false,
		StandardFlowEnabled:          true,
		RedirectURIs:                 &[]string{clusterOathCallbackURI},
	}

	clientConfig := kc.kcClient.ClientConfig(c)
	internalClient, err := kc.kcClient.CreateClient(clientConfig, accessToken)
	if err != nil {
		return "", errors.FailedToCreateSSOClient("failed to create the sso client: %v", err)
	}
	secretValue, err := kc.kcClient.GetClientSecret(internalClient, accessToken)
	if err != nil {
		return "", errors.FailedToGetSSOClientSecret("failed to get the sso client secret: %v", err)
	}

	return secretValue, nil
}

func (kc *keycloakService) DeRegisterClientInSSO(clientId string) *errors.ServiceError {
	accessToken, _ := kc.kcClient.GetToken()
	internalClientID, _ := kc.kcClient.IsClientExist(clientId, accessToken)
	if internalClientID == "" {
		return nil
	}
	err := kc.kcClient.DeleteClient(internalClientID, accessToken)
	if err != nil {
		return errors.FailedToDeleteSSOClient("failed to delete the sso client: %v", err)
	}
	return nil
}

func (kc *keycloakService) GetConfig() *config.KeycloakConfig {
	return kc.kcClient.GetConfig()
}

func (kc *keycloakService) GetRealmConfig() *config.KeycloakRealmConfig {
	return kc.kcClient.GetRealmConfig()
}

func (kc keycloakService) IsKafkaClientExist(clientId string) *errors.ServiceError {
	accessToken, _ := kc.kcClient.GetToken()
	_, err := kc.kcClient.IsClientExist(clientId, accessToken)
	if err != nil {
		return errors.FailedToGetSSOClient("failed to get the sso client: %v", err)
	}
	return nil
}

func (kc keycloakService) GetKafkaClientSecret(clientId string) (string, *errors.ServiceError) {
	accessToken, _ := kc.kcClient.GetToken()
	internalClientID, err := kc.kcClient.IsClientExist(clientId, accessToken)
	if err != nil {
		return "", errors.FailedToGetSSOClient("failed to get the sso client: %v", err)
	}
	clientSecret, err := kc.kcClient.GetClientSecret(internalClientID, accessToken)
	if err != nil {
		return "", errors.FailedToGetSSOClientSecret("failed to get client secret: %v", err)
	}
	return clientSecret, nil
}

func (kc *keycloakService) CreateServiceAccount(serviceAccountRequest *api.ServiceAccountRequest, ctx context.Context) (*api.ServiceAccount, *errors.ServiceError) {
	var serviceAcc api.ServiceAccount
	accessToken, _ := kc.kcClient.GetToken()
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, errors.Unauthenticated("user not authenticated")
	}
	orgId := auth.GetOrgIdFromClaims(claims)
	ownerAccountId := auth.GetAccountIdFromClaims(claims)
	owner := auth.GetUsernameFromClaims(claims)
	createdAt := time.Now().Format(time.RFC3339)
	rhAccountID := map[string][]string{
		rhOrgId:  {orgId},
		rhUserId: {ownerAccountId},
		username: {owner},
	}
	rhOrgIdAttributes := map[string]string{
		rhOrgId:    orgId,
		rhUserId:   ownerAccountId,
		username:   owner,
		created_at: createdAt,
	}
	OrgIdProtocolMapper := kc.kcClient.CreateProtocolMapperConfig(rhOrgId)
	userIdProtocolMapper := kc.kcClient.CreateProtocolMapperConfig(rhUserId)
	userProtocolMapper := kc.kcClient.CreateProtocolMapperConfig(username)
	protocolMapper := append(OrgIdProtocolMapper, userIdProtocolMapper...)
	protocolMapper = append(protocolMapper, userProtocolMapper...)

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
		return nil, errors.FailedToCreateServiceAccount("failed to create the service account: %v", err)
	}
	clientSecret, err := kc.kcClient.GetClientSecret(internalClient, accessToken)
	if err != nil {
		return nil, errors.FailedToGetSSOClientSecret("failed to get service account secret: %v", err)
	}
	serviceAccountUser, err := kc.kcClient.GetClientServiceAccount(accessToken, internalClient)
	if err != nil {
		return nil, errors.FailedToGetServiceAccount("failed fetch the service account user: %v", err)
	}
	serviceAccountUser.Attributes = &rhAccountID
	serAccUser := *serviceAccountUser
	updateErr := kc.kcClient.UpdateServiceAccountUser(accessToken, serAccUser)
	if updateErr != nil {
		return nil, errors.GeneralError("failed add attributes to service account user: %v", updateErr)
	}
	serviceAcc.ID = internalClient
	serviceAcc.Owner = owner
	serviceAcc.Name = c.Name
	serviceAcc.ClientID = c.ClientID
	serviceAcc.Description = c.Description
	serviceAcc.ClientSecret = clientSecret
	if createdAt != "" {
		serviceAcc.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			serviceAcc.CreatedAt = time.Time{}
		}
	}
	return &serviceAcc, nil
}

func (kc *keycloakService) buildServiceAccountIdentifier() string {
	return "srvc-acct-" + NewUUID()
}

func (kc *keycloakService) ListServiceAcc(ctx context.Context, first int, max int) ([]api.ServiceAccount, *errors.ServiceError) {
	accessToken, _ := kc.kcClient.GetToken()
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, errors.Unauthenticated("user not authenticated")
	}
	orgId := auth.GetOrgIdFromClaims(claims)
	var sa []api.ServiceAccount
	clients, err := kc.kcClient.GetClients(accessToken, first, max)
	if err != nil {
		return nil, errors.GeneralError("failed to check the sso client exists: %v", err)
	}
	for _, client := range clients {
		acc := api.ServiceAccount{}
		attributes := client.Attributes
		att := *attributes
		if att["rh-org-id"] == orgId && strings.HasPrefix(safeString(client.ClientID), "srvc-acct") {
			createdAt, err := time.Parse(time.RFC3339, att["created_at"])
			if err != nil {
				createdAt = time.Time{}
			}
			acc.ID = *client.ID
			acc.Owner = att["username"]
			acc.CreatedAt = createdAt
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
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return errors.Unauthenticated("user not authenticated")
	}
	orgId := auth.GetOrgIdFromClaims(claims)
	userId := auth.GetAccountIdFromClaims(claims)
	c, err := kc.kcClient.GetClientById(id, accessToken)
	if err != nil {
		return errors.FailedToGetServiceAccount("failed to check the service account exists: %v", err)
	}
	if kc.kcClient.IsSameOrg(c, orgId) && kc.kcClient.IsOwner(c, userId) {
		err = kc.kcClient.DeleteClient(id, accessToken)
		if err != nil {
			return errors.FailedToDeleteServiceAccount("failed to delete service account: %v", err)
		}
		return nil
	} else {
		return errors.Forbidden("can not delete sso client due to permission error")
	}
}

func (kc *keycloakService) ResetServiceAccountCredentials(ctx context.Context, id string) (*api.ServiceAccount, *errors.ServiceError) {
	accessToken, _ := kc.kcClient.GetToken()
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, errors.Unauthenticated("user not authenticated")
	}
	orgId := auth.GetOrgIdFromClaims(claims)
	userId := auth.GetAccountIdFromClaims(claims)
	owner := auth.GetUsernameFromClaims(claims)
	c, err := kc.kcClient.GetClientById(id, accessToken)
	if err != nil {
		return nil, errors.FailedToGetServiceAccount("failed to check the service account exists: %v", err)
	}
	if kc.kcClient.IsSameOrg(c, orgId) && kc.kcClient.IsOwner(c, userId) {
		credRep, err := kc.kcClient.RegenerateClientSecret(accessToken, id)
		if err != nil {
			return nil, errors.GeneralError("failed to regenerate service account secret: %v", err)
		}
		value := *credRep.Value
		attributes := c.Attributes
		att := *attributes
		createdAt, err := time.Parse(time.RFC3339, att["created_at"])
		if err != nil {
			createdAt = time.Time{}
		}
		return &api.ServiceAccount{
			ID:           *c.ID,
			ClientID:     *c.ClientID,
			CreatedAt:    createdAt,
			Owner:        owner,
			ClientSecret: value,
			Name:         safeString(c.Name),
			Description:  safeString(c.Description),
		}, nil
	} else {
		return nil, errors.Forbidden("can not regenerate service account secret due to permission error")
	}
}

func (kc *keycloakService) GetServiceAccountById(ctx context.Context, id string) (*api.ServiceAccount, *errors.ServiceError) {
	accessToken, _ := kc.kcClient.GetToken()
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, errors.Unauthenticated("user not authenticated")
	}
	orgId := auth.GetOrgIdFromClaims(claims)
	userId := auth.GetAccountIdFromClaims(claims)
	owner := auth.GetUsernameFromClaims(claims)
	c, err := kc.kcClient.GetClientById(id, accessToken)
	if err != nil {
		return nil, errors.FailedToGetServiceAccount("failed to check the service account exists: %v", err)
	}
	attributes := c.Attributes
	att := *attributes
	createdAt, err := time.Parse(time.RFC3339, att["created_at"])
	if err != nil {
		createdAt = time.Time{}
	}
	if kc.kcClient.IsSameOrg(c, orgId) && kc.kcClient.IsOwner(c, userId) {
		return &api.ServiceAccount{
			ID:          *c.ID,
			ClientID:    *c.ClientID,
			CreatedAt:   createdAt,
			Owner:       owner,
			Name:        safeString(c.Name),
			Description: safeString(c.Description),
		}, nil
	} else {
		return nil, errors.FailedToGetServiceAccount("failed to get service account due to permission error")
	}
}

func (kc *keycloakService) RegisterKasFleetshardOperatorServiceAccount(agentClusterId string, roleName string) (*api.ServiceAccount, *errors.ServiceError) {
	serviceAccountId := buildAgentOperatorServiceAccountId(agentClusterId)
	return kc.registerAgentServiceAccount(clusterId, serviceAccountId, agentClusterId, roleName)
}

func (kc *keycloakService) DeRegisterKasFleetshardOperatorServiceAccount(agentClusterId string) *errors.ServiceError {
	accessToken, err := kc.kcClient.GetToken()
	if err != nil {
		return errors.GeneralError("failed to get token: %s", err.Error())
	}
	serviceAccountId := buildAgentOperatorServiceAccountId(agentClusterId)
	err = kc.kcClient.DeleteClient(serviceAccountId, accessToken)
	if err != nil {
		return errors.FailedToDeleteServiceAccount("Failed to delete service account: %s", err.Error())
	}
	return nil
}

func (kc *keycloakService) RegisterConnectorFleetshardOperatorServiceAccount(agentClusterId string, roleName string) (*api.ServiceAccount, *errors.ServiceError) { // (agentClusterId string, roleName string) (*api.ServiceAccount, *errors.ServiceError) {
	serviceAccountId := fmt.Sprintf("connector-fleetshard-agent-%s", agentClusterId)
	return kc.registerAgentServiceAccount(connectorClusterId, serviceAccountId, agentClusterId, roleName)
}

func (kc *keycloakService) registerAgentServiceAccount(clusterId string, serviceAccountId string, agentClusterId string, roleName string) (*api.ServiceAccount, *errors.ServiceError) {

	accessToken, _ := kc.kcClient.GetToken()
	role, err := kc.createRealmRoleIfNotExists(accessToken, roleName)
	if err != nil {
		return nil, err
	}
	protocolMapper := kc.kcClient.CreateProtocolMapperConfig(clusterId)
	c := keycloak.ClientRepresentation{
		ClientID:               serviceAccountId,
		Name:                   serviceAccountId,
		Description:            fmt.Sprintf("service account for agent on cluster %s", agentClusterId),
		ServiceAccountsEnabled: true,
		StandardFlowEnabled:    false,
		ProtocolMappers:        protocolMapper,
		Attributes: map[string]string{
			clusterId: agentClusterId,
		},
	}
	account, err := kc.createServiceAccountIfNotExists(accessToken, c)
	if err != nil {
		return nil, errors.GeneralError("failed to create service account: %v", err)
	}
	serviceAccountUser, getErr := kc.kcClient.GetClientServiceAccount(accessToken, account.ID)
	if getErr != nil {
		return nil, errors.GeneralError("failed to read service account user: %v", getErr)
	}
	if serviceAccountUser.Attributes == nil || !gocloak.UserAttributeContains(*serviceAccountUser.Attributes, clusterId, agentClusterId) {
		glog.V(10).Infof("Client %s has no attribute %s, set it", serviceAccountId, clusterId)
		serviceAccountUser.Attributes = &map[string][]string{
			clusterId: {agentClusterId},
		}
		updateErr := kc.kcClient.UpdateServiceAccountUser(accessToken, *serviceAccountUser)
		if updateErr != nil {
			return nil, errors.GeneralError("failed to update service account user: %v", updateErr)
		}
	}
	glog.V(5).Infof("Attribute %s is added for client %s", clusterId, serviceAccountId)
	hasRole, checkErr := kc.kcClient.UserHasRealmRole(accessToken, *serviceAccountUser.ID, roleName)
	if checkErr != nil {
		return nil, errors.GeneralError("failed to check if user has role: %v", checkErr)
	}
	if hasRole == nil {
		glog.V(10).Infof("Client %s has no role %s, adding", serviceAccountId, roleName)
		addRoleErr := kc.kcClient.AddRealmRoleToUser(accessToken, *serviceAccountUser.ID, *role)
		if addRoleErr != nil {
			return nil, errors.GeneralError("failed to add role to user: %v", addRoleErr)
		}
	}
	glog.V(5).Infof("Client %s created successfully", serviceAccountId)
	return account, nil
}

func (kc *keycloakService) createRealmRoleIfNotExists(token string, roleName string) (*gocloak.Role, *errors.ServiceError) {
	glog.V(5).Infof("Creating realm role %s", roleName)
	role, err := kc.kcClient.GetRealmRole(token, roleName)
	if err != nil {
		return nil, errors.GeneralError("failed to get realm role: %v", err)
	}
	if role == nil {
		glog.V(10).Infof("No existing realm role %s found, creating a new one", roleName)
		role, err = kc.kcClient.CreateRealmRole(token, roleName)
		if err != nil {
			return nil, errors.GeneralError("failed to create realm role: %v", err)
		}
	}
	glog.V(5).Infof("Realm role %s created. id = %s", roleName, *role.ID)
	return role, nil
}

func (kc *keycloakService) createServiceAccountIfNotExists(token string, clientRep keycloak.ClientRepresentation) (*api.ServiceAccount, *errors.ServiceError) {
	glog.V(5).Infof("Creating service account: clientId = %s", clientRep.ClientID)
	client, err := kc.kcClient.GetClient(clientRep.ClientID, token)
	if err != nil {
		return nil, errors.GeneralError("failed to check if client exists: %v", err)
	}
	var internalClientId, clientSecret string
	if client == nil {
		glog.V(10).Infof("No exiting client found for %s, creating a new one", clientRep.ClientID)
		clientConfig := kc.kcClient.ClientConfig(clientRep)
		internalClientId, err = kc.kcClient.CreateClient(clientConfig, token)
		if err != nil {
			return nil, errors.FailedToCreateServiceAccount("failed to create the service account: %v", err)
		}
		clientSecret, err = kc.kcClient.GetClientSecret(internalClientId, token)
		if err != nil {
			return nil, errors.FailedToGetSSOClientSecret("failed to get service account secret: %v", err)
		}
	} else {
		glog.V(10).Infof("Existing client found for %s, internalId=%s", clientRep.ClientID, *client.ID)
		internalClientId = *client.ID
		clientSecret, err = kc.kcClient.GetClientSecret(internalClientId, token)
		if err != nil {
			return nil, errors.FailedToGetSSOClientSecret("failed to get service account secret: %v", err)
		}
	}
	serviceAcc := &api.ServiceAccount{
		ID:           internalClientId,
		ClientID:     clientRep.ClientID,
		ClientSecret: clientSecret,
		Name:         clientRep.Name,
		Description:  clientRep.Description,
	}
	return serviceAcc, nil
}

func buildAgentOperatorServiceAccountId(agentClusterId string) string {
	return fmt.Sprintf("kas-fleetshard-agent-%s", agentClusterId)
}

func NewUUID() string {
	return uuid.New().String()
}
