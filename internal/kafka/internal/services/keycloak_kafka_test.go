package services

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v8"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/dgrijalva/jwt-go"
)

const (
	token        = "token"
	testClientID = "12221"
	secret       = "secret"
)

func TestKeycloakService_CreateServiceAccount(t *testing.T) {
	type fields struct {
		kcClient keycloak.KcClient
	}

	type args struct {
		serviceAccountRequest *api.ServiceAccountRequest
		ctx                   context.Context
	}

	authHelper, err := auth.NewAuthHelper(JwtKeyFile, JwtCAFile, "")
	if err != nil {
		t.Fatalf("failed to create auth helper: %s", err.Error())
	}
	account, err := authHelper.NewAccount("", "", "", testClientID)
	if err != nil {
		t.Fatal("failed to build a new account")
	}

	jwt, err := authHelper.CreateJWTWithClaims(account, nil)
	if err != nil {
		t.Fatalf("failed to create jwt: %s", err.Error())
	}
	currTime := time.Now().Format(time.RFC3339)
	createdAt, _ := time.Parse(time.RFC3339, currTime)
	testServiceAccount := api.ServiceAccount{
		ID:           testClientID,
		ClientSecret: secret,
		Name:         "test-svc",
		Description:  "desc",
		ClientID:     "srvc-acct-cca1a262-9465-4878-9f76-c3bb59d4b4b5",
		CreatedAt:    createdAt,
	}

	tests := []struct {
		name    string
		fields  fields
		want    *api.ServiceAccount
		wantErr bool
		args    args
	}{
		{
			name: "successfully created a service account in sso",
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
						att := map[string]string{}
						return gocloak.Client{
							ClientID:   &testID,
							Attributes: &att,
						}
					},
					GetClientsFunc: func(accessToken string, first int, max int, attribute string) ([]*gocloak.Client, error) {
						testID := "12221"
						att := map[string]string{}
						clients := []*gocloak.Client{
							{
								ClientID:   &testID,
								Attributes: &att,
							},
						}
						return clients, nil
					},
					GetClientServiceAccountFunc: func(accessToken string, internalClient string) (*gocloak.User, error) {
						id := "1"
						return &gocloak.User{
							ID: &id,
						}, nil
					},
					UpdateServiceAccountUserFunc: func(accessToken string, serviceAccountUser gocloak.User) error {
						return nil
					},
					CreateProtocolMapperConfigFunc: func(name string) []gocloak.ProtocolMapperRepresentation {
						return []gocloak.ProtocolMapperRepresentation{}
					},
					GetClientFunc: func(clientId, accessToken string) (*gocloak.Client, error) {
						return nil, nil
					},
				},
			},
			args: args{
				serviceAccountRequest: &api.ServiceAccountRequest{
					Name:        "test-svc",
					Description: "desc",
				},
				ctx: auth.SetTokenInContext(context.TODO(), jwt),
			},
			want:    &testServiceAccount,
			wantErr: false,
		},
		{
			name: "failed to created a service account in sso due to max allowed limit",
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
						att := map[string]string{}
						return gocloak.Client{
							ClientID:   &testID,
							Attributes: &att,
						}
					},
					GetClientsFunc: func(accessToken string, first int, max int, attribute string) ([]*gocloak.Client, error) {
						testID := "12221"
						testID2 := "21222"
						att := map[string]string{}
						clients := []*gocloak.Client{
							{
								ClientID:   &testID,
								Attributes: &att,
							},
							{
								ClientID:   &testID2,
								Attributes: &att,
							},
						}
						return clients, nil
					},
					GetClientServiceAccountFunc: func(accessToken string, internalClient string) (*gocloak.User, error) {
						id := "1"
						return &gocloak.User{
							ID: &id,
						}, nil
					},
					UpdateServiceAccountUserFunc: func(accessToken string, serviceAccountUser gocloak.User) error {
						return nil
					},
					CreateProtocolMapperConfigFunc: func(name string) []gocloak.ProtocolMapperRepresentation {
						return []gocloak.ProtocolMapperRepresentation{}
					},
					GetClientFunc: func(clientId, accessToken string) (*gocloak.Client, error) {
						return nil, nil
					},
				},
			},
			args: args{
				serviceAccountRequest: &api.ServiceAccountRequest{
					Name:        "test-svc",
					Description: "desc",
				},
				ctx: auth.SetTokenInContext(context.TODO(), jwt),
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// max allowed: 2
			tt.fields.kcClient.GetConfig().MaxAllowedServiceAccounts = 2
			keycloakService := services.NewKeycloakServiceWithClient(tt.fields.kcClient)

			got, err := keycloakService.CreateServiceAccount(tt.args.serviceAccountRequest, tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateServiceAccount() error = %v, wantErr %v", err, tt.wantErr)
			}
			//over-riding the random generate id
			if got != nil {
				got.ClientID = "srvc-acct-cca1a262-9465-4878-9f76-c3bb59d4b4b5"
				got.CreatedAt = createdAt
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateServiceAccount() got = %+v, want %+v", got, tt.want)
			}
		})
	}

}

func TestKeycloakService_DeleteServiceAccount(t *testing.T) {
	type fields struct {
		kcClient keycloak.KcClient
	}

	type args struct {
		ctx context.Context
	}

	authHelper, err := auth.NewAuthHelper(JwtKeyFile, JwtCAFile, "")
	if err != nil {
		t.Fatalf("failed to create auth helper: %s", err.Error())
	}
	account, err := authHelper.NewAccount("", "", "", testClientID)
	if err != nil {
		t.Fatal("failed to build a new account")
	}

	claims := jwt.MapClaims{
		"is_org_admin": true,
	}

	jwt, err := authHelper.CreateJWTWithClaims(account, claims)
	if err != nil {
		t.Fatalf("failed to create jwt: %s", err.Error())
	}

	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
		args    args
	}{
		{
			name: "successfully deleted service account",
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
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						testID := fmt.Sprintf("%s-%s", services.UserServiceAccountPrefix, "12221")
						att := map[string]string{}
						return gocloak.Client{
							ClientID:   &testID,
							Attributes: &att,
						}
					},
					DeleteClientFunc: func(internalClientID string, accessToken string) error {
						return nil
					},
					GetClientByIdFunc: func(id string, accessToken string) (*gocloak.Client, error) {
						testID := fmt.Sprintf("%s-%s", services.UserServiceAccountPrefix, "12221")
						att := map[string]string{}
						return &gocloak.Client{
							ClientID:   &testID,
							Attributes: &att,
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
				ctx: auth.SetTokenInContext(context.TODO(), jwt),
			},
			wantErr: false,
		},
		{
			name: "successfully deleted service account as an admin of the org (not an owner)",
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
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						testID := fmt.Sprintf("%s-%s", services.UserServiceAccountPrefix, "12221")
						att := map[string]string{}
						return gocloak.Client{
							ClientID:   &testID,
							Attributes: &att,
						}
					},
					DeleteClientFunc: func(internalClientID string, accessToken string) error {
						return nil
					},
					GetClientByIdFunc: func(id string, accessToken string) (*gocloak.Client, error) {
						testID := fmt.Sprintf("%s-%s", services.UserServiceAccountPrefix, "12221")
						att := map[string]string{}
						return &gocloak.Client{
							ClientID:   &testID,
							Attributes: &att,
						}, nil
					},
					IsSameOrgFunc: func(client *gocloak.Client, orgId string) bool {
						return true
					},
					IsOwnerFunc: func(client *gocloak.Client, userId string) bool {
						return false
					},
				},
			},
			args: args{
				ctx: auth.SetTokenInContext(context.TODO(), jwt),
			},
			wantErr: false,
		},
		{
			name: "expected deletion failure when admin in different org",
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
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						testID := fmt.Sprintf("%s-%s", services.UserServiceAccountPrefix, "12221")
						att := map[string]string{}
						return gocloak.Client{
							ClientID:   &testID,
							Attributes: &att,
						}
					},
					DeleteClientFunc: func(internalClientID string, accessToken string) error {
						return nil
					},
					GetClientByIdFunc: func(id string, accessToken string) (*gocloak.Client, error) {
						testID := fmt.Sprintf("%s-%s", services.UserServiceAccountPrefix, "12221")
						att := map[string]string{}
						return &gocloak.Client{
							ClientID:   &testID,
							Attributes: &att,
						}, nil
					},
					IsSameOrgFunc: func(client *gocloak.Client, orgId string) bool {
						return false
					},
					IsOwnerFunc: func(client *gocloak.Client, userId string) bool {
						return false
					},
				},
			},
			args: args{
				ctx: auth.SetTokenInContext(context.TODO(), jwt),
			},
			wantErr: true,
		},
		{
			name: "expected deletion failure when not the same org",
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
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						testID := fmt.Sprintf("%s-%s", services.UserServiceAccountPrefix, "12221")
						att := map[string]string{}
						return gocloak.Client{
							ClientID:   &testID,
							Attributes: &att,
						}
					},
					DeleteClientFunc: func(internalClientID string, accessToken string) error {
						return nil
					},
					GetClientByIdFunc: func(id string, accessToken string) (*gocloak.Client, error) {
						testID := fmt.Sprintf("%s-%s", services.UserServiceAccountPrefix, "12221")
						att := map[string]string{}
						return &gocloak.Client{
							ClientID:   &testID,
							Attributes: &att,
						}, nil
					},
					IsSameOrgFunc: func(client *gocloak.Client, orgId string) bool {
						return false
					},
					IsOwnerFunc: func(client *gocloak.Client, userId string) bool {
						return true
					},
				},
			},
			args: args{
				ctx: auth.SetTokenInContext(context.TODO(), jwt),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keycloakService := services.NewKeycloakServiceWithClient(tt.fields.kcClient)
			err := keycloakService.DeleteServiceAccount(tt.args.ctx, testClientID)
			if (err != nil) != tt.wantErr {
				t.Errorf("failed to DeleteServiceAccount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKeycloakService_ListServiceAcc(t *testing.T) {
	type fields struct {
		kcClient keycloak.KcClient
	}

	type args struct {
		ctx context.Context
	}

	var testServiceAcc []api.ServiceAccount

	authHelper, err := auth.NewAuthHelper(JwtKeyFile, JwtCAFile, "")
	if err != nil {
		t.Fatalf("failed to create auth helper: %s", err.Error())
	}
	account, err := authHelper.NewAccount("", "", "", testClientID)
	if err != nil {
		t.Fatal("failed to build a new account")
	}

	jwt, err := authHelper.CreateJWTWithClaims(account, nil)
	if err != nil {
		t.Fatalf("failed to create jwt: %s", err.Error())
	}

	tests := []struct {
		name    string
		fields  fields
		want    []api.ServiceAccount
		wantErr bool
		args    args
	}{
		{
			name: "list service account",
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
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						testID := "12221"
						att := map[string]string{}
						return gocloak.Client{
							ClientID:   &testID,
							Attributes: &att,
						}
					},
					IsSameOrgFunc: func(client *gocloak.Client, orgId string) bool {
						return true
					},
					IsOwnerFunc: func(client *gocloak.Client, userId string) bool {
						return true
					},
					GetClientsFunc: func(accessToken string, first int, max int, attribute string) ([]*gocloak.Client, error) {
						testClient := []*gocloak.Client{}
						return testClient, nil
					},
				},
			},
			args: args{
				ctx: auth.SetTokenInContext(context.TODO(), jwt),
			},
			want:    testServiceAcc,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keycloakService := services.NewKeycloakServiceWithClient(tt.fields.kcClient)
			got, err := keycloakService.ListServiceAcc(tt.args.ctx, 0, 10)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterKafkaClientInSSO() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RegisterKafkaClientInSSO() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestKeycloakService_ResetServiceAccountCredentials(t *testing.T) {
	type fields struct {
		kcClient keycloak.KcClient
	}
	type args struct {
		ctx context.Context
	}

	authHelper, err := auth.NewAuthHelper(JwtKeyFile, JwtCAFile, "")
	if err != nil {
		t.Fatalf("failed to create auth helper: %s", err.Error())
	}
	account, err := authHelper.NewAccount("", "", "", testClientID)
	if err != nil {
		t.Fatal("failed to build a new account")
	}

	claims := jwt.MapClaims{
		"is_org_admin": true,
	}

	jwt, err := authHelper.CreateJWTWithClaims(account, claims)
	if err != nil {
		t.Fatalf("failed to create jwt: %s", err.Error())
	}

	tests := []struct {
		name    string
		fields  fields
		want    *api.ServiceAccount
		args    args
		wantErr bool
	}{
		{
			name: "Reset service account credentials",
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
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						testID := "12221"
						att := map[string]string{}
						return gocloak.Client{
							ClientID:   &testID,
							Attributes: &att,
						}
					},
					IsSameOrgFunc: func(client *gocloak.Client, orgId string) bool {
						return true
					},
					IsOwnerFunc: func(client *gocloak.Client, userId string) bool {
						return true
					},
					GetClientByIdFunc: func(id string, accessToken string) (*gocloak.Client, error) {
						testID := fmt.Sprintf("%s-%s", services.UserServiceAccountPrefix, "12221")
						att := map[string]string{}
						return &gocloak.Client{
							ID:         &testID,
							ClientID:   &testID,
							Attributes: &att,
						}, nil
					},
					RegenerateClientSecretFunc: func(accessToken string, id string) (*gocloak.CredentialRepresentation, error) {
						sec := "secret"
						testID := "12221"
						return &gocloak.CredentialRepresentation{
							Value: &sec,
							ID:    &testID,
						}, nil
					},
				},
			},

			args: args{
				ctx: auth.SetTokenInContext(context.TODO(), jwt),
			},
			want: &api.ServiceAccount{
				ID:           fmt.Sprintf("%s-%s", services.UserServiceAccountPrefix, "12221"),
				ClientID:     fmt.Sprintf("%s-%s", services.UserServiceAccountPrefix, "12221"),
				ClientSecret: secret,
				Name:         "",
				Description:  "",
				CreatedAt:    time.Time{},
			},
			wantErr: false,
		},
		{
			name: "Reset service account credentials as admin (not an owner)",
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
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						testID := "12221"
						att := map[string]string{}
						return gocloak.Client{
							ClientID:   &testID,
							Attributes: &att,
						}
					},
					IsSameOrgFunc: func(client *gocloak.Client, orgId string) bool {
						return true
					},
					IsOwnerFunc: func(client *gocloak.Client, userId string) bool {
						return false
					},
					GetClientByIdFunc: func(id string, accessToken string) (*gocloak.Client, error) {
						testID := fmt.Sprintf("%s-%s", services.UserServiceAccountPrefix, "12221")
						att := map[string]string{}
						return &gocloak.Client{
							ID:         &testID,
							ClientID:   &testID,
							Attributes: &att,
						}, nil
					},
					RegenerateClientSecretFunc: func(accessToken string, id string) (*gocloak.CredentialRepresentation, error) {
						sec := "secret"
						testID := fmt.Sprintf("%s-%s", services.UserServiceAccountPrefix, "12221")
						return &gocloak.CredentialRepresentation{
							Value: &sec,
							ID:    &testID,
						}, nil
					},
				},
			},

			args: args{
				ctx: auth.SetTokenInContext(context.TODO(), jwt),
			},
			want: &api.ServiceAccount{
				ID:           fmt.Sprintf("%s-%s", services.UserServiceAccountPrefix, "12221"),
				ClientID:     fmt.Sprintf("%s-%s", services.UserServiceAccountPrefix, "12221"),
				ClientSecret: secret,
				Name:         "",
				Description:  "",
				CreatedAt:    time.Time{},
			},
			wantErr: false,
		},
		{
			name: "Unsuccessful reset service account credentials as admin (different org)",
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
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						testID := "12221"
						att := map[string]string{}
						return gocloak.Client{
							ClientID:   &testID,
							Attributes: &att,
						}
					},
					IsSameOrgFunc: func(client *gocloak.Client, orgId string) bool {
						return false
					},
					IsOwnerFunc: func(client *gocloak.Client, userId string) bool {
						return false
					},
					GetClientByIdFunc: func(id string, accessToken string) (*gocloak.Client, error) {
						testID := fmt.Sprintf("%s-%s", services.UserServiceAccountPrefix, "12221")
						att := map[string]string{}
						return &gocloak.Client{
							ID:         &testID,
							ClientID:   &testID,
							Attributes: &att,
						}, nil
					},
					RegenerateClientSecretFunc: func(accessToken string, id string) (*gocloak.CredentialRepresentation, error) {
						sec := "secret"
						testID := "12221"
						return &gocloak.CredentialRepresentation{
							Value: &sec,
							ID:    &testID,
						}, nil
					},
				},
			},

			args: args{
				ctx: auth.SetTokenInContext(context.TODO(), jwt),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keycloakService := services.NewKeycloakServiceWithClient(tt.fields.kcClient)
			got, err := keycloakService.ResetServiceAccountCredentials(tt.args.ctx, testClientID)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterKafkaClientInSSO() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RegisterKafkaClientInSSO() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestKeycloakService_GetServiceAccountById(t *testing.T) {
	type fields struct {
		kcClient keycloak.KcClient
	}
	type args struct {
		ctx context.Context
	}

	authHelper, err := auth.NewAuthHelper(JwtKeyFile, JwtCAFile, "")
	if err != nil {
		t.Fatalf("failed to create auth helper: %s", err.Error())
	}
	account, err := authHelper.NewAccount("", "", "", testClientID)
	if err != nil {
		t.Fatal("failed to build a new account")
	}

	jwt, err := authHelper.CreateJWTWithClaims(account, nil)
	if err != nil {
		t.Fatalf("failed to create jwt: %s", err.Error())
	}

	tests := []struct {
		name    string
		fields  fields
		want    *api.ServiceAccount
		args    args
		wantErr bool
	}{
		{
			name: "Get service account by id",
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
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						testID := "12221"
						att := map[string]string{}
						return gocloak.Client{
							ClientID:   &testID,
							Attributes: &att,
						}
					},
					IsSameOrgFunc: func(client *gocloak.Client, orgId string) bool {
						return true
					},
					IsOwnerFunc: func(client *gocloak.Client, userId string) bool {
						return true
					},
					GetClientByIdFunc: func(id string, accessToken string) (*gocloak.Client, error) {
						testID := fmt.Sprintf("%s-%s", services.UserServiceAccountPrefix, "12221")
						att := map[string]string{}
						return &gocloak.Client{
							ID:         &testID,
							ClientID:   &testID,
							Attributes: &att,
						}, nil
					},
				},
			},
			args: args{
				ctx: auth.SetTokenInContext(context.TODO(), jwt),
			},
			want: &api.ServiceAccount{
				ID:          fmt.Sprintf("%s-%s", services.UserServiceAccountPrefix, "12221"),
				ClientID:    fmt.Sprintf("%s-%s", services.UserServiceAccountPrefix, "12221"),
				Name:        "",
				Description: "",
				CreatedAt:   time.Time{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keycloakService := services.NewKeycloakServiceWithClient(tt.fields.kcClient)
			got, err := keycloakService.GetServiceAccountById(tt.args.ctx, testClientID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetServiceAccountById() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetServiceAccountById() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}
