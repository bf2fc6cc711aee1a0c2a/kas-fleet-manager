package sso

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/redhatsso"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/onsi/gomega"
	pkgErr "github.com/pkg/errors"
	serviceaccountsclient "github.com/redhat-developer/app-services-sdk-go/serviceaccounts/apiv1internal/client"
)

type testGenericOpenAPIError struct {
	body  []byte
	error string
}

func (e testGenericOpenAPIError) Error() string {
	return e.error
}

func TestRedhatSSO_RegisterOSDClusterClientInSSO(t *testing.T) {
	tokenErr := pkgErr.New("token error")
	//failedToCreateClientErr := pkgErr.New("failed to create client")

	type fields struct {
		kcClient redhatsso.SSOClient
	}

	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr *errors.ServiceError
	}{
		{
			name: "throws error when failed to fetch token",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return "", tokenErr
					},
				},
			},
			want:    "",
			wantErr: errors.NewWithCause(errors.ErrorGeneral, tokenErr, "error getting access token"),
		},
		//{ // TODO: feature not yet implemented
		//	name: "fetch osd client secret from sso when client already exists",
		//	fields: fields{
		//		kcClient: &redhatsso.SSOClientMock{
		//			GetTokenFunc: func() (string, error) {
		//				return token, nil
		//			},
		//			GetConfigFunc: func() *redhatsso.RedhatSSORealm {
		//				return redhatsso.NewRedhatSSOConfig()
		//			},
		//		},
		//	},
		//	want:    secret,
		//	wantErr: nil,
		//},
		//{ // TODO: Not yet implemented
		//	name: "successfully register a new sso client for the kafka cluster",
		//	fields: fields{
		//		kcClient: &redhatsso.SSOClientMock{
		//			GetTokenFunc: func() (string, error) {
		//				return token, nil
		//			},
		//			GetConfigFunc: func() *redhatsso.RedhatSSORealm {
		//				return redhatsso.NewRedhatSSOConfig()
		//			},
		//			//IsClientExistFunc: func(clientId string, accessToken string) (string, error) {
		//			//	return "", nil
		//			//},
		//			//GetClientSecretFunc: func(internalClientId string, accessToken string) (string, error) {
		//			//	return secret, nil
		//			//},
		//			//CreateClientFunc: func(client gocloak.Client, accessToken string) (string, error) {
		//			//	return testClientID, nil
		//			//},
		//			//ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
		//			//	testID := "12221"
		//			//	return gocloak.Client{
		//			//		ClientID: &testID,
		//			//	}
		//			//},
		//		},
		//	},
		//	want:    secret,
		//	wantErr: nil,
		//},
		//{ // TODO: not yet implemented
		//	name: "failed to register sso client for the osd cluster",
		//	fields: fields{
		//		kcClient: &redhatsso.SSOClientMock{
		//			GetTokenFunc: func() (string, error) {
		//				return token, nil
		//			},
		//			GetConfigFunc: func() *redhatsso.RedhatSSORealm {
		//				return redhatsso.NewRedhatSSOConfig()
		//			},
		//			//IsClientExistFunc: func(clientId string, accessToken string) (string, error) {
		//			//	return "", nil
		//			//},
		//			//GetClientSecretFunc: func(internalClientId string, accessToken string) (string, error) {
		//			//	return secret, nil
		//			//},
		//			//CreateClientFunc: func(client gocloak.Client, accessToken string) (string, error) {
		//			//	return "", failedToCreateClientErr
		//			//},
		//			//ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
		//			//	testID := "12221"
		//			//	return gocloak.Client{
		//			//		ClientID: &testID,
		//			//	}
		//			//},
		//		},
		//	},
		//	want:    "",
		//	wantErr: errors.NewWithCause(errors.ErrorFailedToCreateSSOClient, failedToCreateClientErr, "failed to create sso client"),
		//},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			keycloakService := keycloakServiceProxy{
				getToken: tt.fields.kcClient.GetToken,
				service:  &redhatssoService{client: tt.fields.kcClient},
			}
			got, err := keycloakService.RegisterClientInSSO("osd-cluster-12212", "https://oauth-openshift-cluster.fr")
			g.Expect(err).To(gomega.Equal(tt.wantErr))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}

}

func TestRedhatSSOService_RegisterKasFleetshardOperatorServiceAccount(t *testing.T) {
	type fields struct {
		kcClient redhatsso.SSOClient
	}
	type args struct {
		clusterId string
	}

	fakeId := "kas-fleetshard-agent-test-cluster-id"
	fakeClientId := "kas-fleetshard-agent-test-cluster-id"
	fakeClientSecret := "test-client-secret"
	createdAt := int64(0)

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *api.ServiceAccount
		wantErr bool
	}{
		{
			name: "test registering serviceaccount for agent operator first time",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					CreateServiceAccountFunc: func(accessToken string, name string, description string) (serviceaccountsclient.ServiceAccountData, error) {
						return serviceaccountsclient.ServiceAccountData{
							Id:          &fakeId,
							ClientId:    &fakeClientId,
							Secret:      &fakeClientSecret,
							Name:        &name,
							Description: &description,
							CreatedBy:   nil,
							CreatedAt:   &createdAt,
						}, nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return keycloak.NewKeycloakConfig()
					},
				},
			},
			args: args{
				clusterId: "test-cluster-id",
			},
			want: &api.ServiceAccount{
				ID:           fakeClientId,
				ClientID:     "kas-fleetshard-agent-test-cluster-id",
				ClientSecret: fakeClientSecret,
				Name:         "test-cluster-id",
				Description:  "service account for agent on cluster test-cluster-id",
				CreatedAt:    time.Unix(0, shared.SafeInt64(&createdAt)*int64(time.Millisecond)),
			},
			wantErr: false,
		},
		{
			name: "test registering serviceaccount for agent operator second time",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					CreateServiceAccountFunc: func(accessToken string, name string, description string) (serviceaccountsclient.ServiceAccountData, error) {
						return serviceaccountsclient.ServiceAccountData{
							Id:          &fakeId,
							ClientId:    &fakeClientId,
							Secret:      &fakeClientSecret,
							Name:        &name,
							Description: &description,
							CreatedBy:   nil,
							CreatedAt:   &createdAt,
						}, nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return keycloak.NewKeycloakConfig()
					},
				},
			},
			args: args{
				clusterId: "test-cluster-id",
			},
			want: &api.ServiceAccount{
				ID:           fakeClientId,
				ClientID:     "kas-fleetshard-agent-test-cluster-id",
				ClientSecret: fakeClientSecret,
				Name:         "test-cluster-id",
				Description:  "service account for agent on cluster test-cluster-id",
				CreatedAt:    time.Unix(0, 0),
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			keycloakService := keycloakServiceProxy{
				getToken: tt.fields.kcClient.GetToken,
				service:  &redhatssoService{client: tt.fields.kcClient},
			}
			got, err := keycloakService.RegisterKasFleetshardOperatorServiceAccount(tt.args.clusterId)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterKasFleetshardOperatorServiceAccount() error = %v, wantErr %v", err, tt.wantErr)
			}
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_SSOClient_CreateServiceAccount(t *testing.T) {

	server := mocks.NewMockServer(mocks.WithServiceAccountLimit(50))
	server.Start()

	t.Cleanup(func() {
		server.Stop()
	})

	type args struct {
		name        string
		description string
		count       int
	}

	type expect struct {
		wantErr *testGenericOpenAPIError
		count   int
	}

	tests := []struct {
		name string
		//fields  fields
		args   args
		expect expect
		setup  func()
	}{
		{
			name: "Test create 30 service accounts",
			args: args{
				name:        "test_%d",
				description: "test account %d",
				count:       30,
			},
			expect: expect{
				wantErr: nil,
				count:   30,
			},
			setup: func() {
				server.DeleteAllServiceAccounts()
			},
		},
		{
			name: "Test create 60 service accounts",
			args: args{
				name:        "test_%d",
				description: "test account %d",
				count:       60,
			},
			expect: expect{
				wantErr: &testGenericOpenAPIError{
					body:  []byte(`{ "error": "service_account_limit_exceeded", "error_description": "Max allowed number:50 of service accounts for user has reached"}`),
					error: "401 Unauthorized",
				},
				count: 50,
			},
			setup: func() {
				server.DeleteAllServiceAccounts()
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			var err error
			count := 0
			if tt.setup != nil {
				tt.setup()
			}
			for i := 0; i < tt.args.count; i++ {
				name := fmt.Sprintf(tt.args.name, i)
				description := fmt.Sprintf(tt.args.description, i)

				client := redhatsso.NewSSOClient(&keycloak.KeycloakConfig{
					SsoBaseUrl: server.BaseURL(),
					RedhatSSORealm: &keycloak.KeycloakRealmConfig{
						Realm:            "redhat-external",
						APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
					},
					SelectSSOProvider: keycloak.REDHAT_SSO,
				},
					&keycloak.KeycloakRealmConfig{
						Realm:            "redhat-external",
						APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
					},
				)

				accessToken := server.GenerateNewAuthToken()

				_, err = client.CreateServiceAccount(accessToken, name, description)
				if err != nil {
					break
				}
				count++
			}

			g.Expect(err == nil).To(gomega.Equal(tt.expect.wantErr == nil))
			g.Expect(count).To(gomega.Equal(tt.expect.count))
			if err != nil {
				g.Expect(err).To(gomega.BeAssignableToTypeOf(serviceaccountsclient.GenericOpenAPIError{}))
				openApiError := err.(serviceaccountsclient.GenericOpenAPIError)
				g.Expect(openApiError.Error()).To(gomega.Equal(tt.expect.wantErr.Error()))
				g.Expect(openApiError.Body()).To(gomega.Equal(tt.expect.wantErr.body))
			}
		})
	}

}

func TestRedhatSSOService_DeRegisterKasFleetshardOperatorServiceAccount(t *testing.T) {
	type fields struct {
		kcClient redhatsso.SSOClient
	}
	type args struct {
		clusterId string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "should receive an error when retrieving the token fails",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return "", fmt.Errorf("some errors")
					},
					DeleteServiceAccountFunc: func(accessToken string, clientId string) error {
						return fmt.Errorf("some error")
					},
				},
			},
			args: args{
				clusterId: "test-cluster-id",
			},
			wantErr: true,
		},
		{
			name: "should receive an error when service account deletion fails",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					GetServiceAccountFunc: func(accessToken string, clientId string) (*serviceaccountsclient.ServiceAccountData, bool, error) {
						return nil, true, nil
					},
					DeleteServiceAccountFunc: func(accessToken string, clientId string) error {
						return fmt.Errorf("some error")
					},
				},
			},
			args: args{
				clusterId: "test-cluster-id",
			},
			wantErr: true,
		},
		{
			name: "should delete the service account",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					GetServiceAccountFunc: func(accessToken string, clientId string) (*serviceaccountsclient.ServiceAccountData, bool, error) {
						return nil, true, nil
					},
					DeleteServiceAccountFunc: func(accessToken string, clientId string) error {
						return nil
					},
				},
			},
			args: args{
				clusterId: "test-cluster-id",
			},
			wantErr: false,
		},
		{
			name: "should not call delete if client doesn't exist",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					GetServiceAccountFunc: func(accessToken string, clientId string) (*serviceaccountsclient.ServiceAccountData, bool, error) {
						return nil, false, nil
					},
					DeleteServiceAccountFunc: func(accessToken string, clientId string) error {
						return fmt.Errorf("this should not be called")
					},
				},
			},
			args: args{
				clusterId: "test-cluster-id",
			},
			wantErr: false,
		},
	}
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			keycloakService := keycloakServiceProxy{
				getToken: tt.fields.kcClient.GetToken,
				service:  &redhatssoService{client: tt.fields.kcClient},
			}
			err := keycloakService.DeRegisterKasFleetshardOperatorServiceAccount(tt.args.clusterId)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestRedhatSSOService_RegisterConnectorFleetshardOperatorServiceAccount(t *testing.T) {
	type fields struct {
		kcClient redhatsso.SSOClient
	}
	type args struct {
		clusterId string
		roleName  string
	}
	//fakeRoleId := "1234"
	fakeClientId := "test-client-id"
	fakeClientSecret := "test-client-secret"
	//fakeUserId := "test-user-id"
	tests := []struct {
		name       string
		disabled   bool
		skipReason string
		fields     fields
		args       args
		want       *api.ServiceAccount
		wantErr    bool
	}{
		{
			name: "test registering serviceaccount for agent operator first time",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					CreateServiceAccountFunc: func(accessToken string, name string, description string) (serviceaccountsclient.ServiceAccountData, error) {
						createdAt := int64(0)
						return serviceaccountsclient.ServiceAccountData{
							Id:          &fakeClientId,
							ClientId:    &fakeClientId,
							Secret:      &fakeClientSecret,
							Name:        &name,
							Description: &description,
							CreatedBy:   nil,
							CreatedAt:   &createdAt,
						}, nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return keycloak.NewKeycloakConfig()
					},
				},
			},
			args: args{
				clusterId: "test-cluster-id",
				roleName:  "test-role-name",
			},
			want: &api.ServiceAccount{
				ID:           fakeClientId,
				ClientID:     fakeClientId,
				ClientSecret: fakeClientSecret,
				Name:         "test-cluster-id",
				Description:  "service account for agent on cluster test-cluster-id",
				CreatedAt:    time.Unix(0, 0),
			},
			wantErr: false,
		},
		{
			name:       "test registering serviceaccount for agent operator second time",
			disabled:   true,
			skipReason: "RegisterConnectorFleetshardOperatorServiceAccount not yet implemented",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return keycloak.NewKeycloakConfig()
					},
				},
			},
			args: args{
				clusterId: "test-cluster-id",
				roleName:  "test-role-name",
			},
			want: &api.ServiceAccount{
				ID:           fakeClientId,
				ClientID:     "connector-fleetshard-agent-test-cluster-id",
				ClientSecret: fakeClientSecret,
				Name:         "connector-fleetshard-agent-test-cluster-id",
				Description:  "service account for agent on cluster test-cluster-id",
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			if tt.disabled {
				t.Skip(tt.skipReason)
			}
			keycloakService := keycloakServiceProxy{
				getToken: tt.fields.kcClient.GetToken,
				service:  &redhatssoService{client: tt.fields.kcClient},
			}
			got, err := keycloakService.RegisterConnectorFleetshardOperatorServiceAccount(tt.args.clusterId)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterConnectorFleetshardOperatorServiceAccount() error = %v, wantErr %v", err, tt.wantErr)
			}
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func TestRedhatSSOService__DeRegisterConnectorFleetshardOperatorServiceAccount(t *testing.T) {
	type fields struct {
		kcClient redhatsso.SSOClient
	}
	type args struct {
		clusterId string
	}
	tests := []struct {
		name       string
		disabled   bool
		skipReason string
		fields     fields
		args       args
		wantErr    bool
	}{
		{
			name: "should receive an error when retrieving the token fails",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return "", fmt.Errorf("some errors")
					},
				},
			},
			args: args{
				clusterId: "test-cluster-id",
			},
			wantErr: true,
		},
		{
			name: "should receive an error when service account deletion fails",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					GetServiceAccountFunc: func(accessToken string, clientId string) (*serviceaccountsclient.ServiceAccountData, bool, error) {
						return nil, true, nil
					},
					DeleteServiceAccountFunc: func(accessToken string, clientId string) error {
						return fmt.Errorf("some error")
					},
				},
			},
			args: args{
				clusterId: "test-cluster-id",
			},
			wantErr: true,
		},
		{
			name: "should delete the service account",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					GetServiceAccountFunc: func(accessToken string, clientId string) (*serviceaccountsclient.ServiceAccountData, bool, error) {
						return nil, true, nil
					},
					DeleteServiceAccountFunc: func(accessToken string, clientId string) error {
						return nil
					},
				},
			},
			args: args{
				clusterId: "test-cluster-id",
			},
			wantErr: false,
		},
		{name: "should not call delete if client doesn't exist",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					GetServiceAccountFunc: func(accessToken string, clientId string) (*serviceaccountsclient.ServiceAccountData, bool, error) {
						return nil, false, nil
					},
					DeleteServiceAccountFunc: func(accessToken string, clientId string) error {
						return fmt.Errorf("this should not be called")
					},
				},
			},
			args: args{
				clusterId: "test-cluster-id",
			},
			wantErr: false,
		},
	}
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			if tt.disabled {
				t.Skip(tt.skipReason)
			}
			g := gomega.NewWithT(t)
			keycloakService := keycloakServiceProxy{
				getToken: tt.fields.kcClient.GetToken,
				service:  &redhatssoService{client: tt.fields.kcClient},
			}
			err := keycloakService.DeRegisterConnectorFleetshardOperatorServiceAccount(tt.args.clusterId)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestRedhatSSOService_DeleteServiceAccountInternal(t *testing.T) {

	type fields struct {
		kcClient redhatsso.SSOClient
	}
	tests := []struct {
		name       string
		disable    bool
		skipReason string
		fields     fields
		wantErr    bool
	}{
		{
			name: "returns error when failed to fetch token",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return "", pkgErr.New("token error")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "do not return an error when service account deleted successfully",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return "", nil
					},
					DeleteServiceAccountFunc: func(accessToken string, clientId string) error {
						return nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "do not return an error when service account does not exists",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return "", nil
					},
					DeleteServiceAccountFunc: func(accessToken string, clientId string) error {
						return nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "returns an error when failed to delete service account",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return "", nil
					},
					DeleteServiceAccountFunc: func(accessToken string, clientId string) error {
						return fmt.Errorf("internal server error")
					},
				},
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			keycloakService := keycloakServiceProxy{
				getToken: tt.fields.kcClient.GetToken,
				service:  &redhatssoService{client: tt.fields.kcClient},
			}
			err := keycloakService.DeleteServiceAccountInternal("account-id")
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}

}

func TestRedhatSSOService_CreateServiceAccountInternal(t *testing.T) {
	tokenErr := pkgErr.New("token error")
	request := CompleteServiceAccountRequest{
		Owner:          "some-owner",
		OwnerAccountId: "owner-account-id",
		ClientId:       "some-client-id",
		Name:           "some-name",
		Description:    "some-description",
		OrgId:          "some-organization-id",
	}
	id := "dsd"
	clientSecret := "secret"
	type fields struct {
		kcClient redhatsso.SSOClient
	}
	tests := []struct {
		name                  string
		fields                fields
		wantErr               bool
		serviceAccountCreated bool
	}{
		{
			name: "returns error when failed to fetch token",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return "", tokenErr
					},
					CreateServiceAccountFunc: func(accessToken, name, description string) (serviceaccountsclient.ServiceAccountData, error) {
						return serviceaccountsclient.ServiceAccountData{}, nil
					},
				},
			},
			wantErr:               true,
			serviceAccountCreated: false,
		},
		{
			name: "returns error when failed to create service account",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return "", nil
					},
					CreateServiceAccountFunc: func(accessToken, name, description string) (serviceaccountsclient.ServiceAccountData, error) {
						return serviceaccountsclient.ServiceAccountData{}, errors.New(errors.ErrorFailedToCreateServiceAccount, "failed to create service account")
					},
				},
			},
			wantErr:               true,
			serviceAccountCreated: false,
		},
		{
			name: "succeed to create service account error when failed to create client",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return "", nil
					},
					CreateServiceAccountFunc: func(accessToken, name, description string) (serviceaccountsclient.ServiceAccountData, error) {
						return serviceaccountsclient.ServiceAccountData{
							Id:       &id,
							ClientId: &request.ClientId,
							Secret:   &clientSecret,
						}, nil
					},
				},
			},
			wantErr:               false,
			serviceAccountCreated: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			keycloakService := keycloakServiceProxy{
				getToken: tt.fields.kcClient.GetToken,
				service:  &redhatssoService{client: tt.fields.kcClient},
			}
			serviceAccount, err := keycloakService.CreateServiceAccountInternal(request)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(serviceAccount != nil).To(gomega.Equal(tt.serviceAccountCreated))
			if tt.serviceAccountCreated {
				g.Expect(serviceAccount.ClientSecret).To(gomega.Equal("secret"))
				g.Expect(serviceAccount.ClientID).To(gomega.Equal(request.ClientId))
				g.Expect(serviceAccount.ID).To(gomega.Equal("dsd"))
			}
		})
	}

}

func Test_redhatssoService_DeRegisterClientInSSO(t *testing.T) {
	type fields struct {
		client redhatsso.SSOClient
	}
	type args struct {
		accessToken string
		clientId    string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *errors.ServiceError
	}{
		{
			name: "should return nil if client is deregistered successfully",
			fields: fields{
				client: &redhatsso.SSOClientMock{
					DeleteServiceAccountFunc: func(accessToken, clientId string) error {
						return nil
					},
				},
			},
			args: args{
				accessToken: token,
				clientId:    testClientID,
			},
			want: nil,
		},
		{
			name: "should return an error if it fails to deregistered client",
			fields: fields{
				client: &redhatsso.SSOClientMock{
					DeleteServiceAccountFunc: func(accessToken, clientId string) error {
						return errors.New(errors.ErrorFailedToDeleteSSOClient, "failed to delete the sso client")
					},
				},
			},
			args: args{
				accessToken: token,
				clientId:    testClientID,
			},
			want: errors.NewWithCause(errors.ErrorFailedToDeleteSSOClient, errors.FailedToDeleteSSOClient("failed to delete the sso client"), "failed to delete the sso client"),
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			r := &redhatssoService{
				client: tt.fields.client,
			}
			g.Expect(r.DeRegisterClientInSSO(tt.args.accessToken, tt.args.clientId)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_redhatssoService_GetConfig(t *testing.T) {
	type fields struct {
		client redhatsso.SSOClient
	}
	tests := []struct {
		name   string
		fields fields
		want   *keycloak.KeycloakConfig
	}{
		{
			name: "should return the redhatsso config",
			fields: fields{
				client: &redhatsso.SSOClientMock{
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{}
					},
				},
			},
			want: &keycloak.KeycloakConfig{},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			r := &redhatssoService{
				client: tt.fields.client,
			}
			g.Expect(r.GetConfig()).To(gomega.Equal(tt.want))
		})
	}
}

func Test_redhatssoService_GetRealmConfig(t *testing.T) {
	type fields struct {
		client redhatsso.SSOClient
	}
	tests := []struct {
		name   string
		fields fields
		want   *keycloak.KeycloakRealmConfig
	}{
		{
			name: "should return the realm config",
			fields: fields{
				client: &redhatsso.SSOClientMock{
					GetRealmConfigFunc: func() *keycloak.KeycloakRealmConfig {
						return &keycloak.KeycloakRealmConfig{}
					},
				},
			},
			want: &keycloak.KeycloakRealmConfig{},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			r := &redhatssoService{
				client: tt.fields.client,
			}
			g.Expect(r.GetRealmConfig()).To(gomega.Equal(tt.want))
		})
	}
}

func Test_redhatssoService_IsKafkaClientExist(t *testing.T) {
	type fields struct {
		client redhatsso.SSOClient
	}
	type args struct {
		accessToken string
		clientId    string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *errors.ServiceError
	}{
		{
			name: "should return nil if kafka client exists",
			fields: fields{
				client: &redhatsso.SSOClientMock{
					GetServiceAccountFunc: func(accessToken, clientId string) (*serviceaccountsclient.ServiceAccountData, bool, error) {
						return &serviceaccountsclient.ServiceAccountData{}, true, nil
					},
				},
			},
			args: args{
				accessToken: token,
				clientId:    testClientID,
			},
			want: nil,
		},
		{
			name: "should return error if it fails to get sso client",
			fields: fields{
				client: &redhatsso.SSOClientMock{
					GetServiceAccountFunc: func(accessToken, clientId string) (*serviceaccountsclient.ServiceAccountData, bool, error) {
						return &serviceaccountsclient.ServiceAccountData{}, true, errors.New(errors.ErrorFailedToGetSSOClient, "failed to get sso client with id: %s", testClientID)
					},
				},
			},
			args: args{
				accessToken: token,
				clientId:    testClientID,
			},
			want: errors.NewWithCause(errors.ErrorFailedToGetSSOClient, errors.FailedToGetSSOClient("failed to get sso client with id: %s", testClientID), "failed to get sso client with id: %s", testClientID),
		},
		{
			name: "should return an error if the sso client cannot be found",
			fields: fields{
				client: &redhatsso.SSOClientMock{
					GetServiceAccountFunc: func(accessToken, clientId string) (*serviceaccountsclient.ServiceAccountData, bool, error) {
						return &serviceaccountsclient.ServiceAccountData{}, false, nil
					},
				},
			},
			args: args{
				accessToken: token,
				clientId:    testClientID,
			},
			want: errors.New(errors.ErrorNotFound, "sso client with id: %s not found", testClientID),
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			r := &redhatssoService{
				client: tt.fields.client,
			}
			g.Expect(r.IsKafkaClientExist(tt.args.accessToken, tt.args.clientId)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_redhatssoService_CreateServiceAccount(t *testing.T) {
	type fields struct {
		client redhatsso.SSOClient
	}
	type args struct {
		accessToken           string
		serviceAccountRequest *api.ServiceAccountRequest
		ctx                   context.Context
	}
	clientId := testClientID
	requestName := "ServiceAccountRequest_Name"
	requestDescription := "ServiceAccountRequest_Description"
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *api.ServiceAccount
		wantErr *errors.ServiceError
	}{
		{
			name: "should return the service account if its created successfully",
			fields: fields{
				client: &redhatsso.SSOClientMock{
					CreateServiceAccountFunc: func(accessToken, name, description string) (serviceaccountsclient.ServiceAccountData, error) {
						return serviceaccountsclient.ServiceAccountData{
							ClientId:    &clientId,
							Name:        &requestName,
							Description: &requestDescription,
						}, nil
					},
				},
			},
			args: args{
				accessToken: token,
				serviceAccountRequest: &api.ServiceAccountRequest{
					Name:        requestName,
					Description: requestDescription,
				},
				ctx: context.Background(),
			},
			want: &api.ServiceAccount{
				ClientID:    testClientID,
				Name:        requestName,
				Description: requestDescription,
			},
			wantErr: nil,
		},
		{
			name: "should return an error if it fails to create a service account ",
			fields: fields{
				client: &redhatsso.SSOClientMock{
					CreateServiceAccountFunc: func(accessToken, name, description string) (serviceaccountsclient.ServiceAccountData, error) {
						return serviceaccountsclient.ServiceAccountData{}, errors.New(errors.ErrorFailedToCreateServiceAccount, "failed to create service account")
					},
				},
			},
			args: args{
				accessToken: token,
				serviceAccountRequest: &api.ServiceAccountRequest{
					Name:        requestName,
					Description: requestDescription,
				},
				ctx: context.Background(),
			},
			want:    nil,
			wantErr: errors.NewWithCause(errors.ErrorFailedToCreateServiceAccount, errors.FailedToCreateServiceAccount("failed to create service account"), "failed to create service account"),
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			r := &redhatssoService{
				client: tt.fields.client,
			}
			got, err := r.CreateServiceAccount(tt.args.accessToken, tt.args.serviceAccountRequest, tt.args.ctx)
			g.Expect(err).To(gomega.Equal(tt.wantErr))

			if tt.want != nil {
				g.Expect(got.ClientID).To(gomega.Equal(tt.want.ClientID))
				g.Expect(got.Name).To(gomega.Equal(tt.want.Name))
				g.Expect(got.Description).To(gomega.Equal(tt.want.Description))
			}
		})
	}
}

func Test_redhatssoService_DeleteServiceAccount(t *testing.T) {
	type fields struct {
		client redhatsso.SSOClient
	}
	type args struct {
		accessToken string
		ctx         context.Context
		clientId    string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *errors.ServiceError
	}{
		{
			name: "should return nil if the client is successfully deleted",
			fields: fields{
				client: &redhatsso.SSOClientMock{
					DeleteServiceAccountFunc: func(accessToken, clientId string) error {
						return nil
					},
				},
			},
			args: args{
				accessToken: token,
				ctx:         context.Background(),
				clientId:    testClientID,
			},
			want: nil,
		},
		{
			name: "should return an error if it fails to delete the client",
			fields: fields{
				client: &redhatsso.SSOClientMock{
					DeleteServiceAccountFunc: func(accessToken, clientId string) error {
						return errors.New(errors.ErrorFailedToDeleteServiceAccount, "failed to delete service account")
					},
				},
			},
			args: args{
				accessToken: token,
				ctx:         context.Background(),
				clientId:    testClientID,
			},
			want: errors.NewWithCause(errors.ErrorFailedToDeleteServiceAccount, errors.FailedToDeleteServiceAccount("failed to delete service account"), "failed to delete service account"),
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			r := &redhatssoService{
				client: tt.fields.client,
			}
			g.Expect(r.DeleteServiceAccount(tt.args.accessToken, tt.args.ctx, tt.args.clientId)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_redhatssoService_ResetServiceAccountCredentials(t *testing.T) {
	type fields struct {
		client redhatsso.SSOClient
	}
	type args struct {
		accessToken string
		ctx         context.Context
		clientId    string
	}
	clientId := testClientID
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *api.ServiceAccount
		wantErr *errors.ServiceError
	}{
		{
			name: "should return the service account if it successfully resets the service account credentials",
			fields: fields{
				client: &redhatsso.SSOClientMock{
					RegenerateClientSecretFunc: func(accessToken, id string) (serviceaccountsclient.ServiceAccountData, error) {
						return serviceaccountsclient.ServiceAccountData{
							ClientId: &clientId,
						}, nil
					},
				},
			},
			args: args{
				accessToken: token,
				ctx:         context.Background(),
				clientId:    testClientID,
			},
			want: &api.ServiceAccount{
				ClientID: clientId,
			},
			wantErr: nil,
		},
		{
			name: "should return an error if it fails to reset the service account credentials",
			fields: fields{
				client: &redhatsso.SSOClientMock{
					RegenerateClientSecretFunc: func(accessToken, id string) (serviceaccountsclient.ServiceAccountData, error) {
						return serviceaccountsclient.ServiceAccountData{}, errors.New(errors.ErrorGeneral, "failed to reset service account credentials")
					},
				},
			},
			args: args{
				accessToken: token,
				ctx:         context.Background(),
				clientId:    testClientID,
			},
			want:    nil,
			wantErr: errors.NewWithCause(errors.ErrorGeneral, errors.GeneralError("failed to reset service account credentials"), "failed to reset service account credentials"),
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			r := &redhatssoService{
				client: tt.fields.client,
			}
			got, err := r.ResetServiceAccountCredentials(tt.args.accessToken, tt.args.ctx, tt.args.clientId)
			g.Expect(err).To(gomega.Equal(tt.wantErr))
			if tt.want != nil {
				g.Expect(got.ClientID).To(gomega.Equal(tt.want.ClientID))
			}
		})
	}
}

func Test_redhatssoService_ListServiceAcc(t *testing.T) {
	type fields struct {
		client redhatsso.SSOClient
	}
	type args struct {
		accessToken string
		ctx         context.Context
		first       int
		max         int
	}
	clientId := testClientID
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []api.ServiceAccount
		wantErr *errors.ServiceError
	}{
		{
			name: "should return a list of service accounts",
			fields: fields{
				client: &redhatsso.SSOClientMock{
					GetServiceAccountsFunc: func(accessToken string, first, max int) ([]serviceaccountsclient.ServiceAccountData, error) {
						return []serviceaccountsclient.ServiceAccountData{
							{
								ClientId: &clientId,
							},
							{
								ClientId: &clientId,
							},
						}, nil
					},
				},
			},
			args: args{
				accessToken: token,
				ctx:         context.Background(),
				first:       1,
				max:         10,
			},
			want: []api.ServiceAccount{
				{
					ClientID:  clientId,
					CreatedAt: time.Unix(0, 0),
				},
				{
					ClientID:  clientId,
					CreatedAt: time.Unix(0, 0),
				},
			},
			wantErr: nil,
		},
		{
			name: "should return an error if it fails to collect the service accounts",
			fields: fields{
				client: &redhatsso.SSOClientMock{
					GetServiceAccountsFunc: func(accessToken string, first, max int) ([]serviceaccountsclient.ServiceAccountData, error) {
						return []serviceaccountsclient.ServiceAccountData{}, errors.New(errors.ErrorGeneral, "failed to collect service accounts")
					},
				},
			},
			args: args{
				accessToken: token,
				ctx:         context.Background(),
				first:       1,
				max:         10,
			},
			want:    nil,
			wantErr: errors.NewWithCause(errors.ErrorGeneral, errors.GeneralError("failed to collect service accounts"), "failed to collect service accounts"),
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			r := &redhatssoService{
				client: tt.fields.client,
			}
			got, err := r.ListServiceAcc(tt.args.accessToken, tt.args.ctx, tt.args.first, tt.args.max)
			g.Expect(err).To(gomega.Equal(tt.wantErr))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_redhatssoService_GetServiceAccountById(t *testing.T) {
	type fields struct {
		client redhatsso.SSOClient
	}
	type args struct {
		accessToken string
		ctx         context.Context
		id          string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *api.ServiceAccount
		wantErr *errors.ServiceError
	}{
		{
			name: "should return the service account",
			fields: fields{
				client: &redhatsso.SSOClientMock{
					GetServiceAccountFunc: func(accessToken, clientId string) (*serviceaccountsclient.ServiceAccountData, bool, error) {
						return &serviceaccountsclient.ServiceAccountData{}, true, nil
					},
				},
			},
			args: args{
				accessToken: token,
				ctx:         context.Background(),
				id:          testClientID,
			},
			want: &api.ServiceAccount{
				CreatedAt: time.Unix(0, 0),
			},
			wantErr: nil,
		},
		{
			name: "should return an error if cant retrieve the service account",
			fields: fields{
				client: &redhatsso.SSOClientMock{
					GetServiceAccountFunc: func(accessToken, clientId string) (*serviceaccountsclient.ServiceAccountData, bool, error) {
						return &serviceaccountsclient.ServiceAccountData{}, true, errors.New(errors.ErrorGeneral, "error retrieving service account with clientId %s", testClientID)
					},
				},
			},
			args: args{
				accessToken: token,
				ctx:         context.Background(),
				id:          testClientID,
			},
			want:    nil,
			wantErr: errors.NewWithCause(errors.ErrorGeneral, errors.GeneralError("error retrieving service account with clientId %s", testClientID), "error retrieving service account with clientId %s", testClientID),
		},
		{
			name: "should return an error if the service account cannot be found",
			fields: fields{
				client: &redhatsso.SSOClientMock{
					GetServiceAccountFunc: func(accessToken, clientId string) (*serviceaccountsclient.ServiceAccountData, bool, error) {
						return &serviceaccountsclient.ServiceAccountData{}, false, nil
					},
				},
			},
			args: args{
				accessToken: token,
				ctx:         context.Background(),
				id:          testClientID,
			},
			want:    nil,
			wantErr: errors.ServiceAccountNotFound("service account not found clientId %s", testClientID),
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			r := &redhatssoService{
				client: tt.fields.client,
			}
			got, err := r.GetServiceAccountById(tt.args.accessToken, tt.args.ctx, tt.args.id)
			g.Expect(err).To(gomega.Equal(tt.wantErr))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_redhatssoService_GetKafkaClientSecret(t *testing.T) {
	type fields struct {
		client redhatsso.SSOClient
	}
	type args struct {
		accessToken string
		clientId    string
	}
	clientSecret := secret
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr *errors.ServiceError
	}{
		{
			name: "should return the the client secret",
			fields: fields{
				client: &redhatsso.SSOClientMock{
					GetServiceAccountFunc: func(accessToken, clientId string) (*serviceaccountsclient.ServiceAccountData, bool, error) {
						return &serviceaccountsclient.ServiceAccountData{
							Secret: &clientSecret,
						}, true, nil
					},
					RegenerateClientSecretFunc: func(accessToken string, id string) (serviceaccountsclient.ServiceAccountData, error) {
						return serviceaccountsclient.ServiceAccountData{
							Secret: &clientSecret,
						}, nil
					},
				},
			},
			args: args{
				accessToken: token,
				clientId:    testClientID,
			},
			want:    clientSecret,
			wantErr: nil,
		},
		{
			name: "should return an error if it failed to find the service account",
			fields: fields{
				client: &redhatsso.SSOClientMock{
					GetServiceAccountFunc: func(accessToken, clientId string) (*serviceaccountsclient.ServiceAccountData, bool, error) {
						return nil, false, errors.New(errors.ErrorFailedToGetSSOClientSecret, "failed to get sso client secret")
					},
				},
			},
			args: args{
				accessToken: token,
				clientId:    testClientID,
			},
			want:    "",
			wantErr: errors.NewWithCause(errors.ErrorFailedToGetSSOClient, errors.FailedToGetSSOClientSecret("failed to get sso client secret"), "failed to get sso client with id: %s", testClientID),
		},
		{
			name: "should return an error if it failed to get the client secret",
			fields: fields{
				client: &redhatsso.SSOClientMock{
					GetServiceAccountFunc: func(accessToken, clientId string) (*serviceaccountsclient.ServiceAccountData, bool, error) {
						return &serviceaccountsclient.ServiceAccountData{}, false, nil
					},
					RegenerateClientSecretFunc: func(accessToken, id string) (serviceaccountsclient.ServiceAccountData, error) {
						return serviceaccountsclient.ServiceAccountData{}, errors.New(errors.ErrorFailedToGetSSOClientSecret, "failed to get sso client secret")
					},
				},
			},
			args: args{
				accessToken: token,
				clientId:    testClientID,
			},
			want:    "",
			wantErr: errors.FailedToGetSSOClientSecret("failed to get sso client secret"),
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			r := &redhatssoService{
				client: tt.fields.client,
			}
			got, err := r.GetKafkaClientSecret(tt.args.accessToken, tt.args.clientId)
			g.Expect(err).To(gomega.Equal(tt.wantErr))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}
