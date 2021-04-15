package services

import (
	"context"
	"fmt"
	"github.com/getsentry/sentry-go"
	"net/http"
	"strings"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/gocloak/v8"
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
	accessToken, tokenErr := kc.kcClient.GetToken()
	if tokenErr != nil {
		sentry.CaptureException(tokenErr)
		return "", errors.GeneralError(tokenErr.Error())
	}
	internalClientId, err := kc.kcClient.IsClientExist(kafkaClusterName, accessToken)
	if err != nil {
		sentry.CaptureException(err)
		return "", handleKeyCloakCheckClientIdExistence(err, kafkaClusterName)
	}
	if internalClientId != "" {
		glog.V(5).Infof("Existing Kafka Client %s found", kafkaClusterName)
		secretValue, secretErr := kc.kcClient.GetClientSecret(internalClientId, accessToken)
		if secretErr != nil {
			sentry.CaptureException(secretErr)
			return "", errors.GeneralError(secretErr.Error())
		}
		return secretValue, nil
	}
	rhOrgIdAttributes := map[string]string{
		"owner-rh-org-id": orgId,
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
		sentry.CaptureException(err)
		return "", errors.FailedToCreateSSOClient(err.Error())
	}
	secretValue, err := kc.kcClient.GetClientSecret(internalClient, accessToken)
	if err != nil {
		sentry.CaptureException(err)
		return "", errors.FailedToGetSSOClientSecret(err.Error())
	}
	glog.V(5).Infof("Kafka Client %s created successfully", kafkaClusterName)
	return secretValue, nil
}

func (kc *keycloakService) RegisterOSDClusterClientInSSO(clusterId string, clusterOathCallbackURI string) (string, *errors.ServiceError) {
	accessToken, tokenErr := kc.kcClient.GetToken()
	if tokenErr != nil {
		sentry.CaptureException(tokenErr)
		return "", errors.GeneralError(tokenErr.Error())
	}

	internalClientId, err := kc.kcClient.IsClientExist(clusterId, accessToken)
	if err != nil {
		sentry.CaptureException(err)
		return "", handleKeyCloakCheckClientIdExistence(err, clusterId)
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
		sentry.CaptureException(err)
		return "", errors.FailedToCreateSSOClient(err.Error())
	}
	secretValue, err := kc.kcClient.GetClientSecret(internalClient, accessToken)
	if err != nil {
		sentry.CaptureException(err)
		return "", errors.FailedToGetSSOClientSecret(err.Error())
	}

	return secretValue, nil
}

func (kc *keycloakService) DeRegisterClientInSSO(clientId string) *errors.ServiceError {
	accessToken, tokenErr := kc.kcClient.GetToken()
	if tokenErr != nil {
		sentry.CaptureException(tokenErr)
		return errors.GeneralError(tokenErr.Error())
	}
	internalClientID, _ := kc.kcClient.IsClientExist(clientId, accessToken)
	glog.V(5).Infof("Existing Kafka Client %s found", clientId)
	if internalClientID == "" {
		return nil
	}
	err := kc.kcClient.DeleteClient(internalClientID, accessToken)
	if err != nil {
		sentry.CaptureException(err)
		return errors.FailedToDeleteSSOClient(err.Error())
	}
	glog.V(5).Infof("Kafka Client %s deleted successfully", clientId)
	return nil
}

func (kc *keycloakService) GetConfig() *config.KeycloakConfig {
	return kc.kcClient.GetConfig()
}

func (kc *keycloakService) GetRealmConfig() *config.KeycloakRealmConfig {
	return kc.kcClient.GetRealmConfig()
}

func (kc keycloakService) IsKafkaClientExist(clientId string) *errors.ServiceError {
	accessToken, tokenErr := kc.kcClient.GetToken()
	if tokenErr != nil {
		sentry.CaptureException(tokenErr)
		return errors.GeneralError(tokenErr.Error())
	}
	_, err := kc.kcClient.IsClientExist(clientId, accessToken)
	if err != nil {
		sentry.CaptureException(err)
		return handleKeyCloakCheckClientIdExistence(err, clientId)
	}
	return nil
}

func (kc keycloakService) GetKafkaClientSecret(clientId string) (string, *errors.ServiceError) {
	accessToken, tokenErr := kc.kcClient.GetToken()
	if tokenErr != nil {
		sentry.CaptureException(tokenErr)
		return "", errors.GeneralError(tokenErr.Error())
	}
	internalClientID, err := kc.kcClient.IsClientExist(clientId, accessToken)
	if err != nil {
		sentry.CaptureException(err)
		return "", handleKeyCloakCheckClientIdExistence(err, clientId)
	}
	clientSecret, err := kc.kcClient.GetClientSecret(internalClientID, accessToken)
	if err != nil {
		sentry.CaptureException(err)
		return "", errors.FailedToGetSSOClientSecret(err.Error())
	}
	return clientSecret, nil
}

func (kc *keycloakService) CreateServiceAccount(serviceAccountRequest *api.ServiceAccountRequest, ctx context.Context) (*api.ServiceAccount, *errors.ServiceError) {
	var serviceAcc api.ServiceAccount
	accessToken, tokenErr := kc.kcClient.GetToken()
	if tokenErr != nil {
		sentry.CaptureException(tokenErr)
		return nil, errors.GeneralError("failed to create service account")
	}
	claims, err := auth.GetClaimsFromContext(ctx) //http requester's info
	if err != nil {
		sentry.CaptureException(err)
		return nil, errors.Unauthenticated("user not authenticated")
	}
	orgId := auth.GetOrgIdFromClaims(claims)
	ownerAccountId := auth.GetAccountIdFromClaims(claims)
	owner := auth.GetUsernameFromClaims(claims)
	isAllowed, err := kc.checkAllowedServiceAccountsLimits(accessToken, kc.GetConfig().MaxAllowedServiceAccounts, ownerAccountId)
	if err != nil { //5xx
		sentry.CaptureException(err)
		return nil, errors.GeneralError("failed to create service account")
	}
	if !isAllowed { //4xx over requesters' limit
		return nil, errors.Forbidden("Max allowed number:%d of service accounts for user:%s has reached", kc.GetConfig().MaxAllowedServiceAccounts, owner)
	}
	glog.V(5).Infof("creating service accounts: user = %s", owner)
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
	//step 1
	internalClient, err := kc.kcClient.CreateClient(clientConfig, accessToken)
	if err != nil { //5xx
		sentry.CaptureException(err)
		return nil, errors.FailedToCreateServiceAccount("failed to create the service account")
	}
	clientSecret, err := kc.kcClient.GetClientSecret(internalClient, accessToken)
	if err != nil { //5xx
		sentry.CaptureException(err)
		return nil, errors.FailedToGetSSOClientSecret("failed to get service account secret")
	}
	serviceAccountUser, err := kc.kcClient.GetClientServiceAccount(accessToken, internalClient)
	if err != nil { //5xx
		sentry.CaptureException(err)
		return nil, errors.FailedToGetServiceAccount("failed to fetch the service account")
	}
	serviceAccountUser.Attributes = &rhAccountID
	serAccUser := *serviceAccountUser
	//step 2
	updateErr := kc.kcClient.UpdateServiceAccountUser(accessToken, serAccUser)
	if updateErr != nil { //5xx
		sentry.CaptureException(updateErr)
		return nil, errors.FailedToCreateServiceAccount("failed to create the service account")
	}
	serviceAcc.ID = internalClient
	serviceAcc.Owner = owner
	serviceAcc.Name = c.Name
	serviceAcc.ClientID = c.ClientID
	serviceAcc.Description = c.Description
	serviceAcc.ClientSecret = clientSecret
	serviceAcc.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		serviceAcc.CreatedAt = time.Time{}
	}
	glog.V(5).Infof("service account clientId = %s created for user = %s", owner, serviceAcc.ClientID)
	return &serviceAcc, nil
}

func (kc *keycloakService) buildServiceAccountIdentifier() string {
	return "srvc-acct-" + NewUUID()
}

func (kc *keycloakService) ListServiceAcc(ctx context.Context, first int, max int) ([]api.ServiceAccount, *errors.ServiceError) {
	accessToken, tokenErr := kc.kcClient.GetToken()
	if tokenErr != nil { //5xx
		sentry.CaptureException(tokenErr)
		return nil, errors.GeneralError("failed to list service accounts")
	}

	claims, err := auth.GetClaimsFromContext(ctx) //http requester's info
	if err != nil {                               //4xx
		sentry.CaptureException(err)
		return nil, errors.Unauthenticated("user not authenticated")
	}
	orgId := auth.GetOrgIdFromClaims(claims)
	searchAtt := fmt.Sprintf("rh-org-id:%s", orgId)
	clients, err := kc.kcClient.GetClients(accessToken, first, max, searchAtt)
	if err != nil { //5xx
		sentry.CaptureException(err)
		return nil, errors.GeneralError("failed to collect service accounts")
	}

	var sa []api.ServiceAccount
	for _, client := range clients {
		acc := api.ServiceAccount{}
		attributes := client.Attributes
		att := *attributes
		if strings.HasPrefix(safeString(client.ClientID), "srvc-acct") {
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
	accessToken, tokenErr := kc.kcClient.GetToken()
	if tokenErr != nil { //5xx
		sentry.CaptureException(tokenErr)
		return errors.GeneralError("failed to delete service account")
	}
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil { //4xx
		sentry.CaptureException(err)
		return errors.Unauthenticated("user not authenticated")
	}
	//get service account info with keycloak service client id token
	c, err := kc.kcClient.GetClientById(id, accessToken)
	if err != nil { //5xx or 4xx
		sentry.CaptureException(err)
		return handleKeyCloakGetClientError(err, id)
	}
	orgId := auth.GetOrgIdFromClaims(claims)
	userId := auth.GetAccountIdFromClaims(claims)
	if kc.kcClient.IsSameOrg(c, orgId) && kc.kcClient.IsOwner(c, userId) {
		err = kc.kcClient.DeleteClient(id, accessToken) //id existence checked
		if err != nil {                                 //5xx
			sentry.CaptureException(err)
			return errors.FailedToDeleteServiceAccount("failed to delete service account")
		}
		return nil
	} else { //4xx
		sentry.CaptureException(err)
		return errors.Forbidden(errors.ErrorForbiddenReason) //only give a general reason
	}
}

func (kc *keycloakService) ResetServiceAccountCredentials(ctx context.Context, id string) (*api.ServiceAccount, *errors.ServiceError) {
	accessToken, tokenErr := kc.kcClient.GetToken()
	if tokenErr != nil { //5xx
		sentry.CaptureException(tokenErr)
		return nil, errors.GeneralError("failed to reset service account credentials")
	}
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil { //4xx
		sentry.CaptureException(err)
		return nil, errors.Unauthenticated("user not authenticated")
	}
	c, err := kc.kcClient.GetClientById(id, accessToken)
	if err != nil { //5xx or 4xx
		sentry.CaptureException(err)
		return nil, handleKeyCloakGetClientError(err, id)
	}
	//http request's info
	orgId := auth.GetOrgIdFromClaims(claims)
	userId := auth.GetAccountIdFromClaims(claims)
	owner := auth.GetUsernameFromClaims(claims)
	if kc.kcClient.IsSameOrg(c, orgId) && kc.kcClient.IsOwner(c, userId) {
		credRep, err := kc.kcClient.RegenerateClientSecret(accessToken, id)
		if err != nil { //5xx
			sentry.CaptureException(err)
			return nil, errors.GeneralError("failed to reset service account credentials.")
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
	} else { //4xx
		return nil, errors.Forbidden(errors.ErrorForbiddenReason) //only give general reason
	}
}

// for internal client id check only
func handleKeyCloakCheckClientIdExistence(err error, clientId string) *errors.ServiceError {
	if keyErr, ok := err.(*gocloak.APIError); ok {
		if keyErr.Code == http.StatusNotFound {
			return errors.SSOClientNotFound(fmt.Sprintf("client id not found %s", clientId))
		}
	}
	return errors.FailedToGetSSOClient("failed to get sso client with id: %s:%+v", clientId, err.Error())
}

// return error object for API caller facing funcs: 5xx or 4xx
func handleKeyCloakGetClientError(err error, id string) *errors.ServiceError {
	if keyErr, ok := err.(*gocloak.APIError); ok {
		if keyErr.Code == http.StatusNotFound {
			return errors.ServiceAccountNotFound(fmt.Sprintf("service account not found %s", id))
		}
	}
	return errors.FailedToGetServiceAccount("failed to get the service account by id %s", id)
}

func (kc *keycloakService) GetServiceAccountById(ctx context.Context, id string) (*api.ServiceAccount, *errors.ServiceError) {
	accessToken, tokenErr := kc.kcClient.GetToken() //get keycloak service client id token
	if tokenErr != nil {
		sentry.CaptureException(tokenErr)
		return nil, errors.GeneralError("failed to get service account by id")
	}
	claims, err := auth.GetClaimsFromContext(ctx) //gather http requester info.
	if err != nil {
		sentry.CaptureException(err)
		return nil, errors.Unauthenticated("user not authenticated")
	}
	//get service account info with keycloak service client id token
	c, err := kc.kcClient.GetClientById(id, accessToken)
	if err != nil { //5xx or 4xx
		sentry.CaptureException(err)
		return nil, handleKeyCloakGetClientError(err, id)
	}
	//http requester's info.
	orgId := auth.GetOrgIdFromClaims(claims)
	userId := auth.GetAccountIdFromClaims(claims)
	owner := auth.GetUsernameFromClaims(claims)
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
		//http requester doesn't have the permission: 4xx
		return nil, errors.Forbidden(errors.ErrorForbiddenReason)
	}
}

func (kc *keycloakService) RegisterKasFleetshardOperatorServiceAccount(agentClusterId string, roleName string) (*api.ServiceAccount, *errors.ServiceError) {
	serviceAccountId := buildAgentOperatorServiceAccountId(agentClusterId)
	return kc.registerAgentServiceAccount(clusterId, serviceAccountId, agentClusterId, roleName)
}

func (kc *keycloakService) DeRegisterKasFleetshardOperatorServiceAccount(agentClusterId string) *errors.ServiceError {
	accessToken, err := kc.kcClient.GetToken()
	if err != nil {
		sentry.CaptureException(err)
		return errors.GeneralError("%s", err.Error())
	}
	serviceAccountId := buildAgentOperatorServiceAccountId(agentClusterId)
	internalServiceAccountId, err := kc.kcClient.IsClientExist(serviceAccountId, accessToken)
	if err != nil { //5xx
		sentry.CaptureException(err)
		return handleKeyCloakCheckClientIdExistence(err, serviceAccountId)
	}
	if internalServiceAccountId == "" {
		return nil
	}
	err = kc.kcClient.DeleteClient(internalServiceAccountId, accessToken)
	if err != nil {
		sentry.CaptureException(err)
		return errors.FailedToDeleteServiceAccount(err.Error())
	}
	return nil
}

func (kc *keycloakService) RegisterConnectorFleetshardOperatorServiceAccount(agentClusterId string, roleName string) (*api.ServiceAccount, *errors.ServiceError) { // (agentClusterId string, roleName string) (*api.ServiceAccount, *errors.ServiceError) {
	serviceAccountId := fmt.Sprintf("connector-fleetshard-agent-%s", agentClusterId)
	return kc.registerAgentServiceAccount(connectorClusterId, serviceAccountId, agentClusterId, roleName)
}

func (kc *keycloakService) registerAgentServiceAccount(clusterId string, serviceAccountId string, agentClusterId string, roleName string) (*api.ServiceAccount, *errors.ServiceError) {
	//get keycloak service client id token
	accessToken, tokenErr := kc.kcClient.GetToken()
	if tokenErr != nil {
		sentry.CaptureException(tokenErr)
		return nil, errors.GeneralError("register agent service account: %v", tokenErr)
	}
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
		return nil, errors.GeneralError(err.Error())
	}
	serviceAccountUser, getErr := kc.kcClient.GetClientServiceAccount(accessToken, account.ID)
	if getErr != nil {
		sentry.CaptureException(getErr)
		return nil, errors.GeneralError(getErr.Error())
	}
	if serviceAccountUser.Attributes == nil || !gocloak.UserAttributeContains(*serviceAccountUser.Attributes, clusterId, agentClusterId) {
		glog.V(10).Infof("Client %s has no attribute %s, set it", serviceAccountId, clusterId)
		serviceAccountUser.Attributes = &map[string][]string{
			clusterId: {agentClusterId},
		}
		updateErr := kc.kcClient.UpdateServiceAccountUser(accessToken, *serviceAccountUser)
		if updateErr != nil {
			sentry.CaptureException(updateErr)
			return nil, errors.GeneralError(updateErr.Error())
		}
	}
	glog.V(5).Infof("Attribute %s is added for client %s", clusterId, serviceAccountId)
	hasRole, checkErr := kc.kcClient.UserHasRealmRole(accessToken, *serviceAccountUser.ID, roleName)
	if checkErr != nil {
		sentry.CaptureException(checkErr)
		return nil, errors.GeneralError(checkErr.Error())
	}
	if hasRole == nil {
		glog.V(10).Infof("Client %s has no role %s, adding", serviceAccountId, roleName)
		addRoleErr := kc.kcClient.AddRealmRoleToUser(accessToken, *serviceAccountUser.ID, *role)
		if addRoleErr != nil {
			sentry.CaptureException(addRoleErr)
			return nil, errors.GeneralError(addRoleErr.Error())
		}
	}
	glog.V(5).Infof("Client %s created successfully", serviceAccountId)
	return account, nil
}

func (kc *keycloakService) createRealmRoleIfNotExists(token string, roleName string) (*gocloak.Role, *errors.ServiceError) {
	glog.V(5).Infof("Creating realm role %s", roleName)
	role, err := kc.kcClient.GetRealmRole(token, roleName)
	if err != nil {
		sentry.CaptureException(err)
		return nil, errors.GeneralError(err.Error())
	}
	if role == nil {
		glog.V(10).Infof("No existing realm role %s found, creating a new one", roleName)
		role, err = kc.kcClient.CreateRealmRole(token, roleName)
		if err != nil {
			sentry.CaptureException(err)
			return nil, errors.GeneralError(err.Error())
		}
	}
	glog.V(5).Infof("Realm role %s created. id = %s", roleName, *role.ID)
	return role, nil
}

func (kc *keycloakService) createServiceAccountIfNotExists(token string, clientRep keycloak.ClientRepresentation) (*api.ServiceAccount, *errors.ServiceError) {
	glog.V(5).Infof("Creating service account: clientId = %s", clientRep.ClientID)
	client, err := kc.kcClient.GetClient(clientRep.ClientID, token)
	if err != nil { //5xx
		sentry.CaptureException(err)
		return nil, errors.GeneralError(err.Error())
	}

	//client exists
	var internalClientId, clientSecret string
	if client == nil {
		glog.V(10).Infof("No exiting client found for %s, creating a new one", clientRep.ClientID)
		clientConfig := kc.kcClient.ClientConfig(clientRep)
		internalClientId, err = kc.kcClient.CreateClient(clientConfig, token)
		if err != nil { //5xx
			sentry.CaptureException(err)
			return nil, errors.FailedToCreateServiceAccount(err.Error())
		}
	} else {
		glog.V(10).Infof("Existing client found for %s, internalId=%s", clientRep.ClientID, *client.ID)
		internalClientId = *client.ID
	}

	clientSecret, err = kc.kcClient.GetClientSecret(internalClientId, token)
	if err != nil { //5xx
		sentry.CaptureException(err)
		return nil, errors.FailedToGetSSOClientSecret(err.Error())
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

func (kc *keycloakService) checkAllowedServiceAccountsLimits(accessToken string, maxAllowed int, userId string) (bool, error) {
	glog.V(5).Infof("Check if user is allowed to create service accounts: userId = %s", userId)
	searchAtt := fmt.Sprintf("rh-user-id:%s", userId)
	clients, err := kc.kcClient.GetClients(accessToken, 0, maxAllowed, searchAtt)
	if err != nil {
		return false, err
	}
	numOfClientsFound := len(clients) + 1
	glog.V(10).Infof("Existing number of clients found: %d & max allowed: %d, for the userId:%s", numOfClientsFound, maxAllowed, userId)
	if numOfClientsFound > maxAllowed {
		return false, nil //http requester's error
	} else {
		return true, nil
	}
}

func buildAgentOperatorServiceAccountId(agentClusterId string) string {
	return fmt.Sprintf("kas-fleetshard-agent-%s", agentClusterId)
}

func NewUUID() string {
	return uuid.New().String()
}
