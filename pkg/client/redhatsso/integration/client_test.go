package integration

import (
	"fmt"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/redhatsso"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/onsi/gomega"
	serviceaccountsclient "github.com/redhat-developer/app-services-sdk-go/serviceaccounts/apiv1internal/client"
)

func getClient(baseURL string) redhatsso.SSOClient {
	config := keycloak.KeycloakConfig{
		SsoBaseUrl: baseURL,
		RedhatSSORealm: &keycloak.KeycloakRealmConfig{
			Realm:            "redhat-external",
			APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", baseURL),
			TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", baseURL),
		},
	}

	return redhatsso.NewSSOClient(&config, config.RedhatSSORealm)
}

func Test_SSOClient_GetServiceAccounts(t *testing.T) {
	g := gomega.NewWithT(t)

	server := mocks.NewMockServer()
	server.Start()

	defer server.Stop()

	client := getClient(server.BaseURL())
	accessToken := server.GenerateNewAuthToken()

	// create 20 service accounts
	for i := 0; i < 20; i++ {
		_, err := client.CreateServiceAccount(accessToken, fmt.Sprintf("test_%d", i), fmt.Sprintf("test account %d", i))
		g.Expect(err).ToNot(gomega.HaveOccurred())
	}
	accounts, err := client.GetServiceAccounts(accessToken, 0, 100)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(accounts).To(gomega.HaveLen(20))
}

func Test_SSOClient_GetServiceAccount(t *testing.T) {
	g := gomega.NewWithT(t)

	server := mocks.NewMockServer()
	server.Start()

	defer server.Stop()

	client := getClient(server.BaseURL())
	accessToken := server.GenerateNewAuthToken()

	var serviceAccountList []serviceaccountsclient.ServiceAccountData
	// create 20 service accounts
	for i := 0; i < 20; i++ {
		serviceAccount, err := client.CreateServiceAccount(accessToken, fmt.Sprintf("test_%d", i), fmt.Sprintf("test account %d", i))
		g.Expect(err).ToNot(gomega.HaveOccurred())
		serviceAccountList = append(serviceAccountList, serviceAccount)
	}

	serviceAccount, found, err := client.GetServiceAccount(accessToken, serviceAccountList[5].GetClientId())
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(found).To(gomega.BeTrue())
	g.Expect(serviceAccount).ToNot(gomega.BeNil())
	g.Expect(serviceAccount.GetSecret()).To(gomega.Equal(serviceAccountList[5].GetSecret()))
}

func Test_SSOClient_RegenerateSecret(t *testing.T) {
	g := gomega.NewWithT(t)

	server := mocks.NewMockServer()
	server.Start()

	defer server.Stop()

	client := getClient(server.BaseURL())
	accessToken := server.GenerateNewAuthToken()

	var serviceAccountList []serviceaccountsclient.ServiceAccountData
	// create 20 service accounts
	for i := 0; i < 20; i++ {
		serviceAccount, err := client.CreateServiceAccount(accessToken, fmt.Sprintf("test_%d", i), fmt.Sprintf("test account %d", i))
		g.Expect(err).ToNot(gomega.HaveOccurred())
		serviceAccountList = append(serviceAccountList, serviceAccount)
	}

	serviceAccount, found, err := client.GetServiceAccount(accessToken, serviceAccountList[5].GetClientId())
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(found).To(gomega.BeTrue())
	g.Expect(serviceAccount).ToNot(gomega.BeNil())
	g.Expect(serviceAccount.GetSecret()).To(gomega.Equal(serviceAccountList[5].GetSecret()))

	updatedServiceAccount, err := client.RegenerateClientSecret(accessToken, serviceAccount.GetClientId())
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(updatedServiceAccount).ToNot(gomega.BeNil())
	g.Expect(updatedServiceAccount.Id).To(gomega.Equal(serviceAccount.Id))
	g.Expect(updatedServiceAccount.Secret).ToNot(gomega.Equal(serviceAccount.Secret))
}

func Test_SSOClient_CreateServiceAccount(t *testing.T) {
	g := gomega.NewWithT(t)

	server := mocks.NewMockServer()
	server.Start()

	defer server.Stop()

	client := getClient(server.BaseURL())
	accessToken := server.GenerateNewAuthToken()
	serviceAccount, err := client.CreateServiceAccount(accessToken, "test_1", "test account 1")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(*serviceAccount.Name).To(gomega.Equal("test_1"))
	g.Expect(*serviceAccount.Description).To(gomega.Equal("test account 1"))
}

func Test_SSOClient_DeleteServiceAccount(t *testing.T) {
	g := gomega.NewWithT(t)

	server := mocks.NewMockServer()
	server.Start()

	defer server.Stop()

	client := getClient(server.BaseURL())
	accessToken := server.GenerateNewAuthToken()

	// create 20 service accounts
	for i := 0; i < 20; i++ {
		_, err := client.CreateServiceAccount(accessToken, fmt.Sprintf("test_%d", i), fmt.Sprintf("test account %d", i))
		g.Expect(err).ToNot(gomega.HaveOccurred())
	}
	accounts, err := client.GetServiceAccounts(accessToken, 0, 100)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(accounts).To(gomega.HaveLen(20))
	err = client.DeleteServiceAccount(accessToken, accounts[5].GetClientId())
	g.Expect(err).ToNot(gomega.HaveOccurred())
	accounts, err = client.GetServiceAccounts(accessToken, 0, 100)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(accounts).To(gomega.HaveLen(19))
}

func Test_SSOClient_UpdateServiceAccount(t *testing.T) {
	g := gomega.NewWithT(t)

	server := mocks.NewMockServer()
	server.Start()

	defer server.Stop()

	client := getClient(server.BaseURL())
	accessToken := server.GenerateNewAuthToken()

	// create 20 service accounts
	for i := 0; i < 20; i++ {
		_, err := client.CreateServiceAccount(accessToken, fmt.Sprintf("test_%d", i), fmt.Sprintf("test account %d", i))
		g.Expect(err).ToNot(gomega.HaveOccurred())
	}
	accounts, err := client.GetServiceAccounts(accessToken, 0, 100)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(accounts).To(gomega.HaveLen(20))

	updatedName := "newName"
	updatedDescription := "newName Description"

	updatedServiceAccount, err := client.UpdateServiceAccount(accessToken, accounts[5].GetClientId(), updatedName, updatedDescription)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(*updatedServiceAccount.Name).To(gomega.Equal(updatedName))
	g.Expect(*updatedServiceAccount.Description).To(gomega.Equal(updatedDescription))
	g.Expect(*updatedServiceAccount.ClientId).To(gomega.Equal(*accounts[5].ClientId))
}

func Test_SSOClient_GetToken(t *testing.T) {
	g := gomega.NewWithT(t)

	server := mocks.NewMockServer()
	server.Start()

	defer server.Stop()

	config := keycloak.KeycloakConfig{
		SsoBaseUrl: server.BaseURL(),
		RedhatSSORealm: &keycloak.KeycloakRealmConfig{
			Realm:            "redhat-external",
			APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
			TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
		},
	}

	client := redhatsso.NewSSOClient(&config, config.RedhatSSORealm)
	accessToken := server.GenerateNewAuthToken()
	serviceAccount, err := client.CreateServiceAccount(accessToken, "test", "test desc")
	g.Expect(err).ToNot(gomega.HaveOccurred())

	config.RedhatSSORealm.ClientID = *serviceAccount.ClientId
	config.RedhatSSORealm.ClientSecret = *serviceAccount.Secret

	client = redhatsso.NewSSOClient(&config, config.RedhatSSORealm)
	token, err := client.GetToken()
	g.Expect(err).ToNot(gomega.HaveOccurred())
	fmt.Printf("TOKEN: %s\n", token)
}
