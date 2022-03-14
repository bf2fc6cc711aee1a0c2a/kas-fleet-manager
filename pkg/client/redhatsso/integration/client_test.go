package integration

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/redhatsso"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	. "github.com/onsi/gomega"
	serviceaccountsclient "github.com/redhat-developer/app-services-sdk-go/serviceaccounts/apiv1internal/client"
	"testing"
)

func getClient(baseURL string) redhatsso.SSOClient {
	config := redhatsso.RedhatSSOConfig{
		BaseURL: baseURL,
		KafkaRealm: redhatsso.RealmConfig{
			Realm:            "redhat-external",
			APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", baseURL),
			TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", baseURL),
		},
	}

	return redhatsso.NewSSOClient(&config)
}

func Test_SSOClient_GetServiceAccounts(t *testing.T) {
	RegisterTestingT(t)

	server := mocks.NewMockServer()
	server.Start()

	defer server.Stop()

	client := getClient(server.BaseURL())
	accessToken := server.GenerateNewAuthToken()

	// create 20 service accounts
	for i := 0; i < 20; i++ {
		_, err := client.CreateServiceAccount(accessToken, fmt.Sprintf("test_%d", i), fmt.Sprintf("test account %d", i))
		Expect(err).ToNot(HaveOccurred())
	}
	accounts, err := client.GetServiceAccounts(accessToken, 0, 100)
	Expect(err).ToNot(HaveOccurred())
	Expect(accounts).To(HaveLen(20))
}

func Test_SSOClient_GetServiceAccount(t *testing.T) {
	RegisterTestingT(t)

	server := mocks.NewMockServer()
	server.Start()

	defer server.Stop()

	client := getClient(server.BaseURL())
	accessToken := server.GenerateNewAuthToken()

	var serviceAccountList []serviceaccountsclient.ServiceAccountData
	// create 20 service accounts
	for i := 0; i < 20; i++ {
		serviceAccount, err := client.CreateServiceAccount(accessToken, fmt.Sprintf("test_%d", i), fmt.Sprintf("test account %d", i))
		Expect(err).ToNot(HaveOccurred())
		serviceAccountList = append(serviceAccountList, serviceAccount)
	}

	serviceAccount, found, err := client.GetServiceAccount(accessToken, serviceAccountList[5].GetClientId())
	Expect(err).ToNot(HaveOccurred())
	Expect(found).To(BeTrue())
	Expect(serviceAccount).ToNot(BeNil())
	Expect(serviceAccount.GetSecret()).To(Equal(serviceAccountList[5].GetSecret()))
}

func Test_SSOClient_RegenerateSecret(t *testing.T) {
	RegisterTestingT(t)

	server := mocks.NewMockServer()
	server.Start()

	defer server.Stop()

	client := getClient(server.BaseURL())
	accessToken := server.GenerateNewAuthToken()

	var serviceAccountList []serviceaccountsclient.ServiceAccountData
	// create 20 service accounts
	for i := 0; i < 20; i++ {
		serviceAccount, err := client.CreateServiceAccount(accessToken, fmt.Sprintf("test_%d", i), fmt.Sprintf("test account %d", i))
		Expect(err).ToNot(HaveOccurred())
		serviceAccountList = append(serviceAccountList, serviceAccount)
	}

	serviceAccount, found, err := client.GetServiceAccount(accessToken, serviceAccountList[5].GetClientId())
	Expect(err).ToNot(HaveOccurred())
	Expect(found).To(BeTrue())
	Expect(serviceAccount).ToNot(BeNil())
	Expect(serviceAccount.GetSecret()).To(Equal(serviceAccountList[5].GetSecret()))

	updatedServiceAccount, err := client.RegenerateClientSecret(accessToken, serviceAccount.GetClientId())
	Expect(err).ToNot(HaveOccurred())
	Expect(updatedServiceAccount).ToNot(BeNil())
	Expect(updatedServiceAccount.Id).To(Equal(serviceAccount.Id))
	Expect(updatedServiceAccount.Secret).ToNot(Equal(serviceAccount.Secret))
}

func Test_SSOClient_CreateServiceAccount(t *testing.T) {
	RegisterTestingT(t)

	server := mocks.NewMockServer()
	server.Start()

	defer server.Stop()

	client := getClient(server.BaseURL())
	accessToken := server.GenerateNewAuthToken()
	serviceAccount, err := client.CreateServiceAccount(accessToken, "test_1", "test account 1")
	Expect(err).ToNot(HaveOccurred())
	Expect(*serviceAccount.Name).To(Equal("test_1"))
	Expect(*serviceAccount.Description).To(Equal("test account 1"))
}

func Test_SSOClient_DeleteServiceAccount(t *testing.T) {
	RegisterTestingT(t)

	server := mocks.NewMockServer()
	server.Start()

	defer server.Stop()

	client := getClient(server.BaseURL())
	accessToken := server.GenerateNewAuthToken()

	// create 20 service accounts
	for i := 0; i < 20; i++ {
		_, err := client.CreateServiceAccount(accessToken, fmt.Sprintf("test_%d", i), fmt.Sprintf("test account %d", i))
		Expect(err).ToNot(HaveOccurred())
	}
	accounts, err := client.GetServiceAccounts(accessToken, 0, 100)
	Expect(err).ToNot(HaveOccurred())
	Expect(accounts).To(HaveLen(20))
	err = client.DeleteServiceAccount(accessToken, accounts[5].GetClientId())
	Expect(err).ToNot(HaveOccurred())
	accounts, err = client.GetServiceAccounts(accessToken, 0, 100)
	Expect(err).ToNot(HaveOccurred())
	Expect(accounts).To(HaveLen(19))
}

func Test_SSOClient_UpdateServiceAccount(t *testing.T) {
	RegisterTestingT(t)

	server := mocks.NewMockServer()
	server.Start()

	defer server.Stop()

	client := getClient(server.BaseURL())
	accessToken := server.GenerateNewAuthToken()

	// create 20 service accounts
	for i := 0; i < 20; i++ {
		_, err := client.CreateServiceAccount(accessToken, fmt.Sprintf("test_%d", i), fmt.Sprintf("test account %d", i))
		Expect(err).ToNot(HaveOccurred())
	}
	accounts, err := client.GetServiceAccounts(accessToken, 0, 100)
	Expect(err).ToNot(HaveOccurred())
	Expect(accounts).To(HaveLen(20))

	updatedName := "newName"
	updatedDescription := "newName Description"

	updatedServiceAccount, err := client.UpdateServiceAccount(accessToken, accounts[5].GetClientId(), updatedName, updatedDescription)
	Expect(err).ToNot(HaveOccurred())
	Expect(*updatedServiceAccount.Name).To(Equal(updatedName))
	Expect(*updatedServiceAccount.Description).To(Equal(updatedDescription))
	Expect(*updatedServiceAccount.ClientId).To(Equal(*accounts[5].ClientId))
}

func Test_SSOClient_GetToken(t *testing.T) {
	RegisterTestingT(t)

	server := mocks.NewMockServer()
	server.Start()

	defer server.Stop()

	config := redhatsso.RedhatSSOConfig{
		BaseURL: server.BaseURL(),
		KafkaRealm: redhatsso.RealmConfig{
			Realm:            "redhat-external",
			APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
			TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
		},
	}

	client := redhatsso.NewSSOClient(&config)
	accessToken := server.GenerateNewAuthToken()
	serviceAccount, err := client.CreateServiceAccount(accessToken, "test", "test desc")
	Expect(err).ToNot(HaveOccurred())

	config.KafkaRealm.ClientID = *serviceAccount.ClientId
	config.KafkaRealm.ClientSecret = *serviceAccount.Secret

	client = redhatsso.NewSSOClient(&config)
	token, err := client.GetToken()
	Expect(err).ToNot(HaveOccurred())
	fmt.Printf("TOKEN: %s\n", token)
}
