package services

import (
	"context"
	"crypto/tls"
	"github.com/Nerzal/gocloak/v7"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
)

//go:generate moq -out keycloakservice_moq.go . KeycloakService
type KeycloakService interface {
	CreateClient(client gocloak.Client, accessToken string) (string, *errors.ServiceError)
	GetToken() (string, error)
	GetClientSecret(id string, accessToken string) (string, *errors.ServiceError)
	ClientConfig(client ClientRepresentation) gocloak.Client
	DeleteClient(id string, accessToken string) *errors.ServiceError
	RegisterKafkaClientInSSO(kafkaNamespace string) (string, *errors.ServiceError)
	DeRegisterKafkaClientInSSO(kafkaNamespace string) *errors.ServiceError
	GetClients(clientId string, accessToken string) ([]*gocloak.Client, *errors.ServiceError)
	GetConfig() *config.KeycloakConfig
}

type keycloakService struct {
	kcClient gocloak.GoCloak
	ctx      context.Context
	config   *config.KeycloakConfig
}

type ClientRepresentation struct {
	Name                   string
	ClientID               string
	ServiceAccountsEnabled bool
	Secret                 string
	StandardFlowEnabled    bool
}

var _ KeycloakService = &keycloakService{}

func NewKeycloakService(config *config.KeycloakConfig) *keycloakService {
	client := gocloak.NewClient(config.BaseURL)
	client.RestyClient().SetDebug(config.Debug)
	client.RestyClient().SetTLSClientConfig(&tls.Config{InsecureSkipVerify: config.InsecureSkipVerify})
	return &keycloakService{
		kcClient: client,
		ctx:      context.Background(),
		config:   config,
	}
}

//use a token from a Service Account with only the create-client role
func (kc *keycloakService) CreateClient(client gocloak.Client, accessToken string) (string, *errors.ServiceError) {
	internalClientID, err := kc.kcClient.CreateClient(kc.ctx, accessToken, kc.config.Realm, client)
	if err != nil {
		return "", errors.GeneralError("Failed to create a client: %v", err)
	}
	return internalClientID, nil
}

//internal client name is required to fetch the secrets
func (kc *keycloakService) GetClientSecret(internalClientID string, accessToken string) (string, *errors.ServiceError) {
	resp, err := kc.kcClient.GetClientSecret(kc.ctx, accessToken, kc.config.Realm, internalClientID)
	if err != nil {
		return "", errors.GeneralError("Failed to retrieve client secret:%v", err)
	}
	value := *resp.Value
	return value, nil
}

func (kc *keycloakService) GetToken() (string, error) {
	options := gocloak.TokenOptions{
		ClientID:     &kc.config.ClientID,
		GrantType:    &kc.config.GrantType,
		ClientSecret: &kc.config.ClientSecret,
	}
	tokenResp, err := kc.kcClient.GetToken(kc.ctx, kc.config.Realm, options)
	if err != nil {
		return "", errors.GeneralError("failed to retrieve the token:%v", err)
	}
	return tokenResp.AccessToken, nil
}

func (kc *keycloakService) RegisterKafkaClientInSSO(kafkaClusterName string) (string, *errors.ServiceError) {
	accessToken, _ := kc.GetToken()
	id, err := kc.isClientExist(kafkaClusterName, accessToken)
	if err != nil {
		return "", errors.GeneralError("failed to check the sso client exists:%v", err)
	}
	if id != "" {
		secretValue, _ := kc.GetClientSecret(id, accessToken)
		return secretValue, nil
	}
	c := ClientRepresentation{
		ClientID:               kafkaClusterName,
		Name:                   kafkaClusterName,
		ServiceAccountsEnabled: true,
		StandardFlowEnabled:    false,
	}
	clientConfig := kc.ClientConfig(c)
	internalClient, err := kc.CreateClient(clientConfig, accessToken)
	if err != nil {
		return "", errors.GeneralError("failed to create the sso client:%v", err)
	}
	secretValue, err := kc.GetClientSecret(internalClient, accessToken)
	if err != nil {
		return "", errors.GeneralError("failed to get the client secret:%v", err)
	}
	return secretValue, err
}

func (kc *keycloakService) DeRegisterKafkaClientInSSO(kafkaClusterName string) *errors.ServiceError {
	accessToken, _ := kc.GetToken()
	id, err := kc.isClientExist(kafkaClusterName, accessToken)
	if err != nil {
		return errors.GeneralError("failed to check the sso client exists:%v", err)
	}
	err = kc.DeleteClient(id, accessToken)
	if err != nil {
		return errors.GeneralError("failed to delete the sso client:%v", err)
	}
	return nil
}

func (kc *keycloakService) ClientConfig(client ClientRepresentation) gocloak.Client {
	return gocloak.Client{
		Name:                   &client.Name,
		ClientID:               &client.ClientID,
		ServiceAccountsEnabled: &client.ServiceAccountsEnabled,
		StandardFlowEnabled:    &client.StandardFlowEnabled,
	}
}

func (kc *keycloakService) DeleteClient(internalClientID string, accessToken string) *errors.ServiceError {
	err := kc.kcClient.DeleteClient(kc.ctx, accessToken, kc.config.Realm, internalClientID)
	if err != nil {
		return errors.GeneralError("Failed to delete client:%v", err)
	}
	return nil
}

func (kc *keycloakService) GetClients(clientId string, accessToken string) ([]*gocloak.Client, *errors.ServiceError) {
	params := gocloak.GetClientsParams{
		ClientID: &clientId,
	}
	client, err := kc.kcClient.GetClients(kc.ctx, accessToken, kc.config.Realm, params)
	if err != nil {
		return nil, errors.GeneralError("Failed to get client:%v", err)
	}
	return client, nil
}

func (kc *keycloakService) GetConfig() *config.KeycloakConfig {
	return kc.config
}

func (kc *keycloakService) isClientExist(clientId string, accessToken string) (string, *errors.ServiceError) {
	clients, err := kc.GetClients(clientId, accessToken)
	var id string
	if err != nil {
		return id, errors.GeneralError("failed to get client:%v", err)
	}
	if len(clients) > 0 {
		for _, client := range clients {
			id = *client.ID
			return id, nil
		}
	}
	return id, nil
}
