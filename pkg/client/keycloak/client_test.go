package keycloak

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/pkg/errors"

	"github.com/Nerzal/gocloak/v11"
	"github.com/golang-jwt/jwt/v4"
	. "github.com/onsi/gomega"
	"github.com/patrickmn/go-cache"
)

const (
	accessToken      = "accessToken"
	clientID         = "123"
	validIssuerURI   = "testIssuerURI"
	jwtKeyFile       = "test/support/jwt_private_key.pem"
	jwtCAFile        = "test/support/jwt_ca.pem"
	issuerURL        = ""
	JwksEndpointURI  = "JwksEndpointURI"
	Realm            = "realmUno"
	TokenEndpointURI = "TokenEndpointURI"
)

var (
	testValue = "test-value"
	testRole  = gocloak.Role{
		Name: &testValue,
	}
	goCloakNotFoundError = gocloak.APIError{Code: http.StatusNotFound}
	grantType            = "grantType"
	otherClientID        = "456"
	correctClientID      = "123"
	correctInternalID    = "correctID"
	otherInternalID      = "otherID"
)

func Test_kcClient_NewClient(t *testing.T) {
	type args struct {
		config      *KeycloakConfig
		realmConfig *KeycloakRealmConfig
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "should create new keycloak client",
			args: args{
				config:      &KeycloakConfig{},
				realmConfig: &KeycloakRealmConfig{},
			},
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(NewClient(tt.args.config, tt.args.realmConfig)).NotTo(BeNil())
		})
	}
}

func Test_kcClient_ClientConfig(t *testing.T) {
	type args struct {
		client ClientRepresentation
	}
	type fields struct {
		kc *kcClient
	}

	tests := []struct {
		name   string
		args   args
		fields fields
	}{
		{
			name: "should create new client config",
			args: args{
				client: ClientRepresentation{},
			},
			fields: fields{
				kc: NewClient(&KeycloakConfig{}, &KeycloakRealmConfig{}),
			},
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(tt.fields.kc.ClientConfig(tt.args.client)).NotTo(BeNil())
		})
	}
}

func Test_kcClient_CreateProtocolMapperConfig(t *testing.T) {
	type args struct {
		name string
	}
	type fields struct {
		kc *kcClient
	}
	tests := []struct {
		name   string
		args   args
		fields fields
		want   []gocloak.ProtocolMapperRepresentation
	}{
		{
			name: "should create protocol mapper config",
			args: args{
				name: testValue,
			},
			fields: fields{
				kc: NewClient(&KeycloakConfig{}, &KeycloakRealmConfig{}),
			},
			want: []gocloak.ProtocolMapperRepresentation{
				{
					Name:           &testValue,
					Protocol:       &protocol,
					ProtocolMapper: &mapper,
					Config: &map[string]string{
						"access.token.claim":   "true",
						"claim.name":           testValue,
						"id.token.claim":       "true",
						"jsonType.label":       "String",
						"user.attribute":       testValue,
						"userinfo.token.claim": "true",
					},
				},
			},
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(tt.fields.kc.CreateProtocolMapperConfig(tt.args.name)).To(Equal(tt.want))
		})
	}
}

func Test_kcClient_CreateClient(t *testing.T) {
	type args struct {
		client      gocloak.Client
		accessToken string
	}
	type fields struct {
		kc gocloak.GoCloak
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "should create gocloak client and return its id",
			fields: fields{
				kc: &GoCloakMock{
					CreateClientFunc: func(ctx context.Context, accessToken, realm string, newClient gocloak.Client) (string, error) {
						return testValue, nil
					},
				},
			},
			want:    testValue,
			wantErr: false,
		},
		{
			name: "should fail to create client if gocloak CreateClient fails",
			fields: fields{
				kc: &GoCloakMock{
					CreateClientFunc: func(ctx context.Context, accessToken, realm string, newClient gocloak.Client) (string, error) {
						return "", errors.New("test")
					},
				},
			},
			want:    "",
			wantErr: true,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kc := &kcClient{
				kcClient:    tt.fields.kc,
				ctx:         context.Background(),
				realmConfig: &KeycloakRealmConfig{},
			}
			id, err := kc.CreateClient(tt.args.client, tt.args.accessToken)
			Expect(id).To(Equal(tt.want))
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func Test_kcClient_GetToken(t *testing.T) {
	RegisterTestingT(t)
	authHelper, err := auth.NewAuthHelper(jwtKeyFile, jwtCAFile, issuerURL)
	Expect(err).NotTo(HaveOccurred())

	acc, err := authHelper.NewAccount("username", "test-user", "", "org-id-0")
	Expect(err).NotTo(HaveOccurred())

	type fields struct {
		goCloakClient gocloak.GoCloak
		ctx           context.Context
		config        *KeycloakConfig
		realmConfig   *KeycloakRealmConfig
		cache         *cache.Cache
	}

	var goCloakToken gocloak.JWT
	cachedTK := fmt.Sprintf("%s%s", validIssuerURI, clientID)

	claimsExpiredEXP := jwt.MapClaims{
		"typ": "",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Minute * time.Duration(-5)).Unix(),
	}
	jwtTokenExpired, _ := authHelper.CreateSignedJWT(acc, claimsExpiredEXP)
	tests := []struct {
		name         string
		fields       fields
		want         string
		wantErr      bool
		setupFn      func(f *fields)
		wantNewToken bool
	}{
		{
			name: "failed to get token",
			fields: fields{
				realmConfig: &KeycloakRealmConfig{
					ClientID:         clientID,
					GrantType:        grantType,
					ValidIssuerURI:   validIssuerURI,
					TokenEndpointURI: TokenEndpointURI,
					JwksEndpointURI:  JwksEndpointURI,
					Realm:            Realm,
				},
				goCloakClient: &GoCloakMock{
					GetTokenFunc: func(ctx context.Context, realm string, options gocloak.TokenOptions) (*gocloak.JWT, error) {
						return nil, errors.Errorf("failed to get token")
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			wantErr: true,
		},
		{
			name: "successfully get new access token when no token is in cache",
			fields: fields{
				realmConfig: &KeycloakRealmConfig{
					ClientID:         clientID,
					GrantType:        grantType,
					ValidIssuerURI:   validIssuerURI,
					TokenEndpointURI: TokenEndpointURI,
					JwksEndpointURI:  JwksEndpointURI,
					Realm:            Realm,
				},
				goCloakClient: &GoCloakMock{
					GetTokenFunc: func(ctx context.Context, realm string, options gocloak.TokenOptions) (*gocloak.JWT, error) {
						goCloakToken.AccessToken = accessToken
						return &goCloakToken, nil
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			wantErr: false,
			want:    accessToken,
		},
		{
			name: "successfully create new token when token retrieved from cache is expired",
			setupFn: func(f *fields) {
				f.cache.Set(cachedTK, jwtTokenExpired, tokenLifeDuration)
			},
			fields: fields{
				realmConfig: &KeycloakRealmConfig{
					ClientID:         clientID,
					GrantType:        grantType,
					ValidIssuerURI:   validIssuerURI,
					TokenEndpointURI: TokenEndpointURI,
					JwksEndpointURI:  JwksEndpointURI,
					Realm:            Realm,
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
				goCloakClient: &GoCloakMock{
					GetTokenFunc: func(ctx context.Context, realm string, options gocloak.TokenOptions) (*gocloak.JWT, error) {
						goCloakToken.AccessToken = accessToken
						return &goCloakToken, nil
					},
				},
			},
			wantErr:      false,
			want:         accessToken,
			wantNewToken: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn(&tt.fields)
			}
			kc := &kcClient{
				kcClient:    tt.fields.goCloakClient,
				ctx:         tt.fields.ctx,
				config:      tt.fields.config,
				realmConfig: tt.fields.realmConfig,
				cache:       tt.fields.cache,
			}
			cachedToken, err := kc.GetToken()

			Expect(err != nil).To(Equal(tt.wantErr))
			if cachedToken != "" && tt.wantNewToken {
				Expect(goCloakToken.AccessToken).To(Equal(tt.want))
			}
		})
	}
}

func Test_kcClient_IsClientExist(t *testing.T) {
	type fields struct {
		goCloakClient gocloak.GoCloak
		realmConfig   *KeycloakRealmConfig
	}

	type args struct {
		requestClientId string
		accessToken     string
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "error when no client exists with request clientId",
			fields: fields{
				goCloakClient: &GoCloakMock{
					GetClientsFunc: func(ctx context.Context, accessToken, realm string, params gocloak.GetClientsParams) ([]*gocloak.Client, error) {
						return nil, errors.Errorf("no client exists with requested clientId")
					},
				},
				realmConfig: &KeycloakRealmConfig{
					ClientID:         clientID,
					GrantType:        grantType,
					ValidIssuerURI:   validIssuerURI,
					TokenEndpointURI: TokenEndpointURI,
					JwksEndpointURI:  JwksEndpointURI,
					Realm:            Realm,
				},
			},
			args: args{
				requestClientId: otherClientID,
			},
			wantErr: true,
			want:    "",
		},
		{
			name: "success when correct internal ID is returned",
			fields: fields{
				goCloakClient: &GoCloakMock{
					GetClientsFunc: func(ctx context.Context, accessToken, realm string, params gocloak.GetClientsParams) ([]*gocloak.Client, error) {
						return []*gocloak.Client{
							{
								ClientID: &otherClientID,
								ID:       &otherInternalID,
							},
							{
								ClientID: &correctClientID,
								ID:       &correctInternalID,
							},
						}, nil
					},
				},
				realmConfig: &KeycloakRealmConfig{
					ClientID:         clientID,
					GrantType:        grantType,
					ValidIssuerURI:   validIssuerURI,
					TokenEndpointURI: TokenEndpointURI,
					JwksEndpointURI:  JwksEndpointURI,
					Realm:            Realm,
				},
			},
			args: args{
				requestClientId: correctClientID,
			},
			wantErr: false,
			want:    correctInternalID,
		},
		{
			name: "empty string returned when no clients exist",
			fields: fields{
				goCloakClient: &GoCloakMock{
					GetClientsFunc: func(ctx context.Context, accessToken, realm string, params gocloak.GetClientsParams) ([]*gocloak.Client, error) {
						return []*gocloak.Client{}, nil
					},
				},
				realmConfig: &KeycloakRealmConfig{
					ClientID:         clientID,
					GrantType:        grantType,
					ValidIssuerURI:   validIssuerURI,
					TokenEndpointURI: TokenEndpointURI,
					JwksEndpointURI:  JwksEndpointURI,
					Realm:            Realm,
				},
			},
			args: args{
				requestClientId: correctClientID,
			},
			wantErr: false,
			want:    "",
		},
		{
			name: "should fail when cliendId param is empty",
			args: args{
				requestClientId: "",
			},
			wantErr: true,
			want:    "",
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kc := &kcClient{
				kcClient:    tt.fields.goCloakClient,
				realmConfig: tt.fields.realmConfig,
			}
			internalId, err := kc.IsClientExist(tt.args.requestClientId, tt.args.accessToken)

			Expect(err != nil).To(Equal(tt.wantErr))
			Expect(internalId).To(Equal(tt.want))

		})
	}
}

func Test_kcClient_GetClient(t *testing.T) {
	type fields struct {
		goCloakClient gocloak.GoCloak
		realmConfig   *KeycloakRealmConfig
	}

	type args struct {
		requestClientId string
		accessToken     string
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "error when no client exists with request clientId",
			fields: fields{
				goCloakClient: &GoCloakMock{
					GetClientsFunc: func(ctx context.Context, accessToken, realm string, params gocloak.GetClientsParams) ([]*gocloak.Client, error) {
						return nil, errors.Errorf("no client exists with requested clientId")
					},
				},
				realmConfig: &KeycloakRealmConfig{
					ClientID:         clientID,
					GrantType:        grantType,
					ValidIssuerURI:   validIssuerURI,
					TokenEndpointURI: TokenEndpointURI,
					JwksEndpointURI:  JwksEndpointURI,
					Realm:            Realm,
				},
			},
			args: args{
				requestClientId: otherClientID,
			},
			wantErr: true,
		},
		{
			name: "success when correct internal ID is returned",
			fields: fields{
				goCloakClient: &GoCloakMock{
					GetClientsFunc: func(ctx context.Context, accessToken, realm string, params gocloak.GetClientsParams) ([]*gocloak.Client, error) {
						return []*gocloak.Client{
							{
								ClientID: &otherClientID,
								ID:       &otherInternalID,
							},
							{
								ClientID: &correctClientID,
								ID:       &correctInternalID,
							},
						}, nil
					},
				},
				realmConfig: &KeycloakRealmConfig{
					ClientID:         clientID,
					GrantType:        grantType,
					ValidIssuerURI:   validIssuerURI,
					TokenEndpointURI: TokenEndpointURI,
					JwksEndpointURI:  JwksEndpointURI,
					Realm:            Realm,
				},
			},
			args: args{
				requestClientId: "123",
			},
			wantErr: false,
			want:    correctInternalID,
		},
		{
			name: "should return nil if no error is thrown and no clients are returned",
			fields: fields{
				goCloakClient: &GoCloakMock{
					GetClientsFunc: func(ctx context.Context, accessToken, realm string, params gocloak.GetClientsParams) ([]*gocloak.Client, error) {
						return []*gocloak.Client{}, nil
					},
				},
				realmConfig: &KeycloakRealmConfig{
					ClientID:         clientID,
					GrantType:        grantType,
					ValidIssuerURI:   validIssuerURI,
					TokenEndpointURI: TokenEndpointURI,
					JwksEndpointURI:  JwksEndpointURI,
					Realm:            Realm,
				},
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kc := &kcClient{
				kcClient:    tt.fields.goCloakClient,
				realmConfig: tt.fields.realmConfig,
			}
			client, err := kc.GetClient(tt.args.requestClientId, tt.args.accessToken)

			Expect(err != nil).To(Equal(tt.wantErr))
			if client != nil {
				Expect(*client.ID).To(Equal(tt.want))
			}
		})
	}
}

func Test_kcClient_GetClientSecret(t *testing.T) {
	type args struct {
		clientID    string
		accessToken string
	}
	type fields struct {
		kc gocloak.GoCloak
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "should return an error when gocloak throws an error when attempting to get client secret",
			args: args{
				clientID:    clientID,
				accessToken: accessToken,
			},
			fields: fields{
				kc: &GoCloakMock{
					GetClientSecretFunc: func(ctx context.Context, token, realm, idOfClient string) (*gocloak.CredentialRepresentation, error) {
						return nil, errors.Errorf("failed to get client secret")
					},
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "should fail if gocloak get client secret returns no value",
			args: args{
				clientID:    clientID,
				accessToken: accessToken,
			},
			fields: fields{
				kc: &GoCloakMock{
					GetClientSecretFunc: func(ctx context.Context, token, realm, idOfClient string) (*gocloak.CredentialRepresentation, error) {
						return &gocloak.CredentialRepresentation{}, nil
					},
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "should return client secret",
			args: args{
				clientID:    clientID,
				accessToken: accessToken,
			},
			fields: fields{
				kc: &GoCloakMock{
					GetClientSecretFunc: func(ctx context.Context, token, realm, idOfClient string) (*gocloak.CredentialRepresentation, error) {
						return &gocloak.CredentialRepresentation{
							Value: &testValue,
						}, nil
					},
				},
			},
			want:    testValue,
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kc := &kcClient{
				kcClient:    tt.fields.kc,
				ctx:         context.Background(),
				realmConfig: &KeycloakRealmConfig{},
			}
			id, err := kc.GetClientSecret(tt.args.clientID, tt.args.accessToken)
			Expect(id).To(Equal(tt.want))
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func Test_kcClient_DeleteClient(t *testing.T) {
	type args struct {
		clientID    string
		accessToken string
	}
	type fields struct {
		kc gocloak.GoCloak
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr bool
	}{
		{
			name: "should return an error when gocloak throws an error when deleting a client",
			args: args{
				clientID:    clientID,
				accessToken: accessToken,
			},
			fields: fields{
				kc: &GoCloakMock{
					DeleteClientFunc: func(ctx context.Context, token, realm, idOfClient string) error {
						return errors.Errorf("failed to delete client")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should return no error if client deletion is successful",
			args: args{
				clientID:    clientID,
				accessToken: accessToken,
			},
			fields: fields{
				kc: &GoCloakMock{
					DeleteClientFunc: func(ctx context.Context, token, realm, idOfClient string) error {
						return nil
					},
				},
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kc := &kcClient{
				kcClient:    tt.fields.kc,
				ctx:         context.Background(),
				realmConfig: &KeycloakRealmConfig{},
			}
			Expect(kc.DeleteClient(clientID, tt.args.accessToken) != nil).To(Equal(tt.wantErr))
		})
	}
}

func Test_kcClient_GetClientById(t *testing.T) {
	type args struct {
		clientID    string
		accessToken string
	}
	type fields struct {
		kc gocloak.GoCloak
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "should return an error when gocloak fails to get client by id",
			args: args{
				clientID:    clientID,
				accessToken: accessToken,
			},
			fields: fields{
				kc: &GoCloakMock{
					GetClientFunc: func(ctx context.Context, token, realm, idOfClient string) (*gocloak.Client, error) {
						return nil, errors.Errorf("failed to get client by id")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should return client by id",
			args: args{
				clientID:    clientID,
				accessToken: accessToken,
			},
			fields: fields{
				kc: &GoCloakMock{
					GetClientFunc: func(ctx context.Context, token, realm, idOfClient string) (*gocloak.Client, error) {
						return &gocloak.Client{
							ID: &testValue,
						}, nil
					},
				},
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kc := &kcClient{
				kcClient:    tt.fields.kc,
				ctx:         context.Background(),
				realmConfig: &KeycloakRealmConfig{},
			}
			client, err := kc.GetClientById(tt.args.clientID, tt.args.accessToken)
			Expect(client == nil).To(Equal(tt.wantErr))
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func Test_kcClient_GetConfig(t *testing.T) {
	tests := []struct {
		name string
		want *KeycloakConfig
	}{
		{
			name: "should return KeycloakConfig",
			want: &KeycloakConfig{},
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kc := &kcClient{
				config: &KeycloakConfig{},
			}
			Expect(kc.GetConfig()).To(Equal(tt.want))
		})
	}
}

func Test_kcClient_KeycloakRealmConfig(t *testing.T) {
	tests := []struct {
		name string
		want *KeycloakRealmConfig
	}{
		{
			name: "should return KeycloakRealmConfig",
			want: &KeycloakRealmConfig{},
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kc := &kcClient{realmConfig: &KeycloakRealmConfig{}}
			Expect(kc.GetRealmConfig()).To(Equal(tt.want))
		})
	}
}

func Test_kcClient_GetClientServiceAccount(t *testing.T) {
	type args struct {
		internalClient string
		accessToken    string
	}
	type fields struct {
		kc gocloak.GoCloak
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		want    *gocloak.User
		wantErr bool
	}{
		{
			name: "should return an error if gocloak fails to get service account",
			args: args{
				internalClient: testValue,
				accessToken:    accessToken,
			},
			fields: fields{
				kc: &GoCloakMock{
					GetClientServiceAccountFunc: func(ctx context.Context, token, realm, idOfClient string) (*gocloak.User, error) {
						return nil, errors.Errorf("failed to get service account")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should return client service account",
			args: args{
				internalClient: testValue,
				accessToken:    accessToken,
			},
			fields: fields{
				kc: &GoCloakMock{
					GetClientServiceAccountFunc: func(ctx context.Context, token, realm, idOfClient string) (*gocloak.User, error) {
						return &gocloak.User{}, nil
					},
				},
			},
			want:    &gocloak.User{},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kc := &kcClient{
				kcClient:    tt.fields.kc,
				ctx:         context.Background(),
				realmConfig: &KeycloakRealmConfig{},
			}
			svcAcc, err := kc.GetClientServiceAccount(tt.args.accessToken, tt.args.internalClient)
			Expect(svcAcc == nil).To(Equal(tt.wantErr))
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func Test_kcClient_UpdateUser(t *testing.T) {
	type args struct {
		accessToken        string
		serviceAccountUser gocloak.User
	}
	type fields struct {
		kc gocloak.GoCloak
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr bool
	}{
		{
			name: "should return an error when gocloak throws an error when updating service account",
			args: args{
				accessToken:        accessToken,
				serviceAccountUser: gocloak.User{},
			},
			fields: fields{
				kc: &GoCloakMock{
					UpdateUserFunc: func(ctx context.Context, accessToken, realm string, user gocloak.User) error {
						return errors.Errorf("failed to update service account")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should return no error if client update is successful",
			args: args{
				accessToken:        accessToken,
				serviceAccountUser: gocloak.User{},
			},
			fields: fields{
				kc: &GoCloakMock{
					UpdateUserFunc: func(ctx context.Context, accessToken, realm string, user gocloak.User) error {
						return nil
					},
				},
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kc := &kcClient{
				kcClient:    tt.fields.kc,
				ctx:         context.Background(),
				realmConfig: &KeycloakRealmConfig{},
			}
			Expect(kc.UpdateServiceAccountUser(tt.args.accessToken, tt.args.serviceAccountUser) != nil).To(Equal(tt.wantErr))
		})
	}
}

func Test_kcClient_GetClients(t *testing.T) {
	type args struct {
		accessToken, attribute string
		first, max             int
	}
	type fields struct {
		kc gocloak.GoCloak
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		want    []*gocloak.Client
		wantErr bool
	}{
		{
			name: "should return an error when gocloak fails to get clients",
			args: args{
				accessToken: accessToken,
				attribute:   testValue,
				first:       0,
				max:         0,
			},
			fields: fields{
				kc: &GoCloakMock{
					GetClientsFunc: func(ctx context.Context, accessToken, realm string, params gocloak.GetClientsParams) ([]*gocloak.Client, error) {
						return nil, errors.Errorf("failed to get clients")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should get clients without max param set",
			args: args{
				accessToken: accessToken,
				attribute:   testValue,
				first:       0,
			},
			fields: fields{
				kc: &GoCloakMock{
					GetClientsFunc: func(ctx context.Context, accessToken, realm string, params gocloak.GetClientsParams) ([]*gocloak.Client, error) {
						return []*gocloak.Client{}, nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should get clients with max param set higher than 0",
			args: args{
				accessToken: accessToken,
				attribute:   testValue,
				first:       0,
			},
			fields: fields{
				kc: &GoCloakMock{
					GetClientsFunc: func(ctx context.Context, accessToken, realm string, params gocloak.GetClientsParams) ([]*gocloak.Client, error) {
						return []*gocloak.Client{
							{
								ClientID: &testValue,
							},
						}, nil
					},
				},
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kc := &kcClient{
				kcClient: tt.fields.kc,
				config: &KeycloakConfig{
					MaxLimitForGetClients: 5,
				},
				ctx:         context.Background(),
				realmConfig: &KeycloakRealmConfig{},
			}
			clients, err := kc.GetClients(tt.args.accessToken, tt.args.first, tt.args.max, tt.args.attribute)
			Expect(clients == nil).To(Equal(tt.wantErr))
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func Test_kcClient_IsSameOrg(t *testing.T) {
	type args struct {
		client *gocloak.Client
		orgId  string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "should return true if the same org",
			args: args{
				orgId: testValue,
				client: &gocloak.Client{
					Attributes: &map[string]string{OrgKey: testValue},
				},
			},
			want: true,
		},
		{
			name: "should return false if not the same org",
			args: args{
				orgId: "other-org-id",
				client: &gocloak.Client{
					Attributes: &map[string]string{OrgKey: testValue},
				},
			},
			want: false,
		},
		{
			name: "should return false if orgId param is empty",
			args: args{
				orgId: "",
				client: &gocloak.Client{
					Attributes: &map[string]string{OrgKey: testValue},
				},
			},
			want: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kc := &kcClient{}
			Expect(kc.IsSameOrg(tt.args.client, tt.args.orgId)).To(Equal(tt.want))
		})
	}
}

func Test_kcClient_IsOwner(t *testing.T) {
	type args struct {
		client *gocloak.Client
		userId string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "should return true if the client is an owner",
			args: args{
				userId: testValue,
				client: &gocloak.Client{
					Attributes: &map[string]string{UserKey: testValue},
				},
			},
			want: true,
		},
		{
			name: "should return false if client is an owner",
			args: args{
				userId: "other-id",
				client: &gocloak.Client{
					Attributes: &map[string]string{UserKey: testValue},
				},
			},
			want: false,
		},
		{
			name: "should return false if userId param is empty",
			args: args{
				userId: "",
				client: &gocloak.Client{
					Attributes: &map[string]string{UserKey: testValue},
				},
			},
			want: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kc := &kcClient{}
			Expect(kc.IsOwner(tt.args.client, tt.args.userId)).To(Equal(tt.want))
		})
	}
}

func Test_kcClient_RegenerateClientSecret(t *testing.T) {
	type args struct {
		id          string
		accessToken string
	}
	type fields struct {
		kc gocloak.GoCloak
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		want    *gocloak.CredentialRepresentation
		wantErr bool
	}{
		{
			name: "should return an error if gocloak fails to regenerate service account credentials",
			args: args{
				id:          testValue,
				accessToken: accessToken,
			},
			fields: fields{
				kc: &GoCloakMock{
					RegenerateClientSecretFunc: func(ctx context.Context, token string, realm string, idOfClient string) (*gocloak.CredentialRepresentation, error) {
						return nil, errors.Errorf("failed to regenerate service account credentials")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should successfully regenerate service account credentials",
			args: args{
				id:          testValue,
				accessToken: accessToken,
			},
			fields: fields{
				kc: &GoCloakMock{
					RegenerateClientSecretFunc: func(ctx context.Context, token string, realm string, idOfClient string) (*gocloak.CredentialRepresentation, error) {
						return &gocloak.CredentialRepresentation{SecretData: &testValue}, nil
					},
				},
			},
			want:    &gocloak.CredentialRepresentation{SecretData: &testValue},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kc := &kcClient{
				kcClient:    tt.fields.kc,
				ctx:         context.Background(),
				realmConfig: &KeycloakRealmConfig{},
			}
			credentials, err := kc.RegenerateClientSecret(tt.args.accessToken, tt.args.id)
			Expect(credentials).To(Equal(tt.want))
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func Test_kcClient_GetRealmRole(t *testing.T) {
	type args struct {
		roleName    string
		accessToken string
	}
	type fields struct {
		kc gocloak.GoCloak
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		want    *gocloak.Role
		wantErr bool
	}{
		{
			name: "should return an error if gocloak fails to get realm role",
			args: args{
				roleName:    testValue,
				accessToken: accessToken,
			},
			fields: fields{
				kc: &GoCloakMock{
					GetRealmRoleFunc: func(ctx context.Context, token, realm, roleName string) (*gocloak.Role, error) {
						return nil, errors.Errorf("failed to get realm role")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should return no error and no role, if role is not found",
			args: args{
				roleName:    testValue,
				accessToken: accessToken,
			},
			fields: fields{
				kc: &GoCloakMock{
					GetRealmRoleFunc: func(ctx context.Context, token, realm, roleName string) (*gocloak.Role, error) {
						return nil, &goCloakNotFoundError
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should return realm role by its name",
			args: args{
				roleName:    testValue,
				accessToken: accessToken,
			},
			fields: fields{
				kc: &GoCloakMock{
					GetRealmRoleFunc: func(ctx context.Context, token, realm, roleName string) (*gocloak.Role, error) {
						return &gocloak.Role{}, nil
					},
				},
			},
			want:    &gocloak.Role{},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kc := &kcClient{
				kcClient:    tt.fields.kc,
				ctx:         context.Background(),
				realmConfig: &KeycloakRealmConfig{},
			}
			role, err := kc.GetRealmRole(tt.args.accessToken, tt.args.roleName)
			Expect(role).To(Equal(tt.want))
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func Test_kcClient_CreateRealmRole(t *testing.T) {
	type args struct {
		roleName    string
		accessToken string
	}
	type fields struct {
		kc gocloak.GoCloak
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		want    *gocloak.Role
		wantErr bool
	}{
		{
			name: "should return an error if gocloak fails to get create realm role account",
			args: args{
				roleName:    testValue,
				accessToken: accessToken,
			},
			fields: fields{
				kc: &GoCloakMock{
					CreateRealmRoleFunc: func(ctx context.Context, token, realm string, role gocloak.Role) (string, error) {
						return "", errors.Errorf("failed to create realm role")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should return an error if gocloak fails to get created realm role",
			args: args{
				roleName:    testValue,
				accessToken: accessToken,
			},
			fields: fields{
				kc: &GoCloakMock{
					CreateRealmRoleFunc: func(ctx context.Context, token, realm string, role gocloak.Role) (string, error) {
						return testValue, nil
					},
					GetRealmRoleFunc: func(ctx context.Context, token, realm, roleName string) (*gocloak.Role, error) {
						return nil, errors.Errorf("failed to get realm role")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should successfully create and return created role",
			args: args{
				roleName:    testValue,
				accessToken: accessToken,
			},
			fields: fields{
				kc: &GoCloakMock{
					CreateRealmRoleFunc: func(ctx context.Context, token, realm string, role gocloak.Role) (string, error) {
						return testValue, nil
					},
					GetRealmRoleFunc: func(ctx context.Context, token, realm, roleName string) (*gocloak.Role, error) {
						return &gocloak.Role{}, nil
					},
				},
			},
			want:    &gocloak.Role{},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kc := &kcClient{
				kcClient:    tt.fields.kc,
				ctx:         context.Background(),
				realmConfig: &KeycloakRealmConfig{},
			}
			role, err := kc.CreateRealmRole(tt.args.accessToken, tt.args.roleName)
			Expect(role).To(Equal(tt.want))
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func Test_kcClient_UserHasRealmRole(t *testing.T) {
	type args struct {
		roleName    string
		userId      string
		accessToken string
	}
	type fields struct {
		kc gocloak.GoCloak
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		want    *gocloak.Role
		wantErr bool
	}{
		{
			name: "should return an error if gocloak fails to get realm roles by userId",
			args: args{
				roleName:    testValue,
				userId:      testValue,
				accessToken: accessToken,
			},
			fields: fields{
				kc: &GoCloakMock{
					GetRealmRolesByUserIDFunc: func(ctx context.Context, accessToken, realm, userID string) ([]*gocloak.Role, error) {
						return nil, errors.Errorf("failed to get realm roles by userId")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should return no error and no role, if roles returned by gocloak",
			args: args{
				roleName:    testValue,
				userId:      testValue,
				accessToken: accessToken,
			},
			fields: fields{
				kc: &GoCloakMock{
					GetRealmRolesByUserIDFunc: func(ctx context.Context, accessToken, realm, userID string) ([]*gocloak.Role, error) {
						return nil, nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should return realm role by its name for user",
			args: args{
				roleName:    testValue,
				userId:      testValue,
				accessToken: accessToken,
			},
			fields: fields{
				kc: &GoCloakMock{
					GetRealmRolesByUserIDFunc: func(ctx context.Context, accessToken, realm, userID string) ([]*gocloak.Role, error) {
						return []*gocloak.Role{
							&testRole,
						}, nil
					},
				},
			},
			want:    &testRole,
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kc := &kcClient{
				kcClient:    tt.fields.kc,
				ctx:         context.Background(),
				realmConfig: &KeycloakRealmConfig{},
			}
			role, err := kc.UserHasRealmRole(tt.args.accessToken, tt.args.userId, tt.args.roleName)
			Expect(role).To(Equal(tt.want))
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func Test_kcClient_AddRealmRoleToUser(t *testing.T) {
	type args struct {
		role        gocloak.Role
		userId      string
		accessToken string
	}
	type fields struct {
		kc gocloak.GoCloak
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr bool
	}{
		{
			name: "should return an error if gocloak fails to add role to user",
			args: args{
				role:        testRole,
				userId:      testValue,
				accessToken: accessToken,
			},
			fields: fields{
				kc: &GoCloakMock{
					AddRealmRoleToUserFunc: func(ctx context.Context, token, realm, userID string, roles []gocloak.Role) error {
						return errors.Errorf("failed to add realm role to a user")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should add realm role to user",
			args: args{
				role:        testRole,
				userId:      testValue,
				accessToken: accessToken,
			},
			fields: fields{
				kc: &GoCloakMock{
					AddRealmRoleToUserFunc: func(ctx context.Context, token, realm, userID string, roles []gocloak.Role) error {
						return nil
					},
				},
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kc := &kcClient{
				kcClient:    tt.fields.kc,
				ctx:         context.Background(),
				realmConfig: &KeycloakRealmConfig{},
			}
			Expect(kc.AddRealmRoleToUser(tt.args.accessToken, tt.args.userId, tt.args.role) != nil).To(Equal(tt.wantErr))
		})
	}
}

func Test_kcClient_isNotFoundError(t *testing.T) {
	type args struct {
		err error
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "should return false if the error is not a 404 and not gocloak error",
			args: args{
				err: errors.New("test"),
			},
			want: false,
		},
		{
			name: "should return false if the error is not a 404 gocloak error",
			args: args{
				err: &gocloak.APIError{Code: http.StatusNotImplemented},
			},
			want: false,
		},
		{
			name: "should return true if the error is a 404 gocloak error",
			args: args{
				err: &goCloakNotFoundError,
			},
			want: true,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(isNotFoundError(tt.args.err)).To(Equal(tt.want))
		})
	}
}
