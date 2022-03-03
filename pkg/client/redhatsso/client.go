package redhatsso

import (
	"context"
	"encoding/json"
	"fmt"
	serviceaccountsclient "github.com/redhat-developer/app-services-sdk-go/serviceaccounts/apiv1internal/client"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type SSOClient interface {
	GetToken() (string, error)
	GetServiceAccounts(accessToken string, first int, max int) ([]serviceaccountsclient.ServiceAccountData, error)
	GetServiceAccount(clientId string, accessToken string) (serviceaccountsclient.ServiceAccountData, error)
	CreateServiceAccount(accessToken string, name string, description string) (serviceaccountsclient.ServiceAccountData, error)
	DeleteServiceAccount(clientId string, accessToken string) error
	UpdateServiceAccount(clientId string, name string, description string, accessToken string) (serviceaccountsclient.ServiceAccountData, error)
	RegenerateClientSecret(accessToken string, id string) (serviceaccountsclient.ServiceAccountData, error)
}

func NewSSOClient(config *RedhatSSOConfig) SSOClient {
	return &rhSSOClient{
		config: config,
	}
}

var _ SSOClient = &rhSSOClient{}

type rhSSOClient struct {
	config        *RedhatSSOConfig
	configuration *serviceaccountsclient.Configuration
}

type tokenResponse struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	TokenType        string `json:"token_type"`
	NotBeforePolicy  int    `json:"not-before-policy"`
	Scope            string `json:"scope"`
}

func (kc *rhSSOClient) getConfiguration(accessToken string) *serviceaccountsclient.Configuration {
	if kc.configuration == nil {
		kc.configuration = &serviceaccountsclient.Configuration{
			DefaultHeader: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				"Content-Type":  "application/json",
			},
			UserAgent: "OpenAPI-Generator/1.0.0/go",
			Debug:     false,
			Servers: serviceaccountsclient.ServerConfigurations{
				{
					URL: kc.config.KafkaRealm.APIEndpointURI,
				},
			},
		}
	}
	return kc.configuration
}

func (kc *rhSSOClient) GetToken() (string, error) {
	client := &http.Client{}
	parameters := url.Values{}
	parameters.Set("grant_type", "client_credentials")
	parameters.Set("client_id", kc.config.KafkaRealm.ClientID)
	parameters.Set("client_secret", kc.config.KafkaRealm.ClientSecret)
	req, err := http.NewRequest("POST", kc.config.KafkaRealm.TokenEndpointURI, strings.NewReader(parameters.Encode()))
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
		return "", fmt.Errorf("%d error getting token", resp.StatusCode)
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

func (kc *rhSSOClient) GetServiceAccounts(accessToken string, first int, max int) ([]serviceaccountsclient.ServiceAccountData, error) {
	serviceAccounts, _, err := serviceaccountsclient.NewAPIClient(kc.getConfiguration(accessToken)).
		ServiceAccountsApi.GetServiceAccounts(context.Background()).
		Max(int32(max)).
		First(int32(first)).
		Execute()
	return serviceAccounts, err
}

func (kc *rhSSOClient) GetServiceAccount(clientId string, accessToken string) (serviceaccountsclient.ServiceAccountData, error) {
	serviceAccount, _, err := serviceaccountsclient.NewAPIClient(kc.getConfiguration(accessToken)).
		ServiceaccountsApi.GetServiceAccount(context.Background(), clientId).
		Execute()
	return serviceAccount, err
}

func (kc *rhSSOClient) CreateServiceAccount(accessToken string, name string, description string) (serviceaccountsclient.ServiceAccountData, error) {
	serviceAccount, _, err := serviceaccountsclient.NewAPIClient(kc.getConfiguration(accessToken)).
		ServiceAccountsApi.CreateServiceAccount(context.Background()).
		ServiceAccountCreateRequestData(
			serviceaccountsclient.ServiceAccountCreateRequestData{
				Name:        name,
				Description: description,
			}).Execute()
	return serviceAccount, err
}

func (kc *rhSSOClient) DeleteServiceAccount(clientId string, accessToken string) error {
	_, err := serviceaccountsclient.NewAPIClient(kc.getConfiguration(accessToken)).
		ServiceAccountsApi.DeleteServiceAccount(context.Background(), clientId).
		Execute()
	return err
}

func (kc *rhSSOClient) UpdateServiceAccount(clientId string, name string, description string, accessToken string) (serviceaccountsclient.ServiceAccountData, error) {
	data, _, err := serviceaccountsclient.NewAPIClient(kc.getConfiguration(accessToken)).
		ServiceAccountsApi.UpdateServiceAccount(context.Background(), clientId).
		ServiceAccountRequestData(serviceaccountsclient.ServiceAccountRequestData{
			Name:        &name,
			Description: &description,
		}).Execute()
	return data, err
}

func (kc *rhSSOClient) RegenerateClientSecret(accessToken string, id string) (serviceaccountsclient.ServiceAccountData, error) {
	data, _, err := serviceaccountsclient.NewAPIClient(kc.getConfiguration(accessToken)).
		ServiceaccountsApi.
		ResetServiceAccountSecret(context.Background(), id).
		Execute()
	return data, err
}
