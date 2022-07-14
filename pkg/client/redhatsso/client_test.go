package redhatsso

import (
	"fmt"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/onsi/gomega"
	"github.com/patrickmn/go-cache"
	serviceaccountsclient "github.com/redhat-developer/app-services-sdk-go/serviceaccounts/apiv1internal/client"
)

const (
	accountName        = "serviceAccount"
	accountDescription = "fake service account"
)

func CreateServiceAccountForTests(accessToken string, server mocks.RedhatSSOMock, accountName string, accountDescription string) serviceaccountsclient.ServiceAccountData {
	c := &rhSSOClient{
		realmConfig: &keycloak.KeycloakRealmConfig{
			ClientID:         "",
			ClientSecret:     "",
			Realm:            "redhat-external",
			APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
			TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
		},
		configuration: &serviceaccountsclient.Configuration{
			DefaultHeader: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				"Content-Type":  "application/json",
			},
			UserAgent: "OpenAPI-Generator/1.0.0/go",
			Debug:     false,
			Servers: serviceaccountsclient.ServerConfigurations{
				{
					URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
				},
			},
		},
	}
	serviceAccount, _ := c.CreateServiceAccount(accessToken, accountName, accountDescription)
	return serviceAccount
}

func TestNewSSOClient(t *testing.T) {
	type args struct {
		config      *keycloak.KeycloakConfig
		realmConfig *keycloak.KeycloakRealmConfig
	}
	tests := []struct {
		name string
		args args
		want SSOClient
	}{
		{
			name: "should successfully return a new sso client",
			args: args{
				config: &keycloak.KeycloakConfig{
					EnableAuthenticationOnKafka: true,
					BaseURL:                     "base_url",
				},
				realmConfig: &keycloak.KeycloakRealmConfig{
					ClientID:     "Client_Id",
					ClientSecret: "ClientSecret",
				},
			},
			want: &rhSSOClient{
				config: &keycloak.KeycloakConfig{
					EnableAuthenticationOnKafka: true,
					BaseURL:                     "base_url",
				},
				realmConfig: &keycloak.KeycloakRealmConfig{
					ClientID:     "Client_Id",
					ClientSecret: "ClientSecret",
				},
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			ssoClient := NewSSOClient(tt.args.config, tt.args.realmConfig)
			g.Expect(ssoClient.GetConfig()).To(gomega.Equal(tt.want.GetConfig()))
			g.Expect(ssoClient.GetRealmConfig()).To(gomega.Equal(tt.want.GetRealmConfig()))
		})
	}
}

func Test_rhSSOClient_getConfiguration(t *testing.T) {
	type fields struct {
		config        *keycloak.KeycloakConfig
		realmConfig   *keycloak.KeycloakRealmConfig
		configuration *serviceaccountsclient.Configuration
		cache         *cache.Cache
	}
	type args struct {
		accessToken string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *serviceaccountsclient.Configuration
	}{
		{
			name: "should return the clients configuration",
			fields: fields{
				config:        &keycloak.KeycloakConfig{},
				realmConfig:   &keycloak.KeycloakRealmConfig{},
				configuration: nil,
				cache:         cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				accessToken: "accessToken",
			},
			want: &serviceaccountsclient.Configuration{
				DefaultHeader: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", "accessToken"),
					"Content-Type":  "application/json",
				},
				UserAgent: "OpenAPI-Generator/1.0.0/go",
				Debug:     false,
				Servers: serviceaccountsclient.ServerConfigurations{
					{
						URL: "",
					},
				},
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &rhSSOClient{
				config:        tt.fields.config,
				realmConfig:   tt.fields.realmConfig,
				configuration: tt.fields.configuration,
				cache:         tt.fields.cache,
			}
			g.Expect(c.getConfiguration(tt.args.accessToken)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_rhSSOClient_getCachedToken(t *testing.T) {
	type fields struct {
		config        *keycloak.KeycloakConfig
		realmConfig   *keycloak.KeycloakRealmConfig
		configuration *serviceaccountsclient.Configuration
		cache         *cache.Cache
	}
	type args struct {
		tokenKey string
	}
	server := mocks.NewMockServer()
	server.Start()
	defer server.Stop()
	accessToken := server.GenerateNewAuthToken()

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
		setupFn func(f *fields)
	}{
		{
			name: "should return the token key if it is cached",
			fields: fields{
				config:        &keycloak.KeycloakConfig{},
				realmConfig:   &keycloak.KeycloakRealmConfig{},
				configuration: &serviceaccountsclient.Configuration{},
				cache:         cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				tokenKey: accessToken,
			},
			want:    accessToken,
			wantErr: false,
			setupFn: func(f *fields) {
				f.cache.Set(accessToken, accessToken, cacheCleanupInterval)
			},
		},
		{
			name: "should return an empty string and a error if token key is not cached",
			fields: fields{
				config:        &keycloak.KeycloakConfig{},
				realmConfig:   &keycloak.KeycloakRealmConfig{},
				configuration: &serviceaccountsclient.Configuration{},
				cache:         cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				tokenKey: "uncached-token-key",
			},
			want:    "",
			wantErr: true,
			setupFn: func(f *fields) {
				f.cache.Set(accessToken, server.GenerateNewAuthToken(), cacheCleanupInterval)
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		if tt.setupFn != nil {
			tt.setupFn(&tt.fields)
		}
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &rhSSOClient{
				config:        tt.fields.config,
				realmConfig:   tt.fields.realmConfig,
				configuration: tt.fields.configuration,
				cache:         tt.fields.cache,
			}
			got, err := c.getCachedToken(tt.args.tokenKey)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_rhSSOClient_GetToken(t *testing.T) {
	type fields struct {
		config        *keycloak.KeycloakConfig
		realmConfig   *keycloak.KeycloakRealmConfig
		configuration *serviceaccountsclient.Configuration
		cache         *cache.Cache
	}
	server := mocks.NewMockServer()
	server.Start()
	defer server.Stop()
	accessToken := server.GenerateNewAuthToken()
	serviceAccount := CreateServiceAccountForTests(accessToken, server, accountName, accountDescription)
	server.SetBearerToken(*serviceAccount.Secret)

	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "should successfully return a clients token",
			fields: fields{
				config: &keycloak.KeycloakConfig{
					SsoBaseUrl: server.BaseURL(),
				},
				realmConfig: &keycloak.KeycloakRealmConfig{
					ClientID:         *serviceAccount.ClientId,
					ClientSecret:     *serviceAccount.Secret,
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
				configuration: &serviceaccountsclient.Configuration{
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			want:    *serviceAccount.Secret,
			wantErr: false,
		},
		{
			name: "should return an error when it fails to retrieve the token key",
			fields: fields{
				config: &keycloak.KeycloakConfig{
					SsoBaseUrl: server.BaseURL(),
				},
				realmConfig: &keycloak.KeycloakRealmConfig{
					ClientID:         "",
					ClientSecret:     "",
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
				configuration: &serviceaccountsclient.Configuration{
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &rhSSOClient{
				config:        tt.fields.config,
				realmConfig:   tt.fields.realmConfig,
				configuration: tt.fields.configuration,
				cache:         tt.fields.cache,
			}
			got, err := c.GetToken()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(got).To(gomega.Equal(tt.want))
		})

	}
}

func Test_rhSSOClient_GetConfig(t *testing.T) {
	type fields struct {
		config *keycloak.KeycloakConfig
	}
	tests := []struct {
		name   string
		fields fields
		want   *keycloak.KeycloakConfig
	}{
		{
			name: "should return the clients keycloak config",
			fields: fields{
				config: &keycloak.KeycloakConfig{},
			},
			want: &keycloak.KeycloakConfig{},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &rhSSOClient{
				config: tt.fields.config,
			}
			g.Expect(c.GetConfig()).To(gomega.Equal(tt.want))
		})
	}
}

func Test_rhSSOClient_GetRealmConfig(t *testing.T) {
	type fields struct {
		realmConfig *keycloak.KeycloakRealmConfig
	}
	tests := []struct {
		name   string
		fields fields
		want   *keycloak.KeycloakRealmConfig
	}{
		{
			name: "should return the clients keycloak Realm config",
			fields: fields{
				realmConfig: &keycloak.KeycloakRealmConfig{},
			},
			want: &keycloak.KeycloakRealmConfig{},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &rhSSOClient{
				realmConfig: tt.fields.realmConfig,
			}
			g.Expect(c.GetRealmConfig()).To(gomega.Equal(tt.want))
		})
	}
}

func Test_rhSSOClient_GetServiceAccounts(t *testing.T) {
	type fields struct {
		config        *keycloak.KeycloakConfig
		realmConfig   *keycloak.KeycloakRealmConfig
		configuration *serviceaccountsclient.Configuration
		cache         *cache.Cache
	}
	type args struct {
		accessToken string
		first       int
		max         int
	}
	server := mocks.NewMockServer()
	server.Start()
	defer server.Stop()
	accessToken := server.GenerateNewAuthToken()

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []serviceaccountsclient.ServiceAccountData
		wantErr bool
	}{
		{
			name: "should return a list of service accounts",
			fields: fields{
				config: &keycloak.KeycloakConfig{},
				realmConfig: &keycloak.KeycloakRealmConfig{
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL())},
				configuration: &serviceaccountsclient.Configuration{
					DefaultHeader: map[string]string{
						"Authorization": fmt.Sprintf("Bearer %s", accessToken),
						"Content-Type":  "application/json",
					},
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				accessToken: accessToken,
				first:       0,
				max:         5,
			},
			want:    []serviceaccountsclient.ServiceAccountData{},
			wantErr: false,
		},
		{
			name: "should return an error when server URL is Missing",
			fields: fields{
				config: &keycloak.KeycloakConfig{},
				realmConfig: &keycloak.KeycloakRealmConfig{
					BaseURL:          server.BaseURL(),
					ClientID:         "",
					ClientSecret:     "",
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL())},
				configuration: &serviceaccountsclient.Configuration{
					DefaultHeader: map[string]string{
						"Authorization": fmt.Sprintf("Bearer %s", accessToken),
						"Content-Type":  "application/json",
					},
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: "",
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				accessToken: accessToken,
				first:       0,
				max:         5,
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &rhSSOClient{
				config:        tt.fields.config,
				realmConfig:   tt.fields.realmConfig,
				configuration: tt.fields.configuration,
				cache:         tt.fields.cache,
			}
			got, err := c.GetServiceAccounts(tt.args.accessToken, tt.args.first, tt.args.max)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_rhSSOClient_GetServiceAccount(t *testing.T) {
	type fields struct {
		config        *keycloak.KeycloakConfig
		realmConfig   *keycloak.KeycloakRealmConfig
		configuration *serviceaccountsclient.Configuration
		cache         *cache.Cache
	}
	type args struct {
		accessToken string
		clientId    string
	}

	server := mocks.NewMockServer()
	server.Start()
	defer server.Stop()
	accessToken := server.GenerateNewAuthToken()
	serviceAccount := CreateServiceAccountForTests(accessToken, server, accountName, accountDescription)

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *serviceaccountsclient.ServiceAccountData
		found   bool
		wantErr bool
	}{
		{
			name: "should return the service account with matching clientId",
			fields: fields{
				config: &keycloak.KeycloakConfig{},
				realmConfig: &keycloak.KeycloakRealmConfig{
					ClientID:         *serviceAccount.ClientId,
					ClientSecret:     *serviceAccount.Secret,
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
				configuration: &serviceaccountsclient.Configuration{
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				accessToken: accessToken,
				clientId:    *serviceAccount.ClientId,
			},
			want:    &serviceAccount,
			found:   true,
			wantErr: false,
		},
		{
			name: "should fail if it cannot find the service account",
			fields: fields{
				config: &keycloak.KeycloakConfig{},
				realmConfig: &keycloak.KeycloakRealmConfig{
					ClientID:         *serviceAccount.ClientId,
					ClientSecret:     *serviceAccount.Secret,
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
				configuration: &serviceaccountsclient.Configuration{
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				accessToken: accessToken,
				clientId:    "wrong_clientId",
			},
			want:    nil,
			found:   false,
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &rhSSOClient{
				config:        tt.fields.config,
				realmConfig:   tt.fields.realmConfig,
				configuration: tt.fields.configuration,
				cache:         tt.fields.cache,
			}
			got, httpStatus, err := c.GetServiceAccount(tt.args.accessToken, tt.args.clientId)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(got).To(gomega.Equal(tt.want))
			g.Expect(httpStatus).To(gomega.Equal(tt.found))
		})
	}
}

func Test_rhSSOClient_CreateServiceAccount(t *testing.T) {
	type fields struct {
		config        *keycloak.KeycloakConfig
		realmConfig   *keycloak.KeycloakRealmConfig
		configuration *serviceaccountsclient.Configuration
		cache         *cache.Cache
	}
	type args struct {
		accessToken string
		name        string
		description string
	}
	server := mocks.NewMockServer()
	server.Start()
	defer server.Stop()
	accessToken := server.GenerateNewAuthToken()
	accountName := "serviceAccount"
	accountDescription := "fake service account"

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    serviceaccountsclient.ServiceAccountData
		wantErr bool
	}{
		{
			name: "should successfully create the service account",
			fields: fields{
				config: &keycloak.KeycloakConfig{},
				realmConfig: &keycloak.KeycloakRealmConfig{
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
				configuration: &serviceaccountsclient.Configuration{
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				accessToken: accessToken,
				name:        "serviceAccount",
				description: "fake service account",
			},
			want: serviceaccountsclient.ServiceAccountData{
				Name:        &accountName,
				Description: &accountDescription,
			},
			wantErr: false,
		},
		{
			name: "should fail to create the service account if wrong access token is given",
			fields: fields{
				config: &keycloak.KeycloakConfig{},
				realmConfig: &keycloak.KeycloakRealmConfig{
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
				configuration: &serviceaccountsclient.Configuration{
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				accessToken: "wrong_access_token",
				name:        "serviceAccount",
				description: "fake service account",
			},
			want: serviceaccountsclient.ServiceAccountData{
				Name:        nil,
				Description: nil,
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &rhSSOClient{
				config:        tt.fields.config,
				realmConfig:   tt.fields.realmConfig,
				configuration: tt.fields.configuration,
				cache:         tt.fields.cache,
			}
			got, err := c.CreateServiceAccount(tt.args.accessToken, tt.args.name, tt.args.description)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(got.Name).To(gomega.Equal(tt.want.Name))
			g.Expect(got.Description).To(gomega.Equal(tt.want.Description))
		})
	}
}

func Test_rhSSOClient_DeleteServiceAccount(t *testing.T) {
	type fields struct {
		config        *keycloak.KeycloakConfig
		realmConfig   *keycloak.KeycloakRealmConfig
		configuration *serviceaccountsclient.Configuration
		cache         *cache.Cache
	}
	type args struct {
		accessToken string
		clientId    string
	}

	server := mocks.NewMockServer()
	server.Start()
	defer server.Stop()
	accessToken := server.GenerateNewAuthToken()
	serviceAccount := CreateServiceAccountForTests(accessToken, server, accountName, accountDescription)

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "should successfully delete the service account",
			fields: fields{
				config: &keycloak.KeycloakConfig{},
				realmConfig: &keycloak.KeycloakRealmConfig{
					ClientID:         *serviceAccount.ClientId,
					ClientSecret:     *serviceAccount.Secret,
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
				configuration: &serviceaccountsclient.Configuration{
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				accessToken: accessToken,
				clientId:    *serviceAccount.ClientId,
			},
			wantErr: false,
		},
		{
			name: "should return an error if it fails to find service account for deletion",
			fields: fields{
				config: &keycloak.KeycloakConfig{},
				realmConfig: &keycloak.KeycloakRealmConfig{
					ClientID:         *serviceAccount.ClientId,
					ClientSecret:     *serviceAccount.Secret,
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
				configuration: &serviceaccountsclient.Configuration{
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				accessToken: accessToken,
				clientId:    "",
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &rhSSOClient{
				config:        tt.fields.config,
				realmConfig:   tt.fields.realmConfig,
				configuration: tt.fields.configuration,
				cache:         tt.fields.cache,
			}
			g.Expect(c.DeleteServiceAccount(tt.args.accessToken, tt.args.clientId) != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_rhSSOClient_UpdateServiceAccount(t *testing.T) {
	type fields struct {
		config        *keycloak.KeycloakConfig
		realmConfig   *keycloak.KeycloakRealmConfig
		configuration *serviceaccountsclient.Configuration
		cache         *cache.Cache
	}
	type args struct {
		accessToken string
		clientId    string
		name        string
		description string
	}

	server := mocks.NewMockServer()
	server.Start()
	defer server.Stop()
	accessToken := server.GenerateNewAuthToken()
	serviceAccount := CreateServiceAccountForTests(accessToken, server, accountName, accountDescription)

	name := "new name"
	description := "new description"

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    serviceaccountsclient.ServiceAccountData
		wantErr bool
	}{
		{
			name: "should successfully update the service account",
			fields: fields{
				config: &keycloak.KeycloakConfig{},
				realmConfig: &keycloak.KeycloakRealmConfig{
					ClientID:         *serviceAccount.ClientId,
					ClientSecret:     *serviceAccount.Secret,
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
				configuration: &serviceaccountsclient.Configuration{
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				accessToken: accessToken,
				clientId:    *serviceAccount.ClientId,
				name:        "new name",
				description: "new description",
			},
			want: serviceaccountsclient.ServiceAccountData{
				Id:          serviceAccount.Id,
				ClientId:    serviceAccount.ClientId,
				Secret:      serviceAccount.Secret,
				Name:        &name,
				Description: &description,
				CreatedBy:   nil,
				CreatedAt:   nil,
			},
			wantErr: false,
		},
		{
			name: "should return an error if it fails to find the service account to update",
			fields: fields{
				config: &keycloak.KeycloakConfig{},
				realmConfig: &keycloak.KeycloakRealmConfig{
					ClientID:         *serviceAccount.ClientId,
					ClientSecret:     *serviceAccount.Secret,
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
				configuration: &serviceaccountsclient.Configuration{
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				accessToken: accessToken,
				clientId:    "",
				name:        "new name",
				description: "new description",
			},
			want:    serviceaccountsclient.ServiceAccountData{},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &rhSSOClient{
				config:        tt.fields.config,
				realmConfig:   tt.fields.realmConfig,
				configuration: tt.fields.configuration,
				cache:         tt.fields.cache,
			}
			got, err := c.UpdateServiceAccount(tt.args.accessToken, tt.args.clientId, tt.args.name, tt.args.description)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_rhSSOClient_RegenerateClientSecret(t *testing.T) {
	type fields struct {
		config        *keycloak.KeycloakConfig
		realmConfig   *keycloak.KeycloakRealmConfig
		configuration *serviceaccountsclient.Configuration
		cache         *cache.Cache
	}
	type args struct {
		accessToken string
		id          string
	}

	server := mocks.NewMockServer()
	server.Start()
	defer server.Stop()
	accessToken := server.GenerateNewAuthToken()
	serviceAccount := CreateServiceAccountForTests(accessToken, server, accountName, accountDescription)

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    serviceaccountsclient.ServiceAccountData
		wantErr bool
	}{
		{
			name: "should successfully regenerate the clients secret",
			fields: fields{
				config: &keycloak.KeycloakConfig{},
				realmConfig: &keycloak.KeycloakRealmConfig{
					ClientID:         *serviceAccount.ClientId,
					ClientSecret:     *serviceAccount.Secret,
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
				configuration: &serviceaccountsclient.Configuration{
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				accessToken: accessToken,
				id:          *serviceAccount.ClientId,
			},
			want: serviceaccountsclient.ServiceAccountData{
				Secret: &accessToken,
			},
			wantErr: false,
		},
		{
			name: "should return an error if it fails to find the service account to regenerate client secret",
			fields: fields{
				config: &keycloak.KeycloakConfig{},
				realmConfig: &keycloak.KeycloakRealmConfig{
					ClientID:         *serviceAccount.ClientId,
					ClientSecret:     *serviceAccount.Secret,
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
				configuration: &serviceaccountsclient.Configuration{
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				accessToken: accessToken,
				id:          "",
			},
			want: serviceaccountsclient.ServiceAccountData{
				Secret: &accessToken,
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &rhSSOClient{
				config:        tt.fields.config,
				realmConfig:   tt.fields.realmConfig,
				configuration: tt.fields.configuration,
				cache:         tt.fields.cache,
			}
			got, err := c.RegenerateClientSecret(tt.args.accessToken, tt.args.id)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(got).To(gomega.Not(gomega.Equal(tt.want)))
		})
	}
}
