package redhatsso

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	serviceaccountsclient "github.com/redhat-developer/app-services-sdk-go/serviceaccounts/apiv1internal/client"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// access token duration before expiration
	tokenLifeDuration    = 5 * time.Minute
	cacheCleanupInterval = 5 * time.Minute
)

//go:generate moq -out client_moq.go . SSOClient
type SSOClient interface {
	GetToken() (string, error)
	GetConfig() *keycloak.KeycloakConfig
	GetRealmConfig() *keycloak.KeycloakRealmConfig
	GetServiceAccounts(accessToken string, first int, max int) ([]serviceaccountsclient.ServiceAccountData, error)
	GetServiceAccount(accessToken string, clientId string) (*serviceaccountsclient.ServiceAccountData, bool, error)
	CreateServiceAccount(accessToken string, name string, description string) (serviceaccountsclient.ServiceAccountData, error)
	DeleteServiceAccount(accessToken string, clientId string) error
	UpdateServiceAccount(accessToken string, clientId string, name string, description string) (serviceaccountsclient.ServiceAccountData, error)
	RegenerateClientSecret(accessToken string, id string) (serviceaccountsclient.ServiceAccountData, error)
}

func NewSSOClient(config *keycloak.KeycloakConfig) SSOClient {
	return &rhSSOClient{
		config: config,
		cache:  cache.New(tokenLifeDuration, cacheCleanupInterval),
	}
}

var _ SSOClient = &rhSSOClient{}

type rhSSOClient struct {
	config        *keycloak.KeycloakConfig
	configuration *serviceaccountsclient.Configuration
	cache         *cache.Cache
}

type tokenResponse struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	TokenType        string `json:"token_type"`
	NotBeforePolicy  int    `json:"not-before-policy"`
	Scope            string `json:"scope"`
}

func (c *rhSSOClient) getConfiguration(accessToken string) *serviceaccountsclient.Configuration {
	if c.configuration == nil {
		c.configuration = &serviceaccountsclient.Configuration{
			DefaultHeader: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				"Content-Type":  "application/json",
			},
			UserAgent: "OpenAPI-Generator/1.0.0/go",
			Debug:     false,
			Servers: serviceaccountsclient.ServerConfigurations{
				{
					URL: c.config.RedhatSSORealm.APIEndpointURI,
				},
			},
		}
	}
	return c.configuration
}

func (c *rhSSOClient) getCachedToken(tokenKey string) (string, error) {
	cachedToken, isCached := c.cache.Get(tokenKey)
	ct, _ := cachedToken.(string)
	if isCached {
		return ct, nil
	}
	return "", errors.Errorf("failed to retrieve cached token")
}

func (c *rhSSOClient) GetToken() (string, error) {
	cachedTokenKey := fmt.Sprintf("%s%s", c.config.RedhatSSORealm.Realm, c.config.RedhatSSORealm.ClientID)
	cachedToken, _ := c.getCachedToken(cachedTokenKey)

	if cachedToken != "" && !shared.IsJWTTokenExpired(cachedToken) {
		return cachedToken, nil
	}

	client := &http.Client{}
	parameters := url.Values{}
	parameters.Set("grant_type", "client_credentials")
	parameters.Set("client_id", c.config.RedhatSSORealm.ClientID)
	parameters.Set("client_secret", c.config.RedhatSSORealm.ClientSecret)
	req, err := http.NewRequest("POST", c.config.RedhatSSORealm.TokenEndpointURI, strings.NewReader(parameters.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(parameters.Encode())))

	resp, err := client.Do(req)

	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error getting token [%d]", resp.StatusCode)
	}

	token, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tokenData tokenResponse
	err = json.Unmarshal(token, &tokenData)
	if err != nil {
		return "", err
	}

	return tokenData.AccessToken, nil
}

func (c *rhSSOClient) GetConfig() *keycloak.KeycloakConfig {
	return c.config
}

func (c *rhSSOClient) GetRealmConfig() *keycloak.KeycloakRealmConfig {
	return c.config.RedhatSSORealm
}

func (c *rhSSOClient) GetServiceAccounts(accessToken string, first int, max int) ([]serviceaccountsclient.ServiceAccountData, error) {
	serviceAccounts, _, err := serviceaccountsclient.NewAPIClient(c.getConfiguration(accessToken)).
		ServiceAccountsApi.GetServiceAccounts(context.Background()).
		Max(int32(max)).
		First(int32(first)).
		Execute()

	return serviceAccounts, err
}

func (c *rhSSOClient) GetServiceAccount(accessToken string, clientId string) (*serviceaccountsclient.ServiceAccountData, bool, error) {
	serviceAccount, resp, err := serviceaccountsclient.NewAPIClient(c.getConfiguration(accessToken)).
		ServiceAccountsApi.GetServiceAccount(context.Background(), clientId).
		Execute()

	if resp.StatusCode == http.StatusNotFound {
		return nil, false, nil
	}
	return &serviceAccount, err == nil, err
}

func (c *rhSSOClient) CreateServiceAccount(accessToken string, name string, description string) (serviceaccountsclient.ServiceAccountData, error) {
	serviceAccount, _, err := serviceaccountsclient.NewAPIClient(c.getConfiguration(accessToken)).
		ServiceAccountsApi.CreateServiceAccount(context.Background()).
		ServiceAccountCreateRequestData(
			serviceaccountsclient.ServiceAccountCreateRequestData{
				Name:        name,
				Description: description,
			}).Execute()
	return serviceAccount, err
}

func (c *rhSSOClient) DeleteServiceAccount(accessToken string, clientId string) error {
	_, err := serviceaccountsclient.NewAPIClient(c.getConfiguration(accessToken)).
		ServiceAccountsApi.DeleteServiceAccount(context.Background(), clientId).
		Execute()
	return err
}

func (c *rhSSOClient) UpdateServiceAccount(accessToken string, clientId string, name string, description string) (serviceaccountsclient.ServiceAccountData, error) {
	data, _, err := serviceaccountsclient.NewAPIClient(c.getConfiguration(accessToken)).
		ServiceAccountsApi.UpdateServiceAccount(context.Background(), clientId).
		ServiceAccountRequestData(serviceaccountsclient.ServiceAccountRequestData{
			Name:        &name,
			Description: &description,
		}).Execute()
	return data, err
}

func (c *rhSSOClient) RegenerateClientSecret(accessToken string, id string) (serviceaccountsclient.ServiceAccountData, error) {
	data, _, err := serviceaccountsclient.NewAPIClient(c.getConfiguration(accessToken)).
		ServiceAccountsApi.
		ResetServiceAccountSecret(context.Background(), id).
		Execute()
	return data, err
}
