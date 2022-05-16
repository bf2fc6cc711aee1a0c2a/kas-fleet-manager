package redhatsso

import (
	"fmt"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	. "github.com/onsi/gomega"
	"github.com/patrickmn/go-cache"
	serviceaccountsclient "github.com/redhat-developer/app-services-sdk-go/serviceaccounts/apiv1internal/client"
)

const (
	accountName        = "serviceAccount"
	accountDescription = "fake service account"
)

func CreateServiceAccountForTests(accessToken string, server mocks.RedhatSSOMock, accountName string, accountDescription string) serviceaccountsclient.ServiceAccountData {
	c := &rhSSOClient{
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
	g := NewWithT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &rhSSOClient{
				config:        tt.fields.config,
				realmConfig:   tt.fields.realmConfig,
				configuration: tt.fields.configuration,
				cache:         tt.fields.cache,
			}
			g.Expect(c.getConfiguration(tt.args.accessToken)).To(Equal(tt.want))
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
	g := NewWithT(t)
	for _, tt := range tests {
		if tt.setupFn != nil {
			tt.setupFn(&tt.fields)
		}
		t.Run(tt.name, func(t *testing.T) {
			c := &rhSSOClient{
				config:        tt.fields.config,
				realmConfig:   tt.fields.realmConfig,
				configuration: tt.fields.configuration,
				cache:         tt.fields.cache,
			}
			got, err := c.getCachedToken(tt.args.tokenKey)
			g.Expect(err != nil).To(Equal(tt.wantErr))
			g.Expect(got).To(Equal(tt.want))
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
	g := NewWithT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &rhSSOClient{
				config: tt.fields.config,
			}
			g.Expect(c.GetConfig()).To(Equal(tt.want))
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
	g := NewWithT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &rhSSOClient{
				realmConfig: tt.fields.realmConfig,
			}
			g.Expect(c.GetRealmConfig()).To(Equal(tt.want))
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
				config:      &keycloak.KeycloakConfig{},
				realmConfig: &keycloak.KeycloakRealmConfig{},
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
	}
	g := NewWithT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &rhSSOClient{
				config:        tt.fields.config,
				realmConfig:   tt.fields.realmConfig,
				configuration: tt.fields.configuration,
				cache:         tt.fields.cache,
			}
			got, err := c.GetServiceAccounts(tt.args.accessToken, tt.args.first, tt.args.max)
			g.Expect(err != nil).To(Equal(tt.wantErr))
			g.Expect(got).To(Equal(tt.want))
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
				config:      &keycloak.KeycloakConfig{},
				realmConfig: &keycloak.KeycloakRealmConfig{},
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
				clientId:    *serviceAccount.ClientId,
			},
			want:    &serviceAccount,
			found:   true,
			wantErr: false,
		},
		{
			name: "should fail if it cannot find the service account",
			fields: fields{
				config:      &keycloak.KeycloakConfig{},
				realmConfig: &keycloak.KeycloakRealmConfig{},
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
				clientId:    "wrong_clientId",
			},
			want:    nil,
			found:   false,
			wantErr: false,
		},
	}
	g := NewWithT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &rhSSOClient{
				config:        tt.fields.config,
				realmConfig:   tt.fields.realmConfig,
				configuration: tt.fields.configuration,
				cache:         tt.fields.cache,
			}
			got, httpStatus, err := c.GetServiceAccount(tt.args.accessToken, tt.args.clientId)
			g.Expect(got).To(Equal(tt.want))
			g.Expect(httpStatus).To(Equal(tt.found))
			g.Expect(err != nil).To(Equal(tt.wantErr))
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
			name: "should sucessfully create the service account",
			fields: fields{
				config:      &keycloak.KeycloakConfig{},
				realmConfig: &keycloak.KeycloakRealmConfig{},
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
				name:        "serviceAccount",
				description: "fake service account",
			},
			want: serviceaccountsclient.ServiceAccountData{
				Name:        &accountName,
				Description: &accountDescription,
			},
			wantErr: false,
		},
	}
	g := NewWithT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &rhSSOClient{
				config:        tt.fields.config,
				realmConfig:   tt.fields.realmConfig,
				configuration: tt.fields.configuration,
				cache:         tt.fields.cache,
			}
			got, err := c.CreateServiceAccount(tt.args.accessToken, tt.args.name, tt.args.description)
			g.Expect(err != nil).To(Equal(tt.wantErr))
			g.Expect(got.Name).To(Equal(tt.want.Name))
			g.Expect(got.Description).To(Equal(tt.want.Description))
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
			name: "should sucessfully delete the service account",
			fields: fields{
				config:      &keycloak.KeycloakConfig{},
				realmConfig: &keycloak.KeycloakRealmConfig{},
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
				clientId:    *serviceAccount.ClientId,
			},
			wantErr: false,
		},
	}
	g := NewWithT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &rhSSOClient{
				config:        tt.fields.config,
				realmConfig:   tt.fields.realmConfig,
				configuration: tt.fields.configuration,
				cache:         tt.fields.cache,
			}
			g.Expect(c.DeleteServiceAccount(tt.args.accessToken, tt.args.clientId) != nil).To(Equal(tt.wantErr))
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
			name: "should sucessfully update the service account",
			fields: fields{
				config:      &keycloak.KeycloakConfig{},
				realmConfig: &keycloak.KeycloakRealmConfig{},
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
				OwnerId:     nil,
				CreatedAt:   nil,
			},
			wantErr: false,
		},
	}
	g := NewWithT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &rhSSOClient{
				config:        tt.fields.config,
				realmConfig:   tt.fields.realmConfig,
				configuration: tt.fields.configuration,
				cache:         tt.fields.cache,
			}
			got, err := c.UpdateServiceAccount(tt.args.accessToken, tt.args.clientId, tt.args.name, tt.args.description)
			g.Expect(got).To(Equal(tt.want))
			g.Expect(err != nil).To(Equal(tt.wantErr))
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
			name: "should sucessfully regenerate the clients secret",
			fields: fields{
				config:      &keycloak.KeycloakConfig{},
				realmConfig: &keycloak.KeycloakRealmConfig{},
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
				id:          *serviceAccount.ClientId,
			},
			want: serviceaccountsclient.ServiceAccountData{
				Id:          serviceAccount.Id,
				ClientId:    serviceAccount.ClientId,
				Secret:      &accessToken,
				Name:        serviceAccount.Name,
				Description: serviceAccount.Description,
				OwnerId:     nil,
				CreatedAt:   nil,
			},
			wantErr: false,
		},
	}
	g := NewWithT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &rhSSOClient{
				config:        tt.fields.config,
				realmConfig:   tt.fields.realmConfig,
				configuration: tt.fields.configuration,
				cache:         tt.fields.cache,
			}
			got, err := c.RegenerateClientSecret(tt.args.accessToken, tt.args.id)
			g.Expect(got.Secret).To(Not(Equal(tt.want.Secret)))
			g.Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}