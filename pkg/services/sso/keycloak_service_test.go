package sso

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/Nerzal/gocloak/v11"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/google/uuid"
	"github.com/onsi/gomega"
	pkgErr "github.com/pkg/errors"
)

const (
	token        = "token"
	testClientID = "12221"
	secret       = "secret"
)

func TestKeycloakService_RegisterOSDClusterClientInSSO(t *testing.T) {
	tokenErr := pkgErr.New("token error")
	failedToCreateClientErr := pkgErr.New("failed to create client")

	type fields struct {
		kcClient keycloak.KcClient
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
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return "", tokenErr
					},
				},
			},
			want:    "",
			wantErr: errors.NewWithCause(errors.ErrorGeneral, tokenErr, "error getting access token"),
		},
		{
			name: "fetch osd client secret from sso when client already exists",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return keycloak.NewKeycloakConfig()
					},
					IsClientExistFunc: func(clientId string, accessToken string) (string, error) {
						return testClientID, nil
					},
					GetClientSecretFunc: func(internalClientId string, accessToken string) (string, error) {
						return secret, nil
					},
				},
			},
			want:    secret,
			wantErr: nil,
		},
		{
			name: "successfully register a new sso client for the kafka cluster",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return keycloak.NewKeycloakConfig()
					},
					IsClientExistFunc: func(clientId string, accessToken string) (string, error) {
						return "", nil
					},
					GetClientSecretFunc: func(internalClientId string, accessToken string) (string, error) {
						return secret, nil
					},
					CreateClientFunc: func(client gocloak.Client, accessToken string) (string, error) {
						return testClientID, nil
					},
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						testID := "12221"
						return gocloak.Client{
							ClientID: &testID,
						}
					},
				},
			},
			want:    secret,
			wantErr: nil,
		},
		{
			name: "failed to register sso client for the osd cluster",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return keycloak.NewKeycloakConfig()
					},
					IsClientExistFunc: func(clientId string, accessToken string) (string, error) {
						return "", nil
					},
					GetClientSecretFunc: func(internalClientId string, accessToken string) (string, error) {
						return secret, nil
					},
					CreateClientFunc: func(client gocloak.Client, accessToken string) (string, error) {
						return "", failedToCreateClientErr
					},
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						testID := "12221"
						return gocloak.Client{
							ClientID: &testID,
						}
					},
				},
			},
			want:    "",
			wantErr: errors.NewWithCause(errors.ErrorFailedToCreateSSOClient, failedToCreateClientErr, "failed to create sso client"),
		},
		{
			name: "returns an error when it fails to find client",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return "", nil
					},
					IsClientExistFunc: func(clientId, accessToken string) (string, error) {
						return "", errors.New(errors.ErrorFailedToGetSSOClient, "failed to get sso client with id: osd-cluster-12212")
					},
				},
			},
			want:    "",
			wantErr: errors.NewWithCause(errors.ErrorFailedToGetSSOClient, errors.FailedToGetSSOClient("failed to get sso client with id: osd-cluster-12212"), "failed to get sso client with id: osd-cluster-12212"),
		},
		{
			name: "successfully register a new sso client for the kafka cluster",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return keycloak.NewKeycloakConfig()
					},
					IsClientExistFunc: func(clientId string, accessToken string) (string, error) {
						return "", nil
					},
					GetClientSecretFunc: func(internalClientId string, accessToken string) (string, error) {
						return "", errors.New(errors.ErrorFailedToGetSSOClientSecret, "failed to get sso client secret")
					},
					CreateClientFunc: func(client gocloak.Client, accessToken string) (string, error) {
						return testClientID, nil
					},
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						testID := "12221"
						return gocloak.Client{
							ClientID: &testID,
						}
					},
				},
			},
			want:    "",
			wantErr: errors.NewWithCause(errors.ErrorFailedToGetSSOClientSecret, errors.FailedToGetSSOClientSecret("failed to get sso client secret"), "failed to get sso client secret"),
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			keycloakService := keycloakServiceProxy{
				getToken: tt.fields.kcClient.GetToken,
				service:  &masService{kcClient: tt.fields.kcClient},
			}
			got, err := keycloakService.RegisterClientInSSO("osd-cluster-12212", "https://oauth-openshift-cluster.fr")
			g.Expect(got).To(gomega.Equal(tt.want))
			g.Expect(err).To(gomega.Equal(tt.wantErr))
		})
	}

}

func TestKeycloakService_RegisterKasFleetshardOperatorServiceAccount(t *testing.T) {
	type fields struct {
		kcClient keycloak.KcClient
	}
	type args struct {
		clusterId string
	}
	fakeRoleId := "1234"
	fakeClientId := "test-client-id"
	fakeClientSecret := "test-client-secret"
	fakeUserId := "test-user-id"
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
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					AddRealmRoleToUserFunc: func(accessToken string, userId string, role gocloak.Role) error {
						return nil
					},
					CreateClientFunc: func(client gocloak.Client, accessToken string) (string, error) {
						return fakeClientId, nil
					},
					GetClientFunc: func(clientId string, accessToken string) (*gocloak.Client, error) {
						return nil, nil
					},
					GetClientSecretFunc: func(internalClientId string, accessToken string) (string, error) {
						return fakeClientSecret, nil
					},
					GetClientServiceAccountFunc: func(accessToken string, internalClient string) (*gocloak.User, error) {
						return &gocloak.User{
							ID: &fakeUserId,
						}, nil
					},
					GetRealmRoleFunc: func(accessToken string, roleName string) (*gocloak.Role, error) {
						return &gocloak.Role{
							ID: &fakeRoleId,
						}, nil
					},
					UpdateServiceAccountUserFunc: func(accessToken string, serviceAccountUser gocloak.User) error {
						return nil
					},
					UserHasRealmRoleFunc: func(accessToken string, userId string, roleName string) (*gocloak.Role, error) {
						return nil, nil
					},
					CreateProtocolMapperConfigFunc: func(in1 string) []gocloak.ProtocolMapperRepresentation {
						return []gocloak.ProtocolMapperRepresentation{{}}
					},
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						return gocloak.Client{}
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
				Name:         "kas-fleetshard-agent-test-cluster-id",
				Description:  "service account for agent on cluster test-cluster-id",
			},
			wantErr: false,
		},
		{
			name: "test registering serviceaccount for agent operator second time",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					GetClientFunc: func(clientId string, accessToken string) (*gocloak.Client, error) {
						return &gocloak.Client{
							ID: &fakeClientId,
						}, nil
					},
					GetClientSecretFunc: func(internalClientId string, accessToken string) (string, error) {
						return fakeClientSecret, nil
					},
					GetClientServiceAccountFunc: func(accessToken string, internalClient string) (*gocloak.User, error) {
						return &gocloak.User{
							ID: &fakeUserId,
							Attributes: &map[string][]string{
								kasClusterId: {"test-cluster-id"},
							},
						}, nil
					},
					GetRealmRoleFunc: func(accessToken string, roleName string) (*gocloak.Role, error) {
						return &gocloak.Role{
							ID: &fakeRoleId,
						}, nil
					},
					UserHasRealmRoleFunc: func(accessToken string, userId string, roleName string) (*gocloak.Role, error) {
						return &gocloak.Role{
							ID: &fakeRoleId,
						}, nil
					},
					CreateProtocolMapperConfigFunc: func(in1 string) []gocloak.ProtocolMapperRepresentation {
						return []gocloak.ProtocolMapperRepresentation{{}}
					},
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						return gocloak.Client{}
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
				Name:         "kas-fleetshard-agent-test-cluster-id",
				Description:  "service account for agent on cluster test-cluster-id",
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
				service:  &masService{kcClient: tt.fields.kcClient},
			}
			got, err := keycloakService.RegisterKasFleetshardOperatorServiceAccount(tt.args.clusterId)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterKasFleetshardOperatorServiceAccount() error = %v, wantErr %v", err, tt.wantErr)
			}
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func TestKeycloakService_DeRegisterKasFleetshardOperatorServiceAccount(t *testing.T) {
	type fields struct {
		kcClient keycloak.KcClient
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
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return "", fmt.Errorf("some errors")
					},
					IsClientExistFunc: func(clientId string, accessToken string) (string, error) {
						return "", nil
					},
					DeleteClientFunc: func(internalClientID, accessToken string) error {
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
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					IsClientExistFunc: func(clientId string, accessToken string) (string, error) {
						return "testclientid", nil
					},
					DeleteClientFunc: func(internalClientID, accessToken string) error {
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
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					IsClientExistFunc: func(clientId string, accessToken string) (string, error) {
						return "testclientid", nil
					},
					DeleteClientFunc: func(internalClientID, accessToken string) error {
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
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					IsClientExistFunc: func(clientId string, accessToken string) (string, error) {
						return "", nil
					},
					DeleteClientFunc: func(internalClientID, accessToken string) error {
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
				service:  &masService{kcClient: tt.fields.kcClient},
			}
			err := keycloakService.DeRegisterKasFleetshardOperatorServiceAccount(tt.args.clusterId)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestKeycloakService_RegisterConnectorFleetshardOperatorServiceAccount(t *testing.T) {
	type fields struct {
		kcClient keycloak.KcClient
	}
	type args struct {
		clusterId string
		roleName  string
	}
	fakeRoleId := "1234"
	fakeClientId := "test-client-id"
	fakeClientSecret := "test-client-secret"
	fakeUserId := "test-user-id"
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
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					AddRealmRoleToUserFunc: func(accessToken string, userId string, role gocloak.Role) error {
						return nil
					},
					CreateClientFunc: func(client gocloak.Client, accessToken string) (string, error) {
						return fakeClientId, nil
					},
					GetClientFunc: func(clientId string, accessToken string) (*gocloak.Client, error) {
						return nil, nil
					},
					GetClientSecretFunc: func(internalClientId string, accessToken string) (string, error) {
						return fakeClientSecret, nil
					},
					GetClientServiceAccountFunc: func(accessToken string, internalClient string) (*gocloak.User, error) {
						return &gocloak.User{
							ID: &fakeUserId,
						}, nil
					},
					GetRealmRoleFunc: func(accessToken string, roleName string) (*gocloak.Role, error) {
						return &gocloak.Role{
							ID:   &fakeRoleId,
							Name: &roleName,
						}, nil
					},
					UpdateServiceAccountUserFunc: func(accessToken string, serviceAccountUser gocloak.User) error {
						return nil
					},
					UserHasRealmRoleFunc: func(accessToken string, userId string, roleName string) (*gocloak.Role, error) {
						return nil, nil
					},
					CreateProtocolMapperConfigFunc: func(in1 string) []gocloak.ProtocolMapperRepresentation {
						return []gocloak.ProtocolMapperRepresentation{{}}
					},
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						return gocloak.Client{}
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
		{
			name: "test registering serviceaccount for agent operator second time",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					GetClientFunc: func(clientId string, accessToken string) (*gocloak.Client, error) {
						return &gocloak.Client{
							ID: &fakeClientId,
						}, nil
					},
					GetClientSecretFunc: func(internalClientId string, accessToken string) (string, error) {
						return fakeClientSecret, nil
					},
					GetClientServiceAccountFunc: func(accessToken string, internalClient string) (*gocloak.User, error) {
						return &gocloak.User{
							ID: &fakeUserId,
							Attributes: &map[string][]string{
								connectorClusterId: {"test-cluster-id"},
							},
						}, nil
					},
					GetRealmRoleFunc: func(accessToken string, roleName string) (*gocloak.Role, error) {
						return &gocloak.Role{
							ID: &fakeRoleId,
						}, nil
					},
					UserHasRealmRoleFunc: func(accessToken string, userId string, roleName string) (*gocloak.Role, error) {
						return &gocloak.Role{
							ID: &fakeRoleId,
						}, nil
					},
					CreateProtocolMapperConfigFunc: func(in1 string) []gocloak.ProtocolMapperRepresentation {
						return []gocloak.ProtocolMapperRepresentation{{}}
					},
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						return gocloak.Client{}
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
			keycloakService := keycloakServiceProxy{
				getToken: tt.fields.kcClient.GetToken,
				service:  &masService{kcClient: tt.fields.kcClient},
			}
			got, err := keycloakService.RegisterConnectorFleetshardOperatorServiceAccount(tt.args.clusterId)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterConnectorFleetshardOperatorServiceAccount() error = %v, wantErr %v", err, tt.wantErr)
			}
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func TestKeycloakService_DeRegisterConnectorFleetshardOperatorServiceAccount(t *testing.T) {
	type fields struct {
		kcClient keycloak.KcClient
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
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return "", fmt.Errorf("some errors")
					},
					IsClientExistFunc: func(clientId string, accessToken string) (string, error) {
						return "", nil
					},
					DeleteClientFunc: func(internalClientID, accessToken string) error {
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
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					IsClientExistFunc: func(clientId string, accessToken string) (string, error) {
						return "testclientid", nil
					},
					DeleteClientFunc: func(internalClientID, accessToken string) error {
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
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					IsClientExistFunc: func(clientId string, accessToken string) (string, error) {
						return "testclientid", nil
					},
					DeleteClientFunc: func(internalClientID, accessToken string) error {
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
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					IsClientExistFunc: func(clientId string, accessToken string) (string, error) {
						return "", nil
					},
					DeleteClientFunc: func(internalClientID, accessToken string) error {
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
				service:  &masService{kcClient: tt.fields.kcClient},
			}
			err := keycloakService.DeRegisterConnectorFleetshardOperatorServiceAccount(tt.args.clusterId)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestKeycloakService_DeleteServiceAccountInternal(t *testing.T) {
	tokenErr := pkgErr.New("token error")

	type fields struct {
		kcClient keycloak.KcClient
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "returns error when failed to fetch token",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return "", tokenErr
					},
				},
			},
			wantErr: true,
		},
		{
			name: "do not return an error when service account deleted successfully",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return "", nil
					},
					DeleteClientFunc: func(internalClientID, accessToken string) error {
						return nil
					},
					IsClientExistFunc: func(clientId, accessToken string) (string, error) {
						return "client-id", nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "do not return an error when service account does not exists",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return "", nil
					},
					DeleteClientFunc: func(internalClientID, accessToken string) error {
						return gocloak.APIError{
							Code: http.StatusNotFound,
						}
					},
					IsClientExistFunc: func(clientId, accessToken string) (string, error) {
						return "client-id", nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "returns an error when failed to delete service account",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return "", nil
					},
					DeleteClientFunc: func(internalClientID, accessToken string) error {
						return gocloak.APIError{
							Code: http.StatusInternalServerError,
						}
					},
					IsClientExistFunc: func(clientId, accessToken string) (string, error) {
						return "client-id", nil
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
				service:  &masService{kcClient: tt.fields.kcClient},
			}
			err := keycloakService.DeleteServiceAccountInternal("account-id")
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}

}

func TestKeycloakService_CreateServiceAccountInternal(t *testing.T) {
	tokenErr := pkgErr.New("token error")
	request := CompleteServiceAccountRequest{
		Owner:          "some-owner",
		OwnerAccountId: "owner-account-id",
		ClientId:       "some-client-id",
		Name:           "some-name",
		Description:    "some-description",
		OrgId:          "some-organization-id",
	}
	type fields struct {
		kcClient keycloak.KcClient
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
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return "", tokenErr
					},
				},
			},
			wantErr:               true,
			serviceAccountCreated: false,
		},
		{
			name: "returns error when failed to create service account",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return "", nil
					},
					CreateProtocolMapperConfigFunc: func(s string) []gocloak.ProtocolMapperRepresentation {
						return []gocloak.ProtocolMapperRepresentation{}
					},
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						return gocloak.Client{}
					},
					CreateClientFunc: func(client gocloak.Client, accessToken string) (string, error) {
						return "", pkgErr.New("failed to create client")
					},
					GetClientFunc: func(clientId, accessToken string) (*gocloak.Client, error) {
						return nil, nil
					},
				},
			},
			wantErr:               true,
			serviceAccountCreated: false,
		},
		{
			name: "succeed to create service account error when failed to create client",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return "", nil
					},
					GetClientFunc: func(clientId, accessToken string) (*gocloak.Client, error) {
						return nil, nil
					},
					CreateProtocolMapperConfigFunc: func(s string) []gocloak.ProtocolMapperRepresentation {
						return []gocloak.ProtocolMapperRepresentation{}
					},
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						return gocloak.Client{}
					},
					CreateClientFunc: func(client gocloak.Client, accessToken string) (string, error) {
						return "dsd", nil
					},
					GetClientSecretFunc: func(internalClientId, accessToken string) (string, error) {
						return "secret", nil
					},
					GetClientServiceAccountFunc: func(accessToken, internalClient string) (*gocloak.User, error) {
						return &gocloak.User{}, nil
					},
					UpdateServiceAccountUserFunc: func(accessToken string, serviceAccountUser gocloak.User) error {
						return nil
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
				service:  &masService{kcClient: tt.fields.kcClient},
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

func TestKeycloakService_checkAllowedServiceAccountsLimits(t *testing.T) {
	type fields struct {
		kcClient keycloak.KcClient
	}
	type args struct {
		accessToken string
		maxAllowed  int
		orgId       string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "Org ID present in the skip list, so no limits apply",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						config := keycloak.NewKeycloakConfig()
						config.ServiceAccounttLimitCheckSkipOrgIdList = []string{"01234", "56789"}
						return config
					},
					GetClientsFunc: func(accesstoken string, first int, max int, searchAttr string) ([]*gocloak.Client, error) {
						var clientArr []*gocloak.Client

						for i := 0; i < 2; i++ {
							client := gocloak.Client{}
							idStr := fmt.Sprintf("srvc-acct-%d", i)
							client.ClientID = &idStr
							clientArr = append(clientArr, &client)
						}
						return clientArr, nil
					},
				},
			},
			args: args{
				accessToken: "bearer: some-random-token",
				maxAllowed:  2,
				orgId:       "01234",
			},

			want:    true,
			wantErr: false,
		},
		{
			name: "Org ID not present in the skip list, limit allows creation of service account",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						config := keycloak.NewKeycloakConfig()
						config.ServiceAccounttLimitCheckSkipOrgIdList = []string{"01234", "56789"}
						return config
					},
					GetClientsFunc: func(accesstoken string, first int, max int, searchAttr string) ([]*gocloak.Client, error) {
						var clientArr []*gocloak.Client

						for i := 0; i < 2; i++ {
							client := gocloak.Client{}
							idStr := fmt.Sprintf("srvc-acct-%d", i)
							client.ClientID = &idStr
							clientArr = append(clientArr, &client)
						}
						return clientArr, nil
					},
				},
			},
			args: args{
				accessToken: "bearer: some-random-token",
				maxAllowed:  3,
				orgId:       "012345",
			},

			want:    true,
			wantErr: false,
		},
		{
			name: "Org ID not present in the skip list, limit disallows creation of service account",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						config := keycloak.NewKeycloakConfig()
						config.ServiceAccounttLimitCheckSkipOrgIdList = []string{"01234", "56789"}
						return config
					},
					GetClientsFunc: func(accesstoken string, first int, max int, searchAttr string) ([]*gocloak.Client, error) {
						var clientArr []*gocloak.Client

						for i := 0; i < 2; i++ {
							client := gocloak.Client{}
							idStr := fmt.Sprintf("srvc-acct-%d", i)
							client.ClientID = &idStr
							clientArr = append(clientArr, &client)
						}
						return clientArr, nil
					},
				},
			},
			args: args{
				accessToken: "bearer: some-random-token",
				maxAllowed:  2,
				orgId:       "012345",
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			service := masService{
				kcClient: tt.fields.kcClient,
			}
			got, err := service.checkAllowedServiceAccountsLimits(tt.args.accessToken, tt.args.maxAllowed, tt.args.orgId)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr), "checkAllowedServiceAccountsLimits() error = %v, wantErr %v", err, tt.wantErr)
			g.Expect(tt.want).To(gomega.Equal(got))
		})
	}
}

func Test_newKeycloakService(t *testing.T) {
	type args struct {
		config      *keycloak.KeycloakConfig
		realmConfig *keycloak.KeycloakRealmConfig
	}
	client := keycloak.NewClient(&keycloak.KeycloakConfig{}, &keycloak.KeycloakRealmConfig{})
	tests := []struct {
		name string
		args args
		want KeycloakService
	}{
		{
			name: "should return new keycloak service",
			args: args{
				config:      &keycloak.KeycloakConfig{},
				realmConfig: &keycloak.KeycloakRealmConfig{},
			},
			want: &keycloakServiceProxy{
				getToken: client.GetToken,
				service: &masService{
					kcClient: client,
				},
			},
		},
	}
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			keycloakService := NewKeycloakServiceBuilder().
				ForKFM().WithConfiguration(tt.args.config).WithRealmConfig(tt.args.realmConfig).Build()
			g.Expect(keycloakService.GetConfig()).To(gomega.Equal(tt.want.GetConfig()))
			g.Expect(keycloakService.GetRealmConfig()).To(gomega.Equal(tt.want.GetRealmConfig()))
		})
	}
}

func Test_masService_DeRegisterClientInSSO(t *testing.T) {
	type fields struct {
		kcClient keycloak.KcClient
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
			name: "should return nil if the client is deregistered successfully",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					IsClientExistFunc: func(clientId, accessToken string) (string, error) {
						return testClientID, nil
					},
					DeleteClientFunc: func(internalClientID, accessToken string) error {
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
			name: "should return nil if the client ID is empty",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					IsClientExistFunc: func(clientId, accessToken string) (string, error) {
						return "", nil
					},
				},
			},
			args: args{
				accessToken: token,
				clientId:    "",
			},
			want: nil,
		},
		{
			name: "should return error if the client is not deregistered",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					IsClientExistFunc: func(clientId, accessToken string) (string, error) {
						return testClientID, nil
					},
					DeleteClientFunc: func(internalClientID, accessToken string) error {
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
			kc := &masService{
				kcClient: tt.fields.kcClient,
			}
			g.Expect(kc.DeRegisterClientInSSO(tt.args.accessToken, tt.args.clientId)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_masService_IsKafkaClientExist(t *testing.T) {
	type fields struct {
		kcClient keycloak.KcClient
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
			name: "should return nil if client exists",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					IsClientExistFunc: func(clientId, accessToken string) (string, error) {
						return testClientID, nil
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
			name: "should return an error if it fails to find the client",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					IsClientExistFunc: func(clientId, accessToken string) (string, error) {
						return "", errors.New(errors.ErrorFailedToGetSSOClient, "failed to get sso client with id: %s", testClientID)
					},
				},
			},
			args: args{
				accessToken: token,
				clientId:    testClientID,
			},
			want: errors.NewWithCause(errors.ErrorFailedToGetSSOClient, errors.FailedToGetSSOClient("failed to get sso client with id: %s", testClientID), "failed to get sso client with id: %s", testClientID),
		},
	}
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			kc := masService{
				kcClient: tt.fields.kcClient,
			}
			g.Expect(kc.IsKafkaClientExist(tt.args.accessToken, tt.args.clientId)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_masService_GetKafkaClientSecret(t *testing.T) {
	type fields struct {
		kcClient keycloak.KcClient
	}
	type args struct {
		accessToken string
		clientId    string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr *errors.ServiceError
	}{
		{
			name: "should return the client secret",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					IsClientExistFunc: func(clientId, accessToken string) (string, error) {
						return testClientID, nil
					},
					GetClientSecretFunc: func(internalClientId, accessToken string) (string, error) {
						return secret, nil
					},
				},
			},
			args: args{
				accessToken: token,
				clientId:    testClientID,
			},
			want:    secret,
			wantErr: nil,
		},
		{
			name: "should return an error in it fails to find the client",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					IsClientExistFunc: func(clientId, accessToken string) (string, error) {
						return "", errors.New(errors.ErrorFailedToGetSSOClient, "failed to get sso client with id: %s", testClientID)
					},
				},
			},
			args: args{
				accessToken: token,
				clientId:    testClientID,
			},
			want:    "",
			wantErr: errors.NewWithCause(errors.ErrorFailedToGetSSOClient, errors.FailedToGetSSOClient("failed to get sso client with id: %s", testClientID), "failed to get sso client with id: %s", testClientID),
		},
		{
			name: "should return an error if it fails to get the client secret",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					IsClientExistFunc: func(clientId, accessToken string) (string, error) {
						return testClientID, nil
					},
					GetClientSecretFunc: func(internalClientId, accessToken string) (string, error) {
						return "", errors.New(errors.ErrorFailedToGetSSOClientSecret, "failed to get sso client secret")
					},
				},
			},
			args: args{
				accessToken: token,
				clientId:    testClientID,
			},
			want:    "",
			wantErr: errors.NewWithCause(errors.ErrorFailedToGetSSOClientSecret, errors.FailedToGetSSOClientSecret("failed to get sso client secret"), "failed to get sso client secret"),
		},
	}
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			kc := masService{
				kcClient: tt.fields.kcClient,
			}
			got, err := kc.GetKafkaClientSecret(tt.args.accessToken, tt.args.clientId)
			g.Expect(err).To(gomega.Equal(tt.wantErr))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_masService_CreateServiceAccount(t *testing.T) {
	type fields struct {
		kcClient keycloak.KcClient
	}
	type args struct {
		accessToken           string
		serviceAccountRequest *api.ServiceAccountRequest
		ctx                   context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *api.ServiceAccount
		wantErr *errors.ServiceError
	}{
		{
			name: "should return the created service account",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetClientFunc: func(clientId, accessToken string) (*gocloak.Client, error) {
						return nil, nil
					},
					CreateProtocolMapperConfigFunc: func(s string) []gocloak.ProtocolMapperRepresentation {
						return []gocloak.ProtocolMapperRepresentation{}
					},
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						return gocloak.Client{}
					},
					CreateClientFunc: func(client gocloak.Client, accessToken string) (string, error) {
						return testClientID, nil
					},
					GetClientSecretFunc: func(internalClientId, accessToken string) (string, error) {
						return secret, nil
					},
					GetClientServiceAccountFunc: func(accessToken, internalClient string) (*gocloak.User, error) {
						return &gocloak.User{}, nil
					},
					UpdateServiceAccountUserFunc: func(accessToken string, serviceAccountUser gocloak.User) error {
						return nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							MaxAllowedServiceAccounts: 1,
						}
					},
					GetClientsFunc: func(accessToken string, first, max int, attribute string) ([]*gocloak.Client, error) {
						return []*gocloak.Client{}, nil
					},
				},
			},
			args: args{
				accessToken: token,
				serviceAccountRequest: &api.ServiceAccountRequest{
					Name:        "ServiceAccountRequest_name",
					Description: "ServiceAccountRequest_description",
				},
				ctx: context.Background(),
			},
			want: &api.ServiceAccount{
				ID:           "12221",
				ClientSecret: "secret",
				Name:         "ServiceAccountRequest_name",
			},
			wantErr: nil,
		},
		{
			name: "should return an error if it fails to create the service account",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetClientFunc: func(clientId, accessToken string) (*gocloak.Client, error) {
						return nil, nil
					},
					CreateProtocolMapperConfigFunc: func(s string) []gocloak.ProtocolMapperRepresentation {
						return []gocloak.ProtocolMapperRepresentation{}
					},
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						return gocloak.Client{}
					},
					CreateClientFunc: func(client gocloak.Client, accessToken string) (string, error) {
						return testClientID, nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							MaxAllowedServiceAccounts: 0,
						}
					},
					GetClientsFunc: func(accessToken string, first, max int, attribute string) ([]*gocloak.Client, error) {
						return []*gocloak.Client{}, nil
					},
				},
			},
			args: args{
				accessToken: token,
				serviceAccountRequest: &api.ServiceAccountRequest{
					Name:        "ServiceAccountRequest_name",
					Description: "ServiceAccountRequest_description",
				},
				ctx: context.Background(),
			},
			want:    nil,
			wantErr: errors.MaxLimitForServiceAccountReached("Max allowed number:0 of service accounts for user in org: has reached"),
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			kc := &masService{
				kcClient: tt.fields.kcClient,
			}
			got, err := kc.CreateServiceAccount(tt.args.accessToken, tt.args.serviceAccountRequest, tt.args.ctx)
			g.Expect(err).To(gomega.Equal(tt.wantErr))

			if tt.want != nil {
				g.Expect(got.ID).To(gomega.Equal(tt.want.ID))
				g.Expect(got.ClientSecret).To(gomega.Equal(tt.want.ClientSecret))
				g.Expect(got.Name).To(gomega.Equal(tt.want.Name))
			}
		})
	}
}

func Test_masService_ListServiceAcc(t *testing.T) {
	type fields struct {
		kcClient keycloak.KcClient
	}
	type args struct {
		accessToken string
		ctx         context.Context
		first       int
		max         int
	}
	id := "id"
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
				kcClient: &keycloak.KcClientMock{
					GetClientsFunc: func(accessToken string, first, max int, attribute string) ([]*gocloak.Client, error) {
						var clientArr []*gocloak.Client

						for i := 0; i < 2; i++ {
							client := gocloak.Client{}
							idStr := fmt.Sprintf("srvc-acct-%d", i)
							client.ClientID = &idStr
							client.Attributes = &map[string]string{}
							client.ID = &id
							clientArr = append(clientArr, &client)
						}
						return clientArr, nil
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
					ID:       "id",
					ClientID: "srvc-acct-0",
				},
				{
					ID:       "id",
					ClientID: "srvc-acct-1",
				},
			},
			wantErr: nil,
		},
		{
			name: "should return an error when it fails to collect service accounts",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetClientsFunc: func(accessToken string, first, max int, attribute string) ([]*gocloak.Client, error) {
						return nil, errors.New(errors.ErrorGeneral, "failed to collect service accounts")
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
			kc := &masService{
				kcClient: tt.fields.kcClient,
			}
			got, err := kc.ListServiceAcc(tt.args.accessToken, tt.args.ctx, tt.args.first, tt.args.max)
			g.Expect(err).To(gomega.Equal(tt.wantErr))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_masService_DeleteServiceAccount(t *testing.T) {
	type fields struct {
		kcClient keycloak.KcClient
	}
	type args struct {
		accessToken string
		ctx         context.Context
		id          string
	}
	clientId := UserServiceAccountPrefix + uuid.New().String()
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *errors.ServiceError
	}{
		{
			name: "return nil if it successfully deletes the service account",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetClientByIdFunc: func(id, accessToken string) (*gocloak.Client, error) {
						return &gocloak.Client{
							ClientID: &clientId,
						}, nil
					},
					IsSameOrgFunc: func(client *gocloak.Client, orgId string) bool {
						return true
					},
					IsOwnerFunc: func(client *gocloak.Client, userId string) bool {
						return true
					},
					DeleteClientFunc: func(internalClientID, accessToken string) error {
						return nil
					},
				},
			},
			args: args{
				accessToken: token,
				ctx:         context.Background(),
				id:          clientId,
			},
			want: nil,
		},
		{
			name: "should return an error if it fails to delete the service account",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetClientByIdFunc: func(id, accessToken string) (*gocloak.Client, error) {
						return &gocloak.Client{
							ClientID: &clientId,
						}, nil
					},
					IsSameOrgFunc: func(client *gocloak.Client, orgId string) bool {
						return true
					},
					IsOwnerFunc: func(client *gocloak.Client, userId string) bool {
						return true
					},
					DeleteClientFunc: func(internalClientID, accessToken string) error {
						return errors.New(errors.ErrorFailedToDeleteServiceAccount, "failed to delete service account")
					},
				},
			},
			args: args{
				accessToken: token,
				ctx:         context.Background(),
				id:          clientId,
			},
			want: errors.NewWithCause(errors.ErrorFailedToDeleteServiceAccount, errors.FailedToDeleteServiceAccount("failed to delete service account"), "failed to delete service account"),
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			kc := &masService{
				kcClient: tt.fields.kcClient,
			}
			g.Expect(kc.DeleteServiceAccount(tt.args.accessToken, tt.args.ctx, tt.args.id)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_masService_ResetServiceAccountCredentials(t *testing.T) {
	type fields struct {
		kcClient keycloak.KcClient
	}
	type args struct {
		accessToken string
		ctx         context.Context
		id          string
	}
	clientId := UserServiceAccountPrefix + uuid.New().String()
	clientSecret := secret
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *api.ServiceAccount
		wantErr *errors.ServiceError
	}{
		{
			name: "should return service account if it successfully resets service account credentials",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetClientByIdFunc: func(id, accessToken string) (*gocloak.Client, error) {
						return &gocloak.Client{
							ClientID:   &clientId,
							Attributes: &map[string]string{},
							ID:         &clientId,
						}, nil
					},
					IsSameOrgFunc: func(client *gocloak.Client, orgId string) bool {
						return true
					},
					IsOwnerFunc: func(client *gocloak.Client, userId string) bool {
						return true
					},
					RegenerateClientSecretFunc: func(accessToken, id string) (*gocloak.CredentialRepresentation, error) {
						return &gocloak.CredentialRepresentation{
							Value: &clientSecret,
						}, nil
					},
				},
			},
			args: args{
				accessToken: token,
				ctx:         context.Background(),
				id:          clientId,
			},
			want: &api.ServiceAccount{
				ID:           clientId,
				ClientID:     clientId,
				ClientSecret: secret,
			},
		},
		{
			name: "should return an error if it fails to resets service account credentials",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetClientByIdFunc: func(id, accessToken string) (*gocloak.Client, error) {
						return &gocloak.Client{
							ClientID:   &clientId,
							Attributes: &map[string]string{},
							ID:         &clientId,
						}, nil
					},
					IsSameOrgFunc: func(client *gocloak.Client, orgId string) bool {
						return true
					},
					IsOwnerFunc: func(client *gocloak.Client, userId string) bool {
						return true
					},
					RegenerateClientSecretFunc: func(accessToken, id string) (*gocloak.CredentialRepresentation, error) {
						return nil, errors.GeneralError("failed to reset service account credentials")
					},
				},
			},
			args: args{
				accessToken: token,
				ctx:         context.Background(),
				id:          clientId,
			},
			want:    nil,
			wantErr: errors.NewWithCause(errors.ErrorGeneral, errors.GeneralError("failed to reset service account credentials"), "failed to reset service account credentials"),
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			kc := &masService{
				kcClient: tt.fields.kcClient,
			}
			got, err := kc.ResetServiceAccountCredentials(tt.args.accessToken, tt.args.ctx, tt.args.id)
			g.Expect(err).To(gomega.Equal(tt.wantErr))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_masService_getServiceAccount(t *testing.T) {
	type fields struct {
		kcClient keycloak.KcClient
	}
	type args struct {
		accessToken   string
		ctx           context.Context
		getClientFunc func(client keycloak.KcClient, accessToken string) (*gocloak.Client, error)
		key           string
	}
	clientId := UserServiceAccountPrefix + uuid.New().String()
	key := "key"
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *api.ServiceAccount
		wantErr *errors.ServiceError
	}{
		{
			name: "should return service account if it successfully retrieves service account",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					IsSameOrgFunc: func(client *gocloak.Client, orgId string) bool {
						return true
					},
					IsOwnerFunc: func(client *gocloak.Client, userId string) bool {
						return true
					},
				},
			},
			args: args{
				accessToken: token,
				ctx:         context.Background(),
				getClientFunc: func(client keycloak.KcClient, accessToken string) (*gocloak.Client, error) {
					return &gocloak.Client{
						ClientID:   &clientId,
						Attributes: &map[string]string{},
						ID:         &clientId,
					}, nil
				},
				key: key,
			},
			want: &api.ServiceAccount{
				ID:       clientId,
				ClientID: clientId,
			},
			wantErr: nil,
		},
		{
			name: "should return error if it fails to find the service account",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					IsSameOrgFunc: func(client *gocloak.Client, orgId string) bool {
						return true
					},
					IsOwnerFunc: func(client *gocloak.Client, userId string) bool {
						return true
					},
				},
			},
			args: args{
				accessToken: token,
				ctx:         context.Background(),
				getClientFunc: func(client keycloak.KcClient, accessToken string) (*gocloak.Client, error) {
					return &gocloak.Client{
						Attributes: &map[string]string{},
						ID:         &clientId,
					}, nil
				},
				key: key,
			},
			want:    nil,
			wantErr: errors.New(errors.ErrorServiceAccountNotFound, "service account not found %s", key),
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)

			kc := &masService{
				kcClient: tt.fields.kcClient,
			}
			got, err := kc.getServiceAccount(tt.args.accessToken, tt.args.ctx, tt.args.getClientFunc, tt.args.key)
			g.Expect(err).To(gomega.Equal(tt.wantErr))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_masService_GetServiceAccountByClientId(t *testing.T) {
	type fields struct {
		kcClient keycloak.KcClient
	}
	type args struct {
		accessToken string
		ctx         context.Context
		clientId    string
	}
	clientId := UserServiceAccountPrefix
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *api.ServiceAccount
		wantErr *errors.ServiceError
	}{
		{
			name: "should return the service account with the client id",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetClientFunc: func(clientId, accessToken string) (*gocloak.Client, error) {
						return &gocloak.Client{
							ClientID:   &clientId,
							Attributes: &map[string]string{},
							ID:         &clientId,
						}, nil
					},
					IsSameOrgFunc: func(client *gocloak.Client, orgId string) bool {
						return true
					},
					IsOwnerFunc: func(client *gocloak.Client, userId string) bool {
						return true
					},
				},
			},
			args: args{
				accessToken: token,
				ctx:         context.Background(),
				clientId:    clientId,
			},
			want: &api.ServiceAccount{
				ID:       "srvc-acct-",
				ClientID: "srvc-acct-",
			},
			wantErr: nil,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			kc := &masService{
				kcClient: tt.fields.kcClient,
			}
			got, err := kc.GetServiceAccountByClientId(tt.args.accessToken, tt.args.ctx, tt.args.clientId)
			g.Expect(err).To(gomega.Equal(tt.wantErr))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_masService_GetServiceAccountById(t *testing.T) {
	type fields struct {
		kcClient keycloak.KcClient
	}
	type args struct {
		accessToken string
		ctx         context.Context
		id          string
	}
	clientId := UserServiceAccountPrefix
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *api.ServiceAccount
		wantErr *errors.ServiceError
	}{
		{
			name: "should return the service account with the client id",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetClientByIdFunc: func(id, accessToken string) (*gocloak.Client, error) {
						return &gocloak.Client{
							ClientID:   &clientId,
							Attributes: &map[string]string{},
							ID:         &clientId,
						}, nil
					},
					IsSameOrgFunc: func(client *gocloak.Client, orgId string) bool {
						return true
					},
					IsOwnerFunc: func(client *gocloak.Client, userId string) bool {
						return true
					},
				},
			},
			args: args{
				accessToken: token,
				ctx:         context.Background(),
				id:          clientId,
			},
			want: &api.ServiceAccount{
				ID:       "srvc-acct-",
				ClientID: "srvc-acct-",
			},
			wantErr: nil,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			kc := &masService{
				kcClient: tt.fields.kcClient,
			}
			got, err := kc.GetServiceAccountById(tt.args.accessToken, tt.args.ctx, tt.args.id)
			g.Expect(err).To(gomega.Equal(tt.wantErr))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}
