package keycloak

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/Nerzal/gocloak/v7"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
)

//go:generate moq -out client_moq.go . KcClient
type KcClient interface {
	CreateClient(client gocloak.Client, accessToken string) (string, error)
	GetToken() (string, error)
	GetClientSecret(internalClientId string, accessToken string) (string, error)
	DeleteClient(internalClientID string, accessToken string) error
	GetClient(clientId string, accessToken string) ([]*gocloak.Client, error)
	IsClientExist(clientId string, accessToken string) (string, error)
	GetConfig() *config.KeycloakConfig
	GetClientById(id string, accessToken string) (*gocloak.Client, error)
	ClientConfig(client ClientRepresentation) gocloak.Client
	CreateProtocolMapperConfig(string) []gocloak.ProtocolMapperRepresentation
	GetClientServiceAccount(accessToken string, internalClient string) (*gocloak.User, error)
	UpdateServiceAccountUser(accessToken string, serviceAccountUser gocloak.User) error
	GetClients(accessToken string) ([]*gocloak.Client, error)
	IsSameOrg(client *gocloak.Client, orgId string) bool
	RegenerateClientSecret(accessToken string, id string) (*gocloak.CredentialRepresentation, error)
}

type ClientRepresentation struct {
	Name                         string
	ClientID                     string
	ServiceAccountsEnabled       bool
	Secret                       string
	StandardFlowEnabled          bool
	Attributes                   map[string]string
	AuthorizationServicesEnabled bool
	ProtocolMappers              []gocloak.ProtocolMapperRepresentation
	Description                  string
}

type kcClient struct {
	kcClient gocloak.GoCloak
	ctx      context.Context
	config   *config.KeycloakConfig
}

var _ KcClient = &kcClient{}

func NewClient(config *config.KeycloakConfig) *kcClient {
	setTokenEndpoints(config)
	client := gocloak.NewClient(config.BaseURL)
	client.RestyClient().SetDebug(config.Debug)
	client.RestyClient().SetTLSClientConfig(&tls.Config{InsecureSkipVerify: config.InsecureSkipVerify})
	return &kcClient{
		kcClient: client,
		ctx:      context.Background(),
		config:   config,
	}
}

func (kc *kcClient) ClientConfig(client ClientRepresentation) gocloak.Client {
	return gocloak.Client{
		Name:                         &client.Name,
		ClientID:                     &client.ClientID,
		ServiceAccountsEnabled:       &client.ServiceAccountsEnabled,
		StandardFlowEnabled:          &client.StandardFlowEnabled,
		Attributes:                   &client.Attributes,
		AuthorizationServicesEnabled: &client.AuthorizationServicesEnabled,
		ProtocolMappers:              &client.ProtocolMappers,
		Description:                  &client.Description,
	}
}

func (kc *kcClient) CreateProtocolMapperConfig(name string) []gocloak.ProtocolMapperRepresentation {
	proto := "openid-connect"
	mapper := "oidc-usermodel-attribute-mapper"
	protocolMapper := []gocloak.ProtocolMapperRepresentation{
		{
			Name:           &name,
			Protocol:       &proto,
			ProtocolMapper: &mapper,
			Config: &map[string]string{
				"access.token.claim":   "true",
				"claim.name":           name,
				"id.token.claim":       "true",
				"jsonType.label":       "String",
				"user.attribute":       name,
				"userinfo.token.claim": "true",
			},
		},
	}
	return protocolMapper
}

func setTokenEndpoints(config *config.KeycloakConfig) {
	config.JwksEndpointURI = config.BaseURL + "/auth/realms/" + config.Realm + "/protocol/openid-connect/certs"
	config.TokenEndpointURI = config.BaseURL + "/auth/realms/" + config.Realm + "/protocol/openid-connect/token"
	config.ValidIssuerURI = config.BaseURL + "/auth/realms/" + config.Realm
}

func (kc *kcClient) CreateClient(client gocloak.Client, accessToken string) (string, error) {
	internalClientID, err := kc.kcClient.CreateClient(kc.ctx, accessToken, kc.config.Realm, client)
	if err != nil {
		return "", fmt.Errorf("failed to create the sso client: %s", err.Error())
	}
	return internalClientID, err
}

func (kc *kcClient) GetClient(clientId string, accessToken string) ([]*gocloak.Client, error) {
	params := gocloak.GetClientsParams{
		ClientID: &clientId,
	}
	client, err := kc.kcClient.GetClients(kc.ctx, accessToken, kc.config.Realm, params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch the sso client: %s", err.Error())
	}
	return client, err
}

func (kc *kcClient) GetToken() (string, error) {
	options := gocloak.TokenOptions{
		ClientID:     &kc.config.ClientID,
		GrantType:    &kc.config.GrantType,
		ClientSecret: &kc.config.ClientSecret,
	}
	tokenResp, err := kc.kcClient.GetToken(kc.ctx, kc.config.Realm, options)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve the token: %s", err.Error())
	}
	return tokenResp.AccessToken, nil
}

func (kc *kcClient) GetClientSecret(internalClientId string, accessToken string) (string, error) {
	resp, err := kc.kcClient.GetClientSecret(kc.ctx, accessToken, kc.config.Realm, internalClientId)
	if err != nil {
		return "", fmt.Errorf("failed to retrive the client secret: %s", err.Error())
	}
	value := *resp.Value
	return value, err
}

func (kc *kcClient) DeleteClient(internalClientID string, accessToken string) error {
	err := kc.kcClient.DeleteClient(kc.ctx, accessToken, kc.config.Realm, internalClientID)
	if err != nil {
		return fmt.Errorf("failed to delete the sso client: %s", err.Error())
	}
	return err
}

func (kc *kcClient) getClient(clientId string, accessToken string) ([]*gocloak.Client, error) {
	params := gocloak.GetClientsParams{
		ClientID: &clientId,
	}
	client, err := kc.kcClient.GetClients(kc.ctx, accessToken, kc.config.Realm, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get the sso clients: %s", err.Error())
	}
	return client, err
}

func (kc *kcClient) GetClientById(id string, accessToken string) (*gocloak.Client, error) {
	client, err := kc.kcClient.GetClient(kc.ctx, accessToken, kc.config.Realm, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get the sso client by id: %s:%s", id, err.Error())
	}
	return client, err
}

func (kc *kcClient) GetConfig() *config.KeycloakConfig {
	return kc.config
}

func (kc *kcClient) IsClientExist(clientId string, accessToken string) (string, error) {
	client, err := kc.getClient(clientId, accessToken)
	var internalClientID string
	if err != nil {
		return internalClientID, fmt.Errorf("failed to get the sso client: %s:%s", clientId, err.Error())
	}
	if len(client) > 0 {
		internalClientID = *client[0].ID
		return internalClientID, nil
	}
	return internalClientID, err
}

func (kc *kcClient) GetClientServiceAccount(accessToken string, internalClient string) (*gocloak.User, error) {
	serviceAccountUser, err := kc.kcClient.GetClientServiceAccount(kc.ctx, accessToken, kc.config.Realm, internalClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get the service account: %s", err.Error())
	}
	return serviceAccountUser, err
}

func (kc *kcClient) UpdateServiceAccountUser(accessToken string, serviceAccountUser gocloak.User) error {
	err := kc.kcClient.UpdateUser(kc.ctx, accessToken, kc.config.Realm, serviceAccountUser)
	if err != nil {
		return fmt.Errorf("failed to update the service account user: %s", err.Error())
	}
	return err
}

func (kc *kcClient) GetClients(accessToken string) ([]*gocloak.Client, error) {
	params := gocloak.GetClientsParams{}
	clients, err := kc.kcClient.GetClients(kc.ctx, accessToken, kc.config.Realm, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get the sso clients: %s", err.Error())
	}
	return clients, err
}

func (kc *kcClient) IsSameOrg(client *gocloak.Client, orgId string) bool {
	if orgId == "" {
		return false
	}
	attributes := *client.Attributes
	return attributes["rh-org-id"] == orgId
}

func (kc *kcClient) RegenerateClientSecret(accessToken string, id string) (*gocloak.CredentialRepresentation, error) {
	credRep, err := kc.kcClient.RegenerateClientSecret(kc.ctx, accessToken, kc.config.Realm, id)
	if err != nil {
		return nil, fmt.Errorf("failed to regenerate client secret: %s:%s", id, err.Error())
	}
	return credRep, err
}
