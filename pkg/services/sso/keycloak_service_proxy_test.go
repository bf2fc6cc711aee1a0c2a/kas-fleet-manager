package sso

import (
	"context"
	"fmt"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/golang-jwt/jwt/v4"
	"github.com/onsi/gomega"
	"github.com/openshift-online/ocm-sdk-go/authentication"
)

var (
	testTokenProviderError = errors.NewWithCause(errors.ErrorGeneral, nil, "error getting access token\n caused by: failure retrieving token")
)

func testTokenProvider(providedToken string, called *bool, fail bool) tokenProvider {
	return func() (string, error) {
		*called = true
		if fail {
			return "", fmt.Errorf("failure retrieving token")
		}
		return providedToken, nil
	}
}

func Test_keycloakServiceProxy_DeRegisterClientInSSO(t *testing.T) {

	type args struct {
		clientID string
	}

	tests := []struct {
		name              string
		args              args
		wantErr           *errors.ServiceError
		tokenProviderFail bool
	}{
		{
			name: "Should succeed",
			args: args{
				clientID: testClientID,
			},
		},
		{
			name: "Should fail",
			args: args{
				clientID: testClientID,
			},
			tokenProviderFail: true,
			wantErr:           errors.NewWithCause(errors.ErrorGeneral, nil, "error getting access token\n caused by: failure retrieving token"),
		},
	}

	for _, tc := range tests {
		tt := tc

		getTokenCalled := false

		mock := &keycloakServiceInternalMock{
			DeRegisterClientInSSOFunc: func(accessToken string, kafkaNamespace string) *errors.ServiceError {
				return nil
			},
		}

		proxy := keycloakServiceProxy{
			getToken: testTokenProvider(token, &getTokenCalled, tt.tokenProviderFail),
			service:  mock,
		}

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			err := proxy.DeRegisterClientInSSO(tt.args.clientID)
			g.Expect(getTokenCalled).To(gomega.BeTrue())
			if tt.tokenProviderFail {
				g.Expect(err).To(gomega.HaveOccurred())
				g.Expect(mock.calls.DeRegisterClientInSSO).To(gomega.HaveLen(0))
				g.Expect(err.Error()).To(gomega.Equal(tt.wantErr.Error()))
				g.Expect(err.Code).To(gomega.Equal(tt.wantErr.Code))
			} else {
				g.Expect(err).ToNot(gomega.HaveOccurred())
				g.Expect(mock.calls.DeRegisterClientInSSO).To(gomega.HaveLen(1))
				g.Expect(mock.calls.DeRegisterClientInSSO[0].AccessToken).To(gomega.Equal(token))
				g.Expect(mock.calls.DeRegisterClientInSSO[0].KafkaNamespace).To(gomega.Equal(tt.args.clientID))
			}
		})
	}
}

func Test_keycloakServiceProxy_RegisterClientInSSO(t *testing.T) {
	testClusterCallbackURI := "testClusterCallbackURI"
	type args struct {
		clientID           string
		clusterCallbackURI string
	}

	tests := []struct {
		name              string
		args              args
		tokenProviderFail bool
		wantErr           *errors.ServiceError
	}{
		{
			name: "Should succeed",
			args: args{
				clientID:           testClientID,
				clusterCallbackURI: testClusterCallbackURI,
			},
		},
		{
			name: "Should fail",
			args: args{
				clientID:           testClientID,
				clusterCallbackURI: testClusterCallbackURI,
			},
			tokenProviderFail: true,
			wantErr:           testTokenProviderError,
		},
	}

	for _, tc := range tests {
		tt := tc

		mock := &keycloakServiceInternalMock{
			RegisterClientInSSOFunc: func(accessToken string, clusterId string, clusterOathCallbackURI string) (string, *errors.ServiceError) {
				return "", nil
			},
		}

		getTokenCalled := false

		proxy := keycloakServiceProxy{
			getToken: testTokenProvider(token, &getTokenCalled, tt.tokenProviderFail),
			service:  mock,
		}

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)

			_, err := proxy.RegisterClientInSSO(testClientID, testClusterCallbackURI)
			g.Expect(getTokenCalled).To(gomega.BeTrue())
			if tt.tokenProviderFail {
				g.Expect(err).To(gomega.HaveOccurred())
				g.Expect(mock.calls.RegisterClientInSSO).To(gomega.HaveLen(0))
				g.Expect(err.Error()).To(gomega.Equal(tt.wantErr.Error()))
				g.Expect(err.Code).To(gomega.Equal(tt.wantErr.Code))
			} else {
				g.Expect(err).ToNot(gomega.HaveOccurred())
				g.Expect(mock.calls.RegisterClientInSSO).To(gomega.HaveLen(1))
				g.Expect(mock.calls.RegisterClientInSSO[0].AccessToken).To(gomega.Equal(token))
				g.Expect(mock.calls.RegisterClientInSSO[0].ClusterId).To(gomega.Equal(testClientID))
				g.Expect(mock.calls.RegisterClientInSSO[0].ClusterOathCallbackURI).To(gomega.Equal(testClusterCallbackURI))
			}

		})
	}
}

func Test_keycloakServiceProxy_GetConfig(t *testing.T) {
	mock := &keycloakServiceInternalMock{
		GetConfigFunc: func() *keycloak.KeycloakConfig {
			return nil
		},
	}

	getTokenCalled := false

	proxy := keycloakServiceProxy{
		getToken: testTokenProvider(token, &getTokenCalled, false),
		service:  mock,
	}
	g := gomega.NewWithT(t)
	_ = proxy.GetConfig()
	g.Expect(mock.calls.GetConfig).To(gomega.HaveLen(1))
	g.Expect(getTokenCalled).To(gomega.BeFalse())
}

func Test_keycloakServiceProxy_GetRealmConfig(t *testing.T) {
	mock := &keycloakServiceInternalMock{
		GetRealmConfigFunc: func() *keycloak.KeycloakRealmConfig {
			return nil
		},
	}

	getTokenCalled := false

	proxy := keycloakServiceProxy{
		getToken: testTokenProvider(token, &getTokenCalled, false),
		service:  mock,
	}
	g := gomega.NewWithT(t)
	_ = proxy.GetRealmConfig()
	g.Expect(mock.calls.GetRealmConfig).To(gomega.HaveLen(1))
	g.Expect(getTokenCalled).To(gomega.BeFalse())
}

func Test_keycloakServiceProxy_IsKafkaClientExist(t *testing.T) {
	type args struct {
		clientID string
	}

	tests := []struct {
		name              string
		args              args
		tokenProviderFail bool
		wantErr           *errors.ServiceError
	}{
		{
			name: "Should succeed",
			args: args{
				clientID: testClientID,
			},
		},
		{
			name: "Should fail",
			args: args{
				clientID: testClientID,
			},
			tokenProviderFail: true,
			wantErr:           testTokenProviderError,
		},
	}

	for _, tc := range tests {
		tt := tc

		mock := &keycloakServiceInternalMock{
			IsKafkaClientExistFunc: func(accessToken string, clientId string) *errors.ServiceError {
				return nil
			},
		}

		getTokenCalled := false

		proxy := keycloakServiceProxy{
			getToken: testTokenProvider(token, &getTokenCalled, tt.tokenProviderFail),
			service:  mock,
		}

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)

			err := proxy.IsKafkaClientExist(tt.args.clientID)
			if tt.tokenProviderFail {
				g.Expect(err).To(gomega.HaveOccurred())
				g.Expect(mock.calls.IsKafkaClientExist).To(gomega.HaveLen(0))
				g.Expect(err.Error()).To(gomega.Equal(tt.wantErr.Error()))
				g.Expect(err.Code).To(gomega.Equal(tt.wantErr.Code))
			} else {
				g.Expect(err).ToNot(gomega.HaveOccurred())
				g.Expect(mock.calls.IsKafkaClientExist).To(gomega.HaveLen(1))
				g.Expect(mock.calls.IsKafkaClientExist[0].ClientId).To(gomega.Equal(testClientID))
				g.Expect(getTokenCalled).To(gomega.BeTrue())
			}
		})
	}
}

func Test_keycloakServiceProxy_CreateServiceAccount(t *testing.T) {
	jwtToken := jwt.Token{Raw: "Token123"}
	tests := []struct {
		name                     string
		ctx                      context.Context
		mock                     *keycloakServiceInternalMock
		expectGetTokenToBeCalled bool
		expectGetTokenToFail     bool
		wantErr                  *errors.ServiceError
	}{
		{
			name: "Test for RedhatSSO",
			ctx:  authentication.ContextWithToken(context.Background(), &jwtToken),
			mock: &keycloakServiceInternalMock{
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "redhat_sso"}
				},
				CreateServiceAccountFunc: func(accessToken string, serviceAccountRequest *api.ServiceAccountRequest, ctx context.Context) (*api.ServiceAccount, *errors.ServiceError) {
					return nil, nil
				},
			},
			expectGetTokenToBeCalled: false,
		},
		{
			name: "Test for MasSSO",
			ctx:  authentication.ContextWithToken(context.Background(), &jwtToken),
			mock: &keycloakServiceInternalMock{
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "mas_sso"}
				},
				CreateServiceAccountFunc: func(accessToken string, serviceAccountRequest *api.ServiceAccountRequest, ctx context.Context) (*api.ServiceAccount, *errors.ServiceError) {
					return nil, nil
				},
			},
			expectGetTokenToBeCalled: true,
		},
		{
			name: "Test for MasSSO - GetTokenError",
			ctx:  authentication.ContextWithToken(context.Background(), &jwtToken),
			mock: &keycloakServiceInternalMock{
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "mas_sso"}
				},
				CreateServiceAccountFunc: func(accessToken string, serviceAccountRequest *api.ServiceAccountRequest, ctx context.Context) (*api.ServiceAccount, *errors.ServiceError) {
					return nil, nil
				},
			},
			expectGetTokenToBeCalled: true,
			expectGetTokenToFail:     true,
			wantErr:                  testTokenProviderError,
		},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			getTokenCalled := false

			proxy := keycloakServiceProxy{
				getToken: testTokenProvider(token, &getTokenCalled, tt.expectGetTokenToFail),
				service:  tt.mock,
			}

			req := api.ServiceAccountRequest{
				Name:        "saTest",
				Description: "saTest Description",
			}
			_, err := proxy.CreateServiceAccount(&req, tt.ctx)
			g.Expect(getTokenCalled).To(gomega.Equal(tt.expectGetTokenToBeCalled))
			if tt.expectGetTokenToFail {
				g.Expect(err).To(gomega.HaveOccurred())
				g.Expect(tt.mock.calls.CreateServiceAccount).To(gomega.HaveLen(0))
				g.Expect(err.Error()).To(gomega.Equal(tt.wantErr.Error()))
				g.Expect(err.Code).To(gomega.Equal(tt.wantErr.Code))
			} else {
				g.Expect(err).ToNot(gomega.HaveOccurred())
				g.Expect(tt.mock.calls.CreateServiceAccount).To(gomega.HaveLen(1))
				g.Expect(tt.mock.calls.CreateServiceAccount[0].ServiceAccountRequest).To(gomega.Equal(&req))
				g.Expect(tt.mock.calls.CreateServiceAccount[0].Ctx).To(gomega.Equal(tt.ctx))
			}
		})
	}
}

func Test_keycloakServiceProxy_DeleteServiceAccount(t *testing.T) {
	jwtToken := jwt.Token{Raw: "Token123"}

	type args struct {
		ctx      context.Context
		clientID string
	}

	tests := []struct {
		name                     string
		mock                     *keycloakServiceInternalMock
		expectGetTokenToBeCalled bool
		expectGetTokenToFail     bool
		args                     args
		wantErr                  *errors.ServiceError
	}{
		{
			name: "Test for RedhatSSO",
			mock: &keycloakServiceInternalMock{
				DeleteServiceAccountFunc: func(accessToken string, ctx context.Context, clientId string) *errors.ServiceError {
					return nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "redhat_sso"}
				},
			},
			args: args{
				ctx:      authentication.ContextWithToken(context.Background(), &jwtToken),
				clientID: testClientID,
			},
			expectGetTokenToBeCalled: false,
		},
		{
			name: "Test for MasSSO",
			mock: &keycloakServiceInternalMock{
				DeleteServiceAccountFunc: func(accessToken string, ctx context.Context, clientId string) *errors.ServiceError {
					return nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "mas_sso"}
				},
			},
			args: args{
				ctx:      authentication.ContextWithToken(context.Background(), &jwtToken),
				clientID: testClientID,
			},
			expectGetTokenToBeCalled: true,
		},
		{
			name: "Test for MasSSO - GetToken Error",
			mock: &keycloakServiceInternalMock{
				DeleteServiceAccountFunc: func(accessToken string, ctx context.Context, clientId string) *errors.ServiceError {
					return nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "mas_sso"}
				},
			},
			args: args{
				ctx:      authentication.ContextWithToken(context.Background(), &jwtToken),
				clientID: testClientID,
			},
			expectGetTokenToBeCalled: true,
			expectGetTokenToFail:     true,
			wantErr:                  testTokenProviderError,
		},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			getTokenCalled := false

			proxy := keycloakServiceProxy{
				getToken: testTokenProvider(token, &getTokenCalled, tt.expectGetTokenToFail),
				service:  tt.mock,
			}
			err := proxy.DeleteServiceAccount(tt.args.ctx, tt.args.clientID)
			if tt.expectGetTokenToFail {
				g.Expect(err).To(gomega.HaveOccurred())
				g.Expect(tt.mock.calls.DeleteServiceAccount).To(gomega.HaveLen(0))
				g.Expect(err.Error()).To(gomega.Equal(tt.wantErr.Error()))
				g.Expect(err.Code).To(gomega.Equal(tt.wantErr.Code))
			} else {
				g.Expect(err).ToNot(gomega.HaveOccurred())
				g.Expect(tt.mock.calls.DeleteServiceAccount).To(gomega.HaveLen(1))
				g.Expect(tt.mock.calls.DeleteServiceAccount[0].ClientId).To(gomega.Equal(tt.args.clientID))
				g.Expect(tt.mock.calls.DeleteServiceAccount[0].Ctx).To(gomega.Equal(tt.args.ctx))
				g.Expect(getTokenCalled).To(gomega.Equal(tt.expectGetTokenToBeCalled))
			}
		})
	}
}

func Test_keycloakServiceProxy_ResetServiceAccountCredentials(t *testing.T) {
	jwtToken := jwt.Token{Raw: "Token123"}

	type args struct {
		ctx      context.Context
		clientID string
	}

	tests := []struct {
		name                     string
		mock                     *keycloakServiceInternalMock
		expectGetTokenToBeCalled bool
		expectGetTokenToFail     bool
		args                     args
		wantErr                  *errors.ServiceError
	}{
		{
			name: "Test for RedhatSSO",
			mock: &keycloakServiceInternalMock{
				ResetServiceAccountCredentialsFunc: func(accessToken string, ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError) {
					return nil, nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "redhat_sso"}
				},
			},
			args: args{
				ctx:      authentication.ContextWithToken(context.Background(), &jwtToken),
				clientID: testClientID,
			},
			expectGetTokenToBeCalled: false,
		},
		{
			name: "Test for MasSSO",
			mock: &keycloakServiceInternalMock{
				ResetServiceAccountCredentialsFunc: func(accessToken string, ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError) {
					return nil, nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "mas_sso"}
				},
			},
			args: args{
				ctx:      authentication.ContextWithToken(context.Background(), &jwtToken),
				clientID: testClientID,
			},
			expectGetTokenToBeCalled: true,
		},
		{
			name: "Test for MasSSO - Get Provider Error",
			mock: &keycloakServiceInternalMock{
				ResetServiceAccountCredentialsFunc: func(accessToken string, ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError) {
					return nil, nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "mas_sso"}
				},
			},
			args: args{
				ctx:      authentication.ContextWithToken(context.Background(), &jwtToken),
				clientID: testClientID,
			},
			expectGetTokenToFail:     true,
			expectGetTokenToBeCalled: true,
			wantErr:                  testTokenProviderError,
		},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			getTokenCalled := false

			proxy := keycloakServiceProxy{
				getToken: testTokenProvider(token, &getTokenCalled, tt.expectGetTokenToFail),
				service:  tt.mock,
			}
			_, err := proxy.ResetServiceAccountCredentials(tt.args.ctx, tt.args.clientID)
			if tt.expectGetTokenToFail {
				g.Expect(err).To(gomega.HaveOccurred())
				g.Expect(tt.mock.calls.ResetServiceAccountCredentials).To(gomega.HaveLen(0))
				g.Expect(err.Error()).To(gomega.Equal(tt.wantErr.Error()))
				g.Expect(err.Code).To(gomega.Equal(tt.wantErr.Code))
			} else {
				g.Expect(err).ToNot(gomega.HaveOccurred())
				g.Expect(tt.mock.calls.ResetServiceAccountCredentials).To(gomega.HaveLen(1))
				g.Expect(tt.mock.calls.ResetServiceAccountCredentials[0].ClientId).To(gomega.Equal(tt.args.clientID))
				g.Expect(tt.mock.calls.ResetServiceAccountCredentials[0].Ctx).To(gomega.Equal(tt.args.ctx))
			}
			g.Expect(getTokenCalled).To(gomega.Equal(tt.expectGetTokenToBeCalled))
		})
	}
}

func Test_keycloakServiceProxy_ListServiceAcc(t *testing.T) {
	jwtToken := jwt.Token{Raw: "Token123"}

	type args struct {
		ctx   context.Context
		first int
		max   int
	}

	tests := []struct {
		name                     string
		mock                     *keycloakServiceInternalMock
		expectGetTokenToBeCalled bool
		expectGetTokenToBeFail   bool
		args                     args
		wantErr                  *errors.ServiceError
	}{
		{
			name: "Test for RedhatSSO",
			mock: &keycloakServiceInternalMock{
				ListServiceAccFunc: func(accessToken string, ctx context.Context, first int, max int) ([]api.ServiceAccount, *errors.ServiceError) {
					return nil, nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "redhat_sso"}
				},
			},
			args: args{
				ctx:   authentication.ContextWithToken(context.Background(), &jwtToken),
				first: 5,
				max:   50,
			},
			expectGetTokenToBeCalled: false,
		},
		{
			name: "Test for MasSSO",
			mock: &keycloakServiceInternalMock{
				ListServiceAccFunc: func(accessToken string, ctx context.Context, first int, max int) ([]api.ServiceAccount, *errors.ServiceError) {
					return nil, nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "mas_sso"}
				},
			},
			args: args{
				ctx:   authentication.ContextWithToken(context.Background(), &jwtToken),
				first: 3,
				max:   300,
			},
			expectGetTokenToBeCalled: true,
		},
		{
			name: "Test for MasSSO - Should fail with GetToken Error",
			mock: &keycloakServiceInternalMock{
				ListServiceAccFunc: func(accessToken string, ctx context.Context, first int, max int) ([]api.ServiceAccount, *errors.ServiceError) {
					return nil, nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "mas_sso"}
				},
			},
			args: args{
				ctx:   authentication.ContextWithToken(context.Background(), &jwtToken),
				first: 3,
				max:   300,
			},
			expectGetTokenToBeCalled: true,
			expectGetTokenToBeFail:   true,
			wantErr:                  testTokenProviderError,
		},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)

			getTokenCalled := false

			proxy := keycloakServiceProxy{
				getToken: testTokenProvider(token, &getTokenCalled, tt.expectGetTokenToBeFail),
				service:  tt.mock,
			}

			_, err := proxy.ListServiceAcc(tt.args.ctx, tt.args.first, tt.args.max)
			if tt.expectGetTokenToBeFail {
				g.Expect(err).To(gomega.HaveOccurred())
				g.Expect(err.Error()).To(gomega.Equal(tt.wantErr.Error()))
				g.Expect(err.Code).To(gomega.Equal(tt.wantErr.Code))
				g.Expect(tt.mock.calls.ListServiceAcc).To(gomega.HaveLen(0))
			} else {
				g.Expect(err).ToNot(gomega.HaveOccurred())
				g.Expect(tt.mock.calls.ListServiceAcc).To(gomega.HaveLen(1))
				g.Expect(tt.mock.calls.ListServiceAcc[0].First).To(gomega.Equal(tt.args.first))
				g.Expect(tt.mock.calls.ListServiceAcc[0].Max).To(gomega.Equal(tt.args.max))
				g.Expect(tt.mock.calls.ListServiceAcc[0].Ctx).To(gomega.Equal(tt.args.ctx))
			}
			g.Expect(getTokenCalled).To(gomega.Equal(tt.expectGetTokenToBeCalled))
		})
	}
}

func Test_keycloakServiceProxy_RegisterKasFleetshardOperatorServiceAccount(t *testing.T) {
	type args struct {
		clusterID string
	}

	tests := []struct {
		name                     string
		mock                     *keycloakServiceInternalMock
		expectGetTokenToBeCalled bool
		expectGetTokenToFail     bool
		args                     args
		wantErr                  *errors.ServiceError
	}{
		{
			name: "Test for RedhatSSO",
			mock: &keycloakServiceInternalMock{
				RegisterKasFleetshardOperatorServiceAccountFunc: func(accessToken string, agentClusterId string) (*api.ServiceAccount, *errors.ServiceError) {
					return nil, nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "redhat_sso"}
				},
			},
			args: args{
				clusterID: testClientID,
			},
			expectGetTokenToBeCalled: true,
		},
		{
			name: "Test for MasSSO",
			mock: &keycloakServiceInternalMock{
				RegisterKasFleetshardOperatorServiceAccountFunc: func(accessToken string, agentClusterId string) (*api.ServiceAccount, *errors.ServiceError) {
					return nil, nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "mas_sso"}
				},
			},
			args: args{
				clusterID: testClientID,
			},
			expectGetTokenToBeCalled: true,
		},
		{
			name: "Test for MasSSO - Should fail with Token Error",
			mock: &keycloakServiceInternalMock{
				RegisterKasFleetshardOperatorServiceAccountFunc: func(accessToken string, agentClusterId string) (*api.ServiceAccount, *errors.ServiceError) {
					return nil, nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "mas_sso"}
				},
			},
			args: args{
				clusterID: testClientID,
			},
			expectGetTokenToBeCalled: true,
			expectGetTokenToFail:     true,
			wantErr:                  testTokenProviderError,
		},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			getTokenCalled := false

			proxy := keycloakServiceProxy{
				getToken: testTokenProvider(token, &getTokenCalled, tt.expectGetTokenToFail),
				service:  tt.mock,
			}

			_, err := proxy.RegisterKasFleetshardOperatorServiceAccount(tt.args.clusterID)
			if tt.expectGetTokenToFail {
				g.Expect(err).To(gomega.HaveOccurred())
				g.Expect(tt.mock.calls.RegisterKasFleetshardOperatorServiceAccount).To(gomega.HaveLen(0))
				g.Expect(err.Error()).To(gomega.Equal(tt.wantErr.Error()))
				g.Expect(err.Code).To(gomega.Equal(tt.wantErr.Code))
			} else {
				g.Expect(err).NotTo(gomega.HaveOccurred())
				g.Expect(tt.mock.calls.RegisterKasFleetshardOperatorServiceAccount).To(gomega.HaveLen(1))
				g.Expect(tt.mock.calls.RegisterKasFleetshardOperatorServiceAccount[0].AgentClusterId).To(gomega.Equal(tt.args.clusterID))
				g.Expect(getTokenCalled).To(gomega.Equal(tt.expectGetTokenToBeCalled))
			}
		})
	}
}

func Test_keycloakServiceProxy_DeRegisterKasFleetshardOperatorServiceAccount(t *testing.T) {
	type args struct {
		clusterID string
	}

	tests := []struct {
		name                     string
		mock                     *keycloakServiceInternalMock
		expectGetTokenToBeCalled bool
		expectGetTokenToFail     bool
		args                     args
		wantErr                  *errors.ServiceError
	}{
		{
			name: "Test for RedhatSSO",
			mock: &keycloakServiceInternalMock{
				DeRegisterKasFleetshardOperatorServiceAccountFunc: func(accessToken string, agentClusterId string) *errors.ServiceError {
					return nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "redhat_sso"}
				},
			},
			args: args{
				clusterID: testClientID,
			},
			expectGetTokenToBeCalled: true,
		},
		{
			name: "Test for MasSSO",
			mock: &keycloakServiceInternalMock{
				DeRegisterKasFleetshardOperatorServiceAccountFunc: func(accessToken string, agentClusterId string) *errors.ServiceError {
					return nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "mas_sso"}
				},
			},
			args: args{
				clusterID: testClientID,
			},
			expectGetTokenToBeCalled: true,
		},
		{
			name: "Test for MasSSO - Should fail with GetToken Error",
			mock: &keycloakServiceInternalMock{
				DeRegisterKasFleetshardOperatorServiceAccountFunc: func(accessToken string, agentClusterId string) *errors.ServiceError {
					return nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "mas_sso"}
				},
			},
			args: args{
				clusterID: testClientID,
			},
			expectGetTokenToBeCalled: true,
			expectGetTokenToFail:     true,
			wantErr:                  testTokenProviderError,
		},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			getTokenCalled := false

			proxy := keycloakServiceProxy{
				getToken: testTokenProvider(token, &getTokenCalled, tt.expectGetTokenToFail),
				service:  tt.mock,
			}

			err := proxy.DeRegisterKasFleetshardOperatorServiceAccount(tt.args.clusterID)
			if tt.expectGetTokenToFail {
				g.Expect(err).To(gomega.HaveOccurred())
				g.Expect(tt.mock.calls.DeRegisterKasFleetshardOperatorServiceAccount).To(gomega.HaveLen(0))
				g.Expect(err.Error()).To(gomega.Equal(tt.wantErr.Error()))
				g.Expect(err.Code).To(gomega.Equal(tt.wantErr.Code))
			} else {
				g.Expect(err).ToNot(gomega.HaveOccurred())
				g.Expect(tt.mock.calls.DeRegisterKasFleetshardOperatorServiceAccount).To(gomega.HaveLen(1))
				g.Expect(tt.mock.calls.DeRegisterKasFleetshardOperatorServiceAccount[0].AgentClusterId).To(gomega.Equal(tt.args.clusterID))
			}
			g.Expect(getTokenCalled).To(gomega.Equal(tt.expectGetTokenToBeCalled))
		})
	}
}

func Test_keycloakServiceProxy_GetServiceAccountById(t *testing.T) {
	jwtToken := jwt.Token{Raw: "Token123"}

	type args struct {
		ctx      context.Context
		clientID string
	}

	tests := []struct {
		name                     string
		mock                     *keycloakServiceInternalMock
		expectGetTokenToBeCalled bool
		expectGetTokenToFail     bool
		args                     args
		wantErr                  *errors.ServiceError
	}{
		{
			name: "Test for RedhatSSO",
			mock: &keycloakServiceInternalMock{
				GetServiceAccountByIdFunc: func(accessToken string, ctx context.Context, id string) (*api.ServiceAccount, *errors.ServiceError) {
					return nil, nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "redhat_sso"}
				},
			},
			args: args{
				ctx:      authentication.ContextWithToken(context.Background(), &jwtToken),
				clientID: testClientID,
			},
			expectGetTokenToBeCalled: false,
		},
		{
			name: "Test for MasSSO",
			mock: &keycloakServiceInternalMock{
				GetServiceAccountByIdFunc: func(accessToken string, ctx context.Context, id string) (*api.ServiceAccount, *errors.ServiceError) {
					return nil, nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "mas_sso"}
				},
			},
			args: args{
				ctx:      authentication.ContextWithToken(context.Background(), &jwtToken),
				clientID: testClientID,
			},
			expectGetTokenToBeCalled: true,
		},
		{
			name: "Test for MasSSO - Should fail with GetToken Error",
			mock: &keycloakServiceInternalMock{
				GetServiceAccountByIdFunc: func(accessToken string, ctx context.Context, id string) (*api.ServiceAccount, *errors.ServiceError) {
					return nil, nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "mas_sso"}
				},
			},
			args: args{
				ctx:      authentication.ContextWithToken(context.Background(), &jwtToken),
				clientID: testClientID,
			},
			expectGetTokenToBeCalled: true,
			expectGetTokenToFail:     true,
			wantErr:                  testTokenProviderError,
		},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			getTokenCalled := false

			proxy := keycloakServiceProxy{
				getToken: testTokenProvider(token, &getTokenCalled, tt.expectGetTokenToFail),
				service:  tt.mock,
			}

			_, err := proxy.GetServiceAccountById(tt.args.ctx, tt.args.clientID)
			if tt.expectGetTokenToFail {
				g.Expect(err).To(gomega.HaveOccurred())
				g.Expect(tt.mock.calls.GetServiceAccountById).To(gomega.HaveLen(0))
				g.Expect(err.Error()).To(gomega.Equal(tt.wantErr.Error()))
				g.Expect(err.Code).To(gomega.Equal(tt.wantErr.Code))
			} else {
				g.Expect(err).NotTo(gomega.HaveOccurred())
				g.Expect(tt.mock.calls.GetServiceAccountById).To(gomega.HaveLen(1))
				g.Expect(tt.mock.calls.GetServiceAccountById[0].ID).To(gomega.Equal(tt.args.clientID))
				g.Expect(tt.mock.calls.GetServiceAccountById[0].Ctx).To(gomega.Equal(tt.args.ctx))
			}
			g.Expect(getTokenCalled).To(gomega.Equal(tt.expectGetTokenToBeCalled))
		})
	}
}

func Test_keycloakServiceProxy_GetServiceAccountByClientId(t *testing.T) {
	jwtToken := jwt.Token{Raw: "Token123"}

	type args struct {
		ctx      context.Context
		clientID string
	}

	tests := []struct {
		name                     string
		mock                     *keycloakServiceInternalMock
		expectGetTokenToBeCalled bool
		expectGetTokenToFail     bool
		args                     args
		wantErr                  *errors.ServiceError
	}{
		{
			name: "Test for RedhatSSO",
			mock: &keycloakServiceInternalMock{
				GetServiceAccountByClientIdFunc: func(accessToken string, ctx context.Context, id string) (*api.ServiceAccount, *errors.ServiceError) {
					return nil, nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "redhat_sso"}
				},
			},
			args: args{
				ctx:      authentication.ContextWithToken(context.Background(), &jwtToken),
				clientID: testClientID,
			},
			expectGetTokenToBeCalled: false,
		},
		{
			name: "Test for MasSSO",
			mock: &keycloakServiceInternalMock{
				GetServiceAccountByClientIdFunc: func(accessToken string, ctx context.Context, id string) (*api.ServiceAccount, *errors.ServiceError) {
					return nil, nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "mas_sso"}
				},
			},
			args: args{
				ctx:      authentication.ContextWithToken(context.Background(), &jwtToken),
				clientID: testClientID,
			},
			expectGetTokenToBeCalled: true,
		},
		{
			name: "Test for MasSSO - Fail with GetToken Error",
			mock: &keycloakServiceInternalMock{
				GetServiceAccountByClientIdFunc: func(accessToken string, ctx context.Context, id string) (*api.ServiceAccount, *errors.ServiceError) {
					return nil, nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "mas_sso"}
				},
			},
			args: args{
				ctx:      authentication.ContextWithToken(context.Background(), &jwtToken),
				clientID: testClientID,
			},
			expectGetTokenToBeCalled: true,
			expectGetTokenToFail:     true,
			wantErr:                  testTokenProviderError,
		},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			getTokenCalled := false

			proxy := keycloakServiceProxy{
				getToken: testTokenProvider(token, &getTokenCalled, tt.expectGetTokenToFail),
				service:  tt.mock,
			}

			_, err := proxy.GetServiceAccountByClientId(tt.args.ctx, tt.args.clientID)
			if tt.expectGetTokenToFail {
				g.Expect(err).To(gomega.HaveOccurred())
				g.Expect(tt.mock.calls.GetServiceAccountByClientId).To(gomega.HaveLen(0))
				g.Expect(err.Error()).To(gomega.Equal(tt.wantErr.Error()))
				g.Expect(err.Code).To(gomega.Equal(tt.wantErr.Code))
			} else {
				g.Expect(err).ToNot(gomega.HaveOccurred())
				g.Expect(tt.mock.calls.GetServiceAccountByClientId).To(gomega.HaveLen(1))
				g.Expect(tt.mock.calls.GetServiceAccountByClientId[0].ClientId).To(gomega.Equal(tt.args.clientID))
				g.Expect(tt.mock.calls.GetServiceAccountByClientId[0].Ctx).To(gomega.Equal(tt.args.ctx))
			}
			g.Expect(getTokenCalled).To(gomega.Equal(tt.expectGetTokenToBeCalled))
		})
	}
}

func Test_keycloakServiceProxy_RegisterConnectorFleetshardOperatorServiceAccount(t *testing.T) {
	type args struct {
		clusterID string
	}

	tests := []struct {
		name                     string
		mock                     *keycloakServiceInternalMock
		expectGetTokenToBeCalled bool
		expectGetTokenToFail     bool
		args                     args
		wantErr                  *errors.ServiceError
	}{
		{
			name: "Test for RedhatSSO",
			mock: &keycloakServiceInternalMock{
				RegisterConnectorFleetshardOperatorServiceAccountFunc: func(accessToken string, agentClusterId string) (*api.ServiceAccount, *errors.ServiceError) {
					return nil, nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "redhat_sso"}
				},
			},
			args: args{
				clusterID: testClientID,
			},
			expectGetTokenToBeCalled: true,
		},
		{
			name: "Test for MasSSO",
			mock: &keycloakServiceInternalMock{
				RegisterConnectorFleetshardOperatorServiceAccountFunc: func(accessToken string, agentClusterId string) (*api.ServiceAccount, *errors.ServiceError) {
					return nil, nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "mas_sso"}
				},
			},
			args: args{
				clusterID: testClientID,
			},
			expectGetTokenToBeCalled: true,
		},
		{
			name: "Test for MasSSO - Should fail with GetToken Error",
			mock: &keycloakServiceInternalMock{
				RegisterConnectorFleetshardOperatorServiceAccountFunc: func(accessToken string, agentClusterId string) (*api.ServiceAccount, *errors.ServiceError) {
					return nil, nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "mas_sso"}
				},
			},
			args: args{
				clusterID: testClientID,
			},
			expectGetTokenToBeCalled: true,
			expectGetTokenToFail:     true,
			wantErr:                  testTokenProviderError,
		},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			getTokenCalled := false

			proxy := keycloakServiceProxy{
				getToken: testTokenProvider(token, &getTokenCalled, tt.expectGetTokenToFail),
				service:  tt.mock,
			}

			_, err := proxy.RegisterConnectorFleetshardOperatorServiceAccount(tt.args.clusterID)
			if tt.expectGetTokenToFail {
				g.Expect(err).To(gomega.HaveOccurred())
				g.Expect(tt.mock.calls.RegisterConnectorFleetshardOperatorServiceAccount).To(gomega.HaveLen(0))
				g.Expect(err.Error()).To(gomega.Equal(tt.wantErr.Error()))
				g.Expect(err.Code).To(gomega.Equal(tt.wantErr.Code))
			} else {
				g.Expect(err).ToNot(gomega.HaveOccurred())
				g.Expect(tt.mock.calls.RegisterConnectorFleetshardOperatorServiceAccount).To(gomega.HaveLen(1))
				g.Expect(tt.mock.calls.RegisterConnectorFleetshardOperatorServiceAccount[0].AgentClusterId).To(gomega.Equal(tt.args.clusterID))
			}
			g.Expect(getTokenCalled).To(gomega.Equal(tt.expectGetTokenToBeCalled))
		})
	}
}

func Test_keycloakServiceProxy_DeRegisterConnectorFleetshardOperatorServiceAccount(t *testing.T) {
	type args struct {
		clusterID string
	}

	tests := []struct {
		name                     string
		mock                     *keycloakServiceInternalMock
		expectGetTokenToBeCalled bool
		expectGetTokenToFail     bool
		args                     args
		wantErr                  *errors.ServiceError
	}{
		{
			name: "Test for RedhatSSO",
			mock: &keycloakServiceInternalMock{
				DeRegisterConnectorFleetshardOperatorServiceAccountFunc: func(accessToken string, agentClusterId string) *errors.ServiceError {
					return nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "redhat_sso"}
				},
			},
			args: args{
				clusterID: testClientID,
			},
			expectGetTokenToBeCalled: true,
		},
		{
			name: "Test for MasSSO",
			mock: &keycloakServiceInternalMock{
				DeRegisterConnectorFleetshardOperatorServiceAccountFunc: func(accessToken string, agentClusterId string) *errors.ServiceError {
					return nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "mas_sso"}
				},
			},
			args: args{
				clusterID: testClientID,
			},
			expectGetTokenToBeCalled: true,
		},
		{
			name: "Test for MasSSO - Should fail with GetToken Error",
			mock: &keycloakServiceInternalMock{
				DeRegisterConnectorFleetshardOperatorServiceAccountFunc: func(accessToken string, agentClusterId string) *errors.ServiceError {
					return nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "mas_sso"}
				},
			},
			args: args{
				clusterID: testClientID,
			},
			expectGetTokenToBeCalled: true,
			expectGetTokenToFail:     true,
			wantErr:                  testTokenProviderError,
		},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			getTokenCalled := false

			proxy := keycloakServiceProxy{
				getToken: testTokenProvider(token, &getTokenCalled, tt.expectGetTokenToFail),
				service:  tt.mock,
			}

			err := proxy.DeRegisterConnectorFleetshardOperatorServiceAccount(tt.args.clusterID)
			if tt.expectGetTokenToFail {
				g.Expect(err).To(gomega.HaveOccurred())
				g.Expect(tt.mock.calls.DeRegisterConnectorFleetshardOperatorServiceAccount).To(gomega.HaveLen(0))
				g.Expect(err.Error()).To(gomega.Equal(tt.wantErr.Error()))
				g.Expect(err.Code).To(gomega.Equal(tt.wantErr.Code))
			} else {
				g.Expect(err).ToNot(gomega.HaveOccurred())
				g.Expect(tt.mock.calls.DeRegisterConnectorFleetshardOperatorServiceAccount).To(gomega.HaveLen(1))
				g.Expect(tt.mock.calls.DeRegisterConnectorFleetshardOperatorServiceAccount[0].AgentClusterId).To(gomega.Equal(tt.args.clusterID))
			}
			g.Expect(getTokenCalled).To(gomega.Equal(tt.expectGetTokenToBeCalled))
		})
	}
}

func Test_keycloakServiceProxy_GetKafkaClientSecret(t *testing.T) {
	type args struct {
		clusterID         string
		tokenProviderFail bool
	}

	tests := []struct {
		name                     string
		mock                     *keycloakServiceInternalMock
		expectGetTokenToBeCalled bool
		args                     args
		wantErr                  *errors.ServiceError
	}{
		{
			name: "Test for RedhatSSO",
			mock: &keycloakServiceInternalMock{
				GetKafkaClientSecretFunc: func(accessToken string, clientId string) (string, *errors.ServiceError) {
					return "", nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "redhat_sso"}
				},
			},
			args: args{
				clusterID: testClientID,
			},
			expectGetTokenToBeCalled: true,
		},
		{
			name: "Test for MasSSO",
			mock: &keycloakServiceInternalMock{
				DeRegisterConnectorFleetshardOperatorServiceAccountFunc: func(accessToken string, agentClusterId string) *errors.ServiceError {
					return nil
				},
				GetKafkaClientSecretFunc: func(accessToken string, clientId string) (string, *errors.ServiceError) {
					return "", nil
				},
			},
			args: args{
				clusterID: testClientID,
			},
			expectGetTokenToBeCalled: true,
		},
		{
			name: "Should fail retrieving token",
			mock: &keycloakServiceInternalMock{
				DeRegisterConnectorFleetshardOperatorServiceAccountFunc: func(accessToken string, agentClusterId string) *errors.ServiceError {
					return nil
				},
				GetKafkaClientSecretFunc: func(accessToken string, clientId string) (string, *errors.ServiceError) {
					return "", nil
				},
			},
			args: args{
				clusterID:         testClientID,
				tokenProviderFail: true,
			},
			expectGetTokenToBeCalled: true,
			wantErr:                  errors.NewWithCause(errors.ErrorGeneral, nil, "error getting access token\n caused by: failure retrieving token"),
		},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			getTokenCalled := false

			proxy := keycloakServiceProxy{
				getToken: testTokenProvider(token, &getTokenCalled, tt.args.tokenProviderFail),
				service:  tt.mock,
			}

			_, err := proxy.GetKafkaClientSecret(tt.args.clusterID)
			if tt.args.tokenProviderFail {
				g.Expect(err).To(gomega.HaveOccurred())
				g.Expect(tt.mock.calls.GetKafkaClientSecret).To(gomega.HaveLen(0))
				g.Expect(err.Error()).To(gomega.Equal(tt.wantErr.Error()))
				g.Expect(err.Code).To(gomega.Equal(tt.wantErr.Code))
			} else {
				g.Expect(err).ToNot(gomega.HaveOccurred())
				g.Expect(tt.mock.calls.GetKafkaClientSecret).To(gomega.HaveLen(1))
				g.Expect(tt.mock.calls.GetKafkaClientSecret[0].ClientId).To(gomega.Equal(tt.args.clusterID))
			}

			g.Expect(getTokenCalled).To(gomega.Equal(tt.expectGetTokenToBeCalled))
		})
	}
}

func Test_keycloakServiceProxy_CreateServiceAccountInternal(t *testing.T) {
	type args struct {
		request           CompleteServiceAccountRequest
		tokenProviderFail bool
	}

	tests := []struct {
		name                     string
		mock                     *keycloakServiceInternalMock
		expectGetTokenToBeCalled bool
		args                     args
		wantErr                  *errors.ServiceError
	}{
		{
			name: "Test for RedhatSSO",
			mock: &keycloakServiceInternalMock{
				CreateServiceAccountInternalFunc: func(accessToken string, request CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError) {
					return nil, nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "redhat_sso"}
				},
			},
			args: args{
				request: CompleteServiceAccountRequest{Owner: "test"},
			},
			expectGetTokenToBeCalled: true,
		},
		{
			name: "Test for MasSSO",
			mock: &keycloakServiceInternalMock{
				CreateServiceAccountInternalFunc: func(accessToken string, request CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError) {
					return nil, nil
				},
				GetKafkaClientSecretFunc: func(accessToken string, clientId string) (string, *errors.ServiceError) {
					return "", nil
				},
			},
			args: args{
				request: CompleteServiceAccountRequest{Owner: "test"},
			},
			expectGetTokenToBeCalled: true,
		},
		{
			name: "Should fail retrieving token",
			mock: &keycloakServiceInternalMock{
				CreateServiceAccountInternalFunc: func(accessToken string, request CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError) {
					return nil, nil
				},
				GetKafkaClientSecretFunc: func(accessToken string, clientId string) (string, *errors.ServiceError) {
					return "", nil
				},
			},
			args: args{
				request:           CompleteServiceAccountRequest{Owner: "test"},
				tokenProviderFail: true,
			},
			expectGetTokenToBeCalled: true,
			wantErr:                  errors.NewWithCause(errors.ErrorGeneral, nil, "error getting access token\n caused by: failure retrieving token"),
		},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			getTokenCalled := false

			proxy := keycloakServiceProxy{
				getToken: testTokenProvider(token, &getTokenCalled, tt.args.tokenProviderFail),
				service:  tt.mock,
			}

			_, err := proxy.CreateServiceAccountInternal(tt.args.request)
			if tt.args.tokenProviderFail {
				g.Expect(err).To(gomega.HaveOccurred())
				g.Expect(tt.mock.calls.CreateServiceAccountInternal).To(gomega.HaveLen(0))
				g.Expect(err.Error()).To(gomega.Equal(tt.wantErr.Error()))
				g.Expect(err.Code).To(gomega.Equal(tt.wantErr.Code))
			} else {
				g.Expect(err).ToNot(gomega.HaveOccurred())
				g.Expect(tt.mock.calls.CreateServiceAccountInternal).To(gomega.HaveLen(1))
				g.Expect(tt.mock.calls.CreateServiceAccountInternal[0].Request).To(gomega.Equal(tt.args.request))
			}

			g.Expect(getTokenCalled).To(gomega.Equal(tt.expectGetTokenToBeCalled))
		})
	}
}

func Test_keycloakServiceProxy_DeleteServiceAccountInternal(t *testing.T) {
	type args struct {
		clientID          string
		tokenProviderFail bool
	}

	tests := []struct {
		name                     string
		mock                     *keycloakServiceInternalMock
		expectGetTokenToBeCalled bool
		args                     args
		wantErr                  *errors.ServiceError
	}{
		{
			name: "Test for RedhatSSO",
			mock: &keycloakServiceInternalMock{
				DeleteServiceAccountInternalFunc: func(accessToken string, clientId string) *errors.ServiceError {
					return nil
				},
				GetConfigFunc: func() *keycloak.KeycloakConfig {
					return &keycloak.KeycloakConfig{SelectSSOProvider: "redhat_sso"}
				},
			},
			args: args{
				clientID: testClientID,
			},
			expectGetTokenToBeCalled: true,
		},
		{
			name: "Test for MasSSO",
			mock: &keycloakServiceInternalMock{
				DeleteServiceAccountInternalFunc: func(accessToken string, clientId string) *errors.ServiceError {
					return nil
				},
				GetKafkaClientSecretFunc: func(accessToken string, clientId string) (string, *errors.ServiceError) {
					return "", nil
				},
			},
			args: args{
				clientID: testClientID,
			},
			expectGetTokenToBeCalled: true,
		},
		{
			name: "Should fail retrieving token",
			mock: &keycloakServiceInternalMock{
				DeleteServiceAccountInternalFunc: func(accessToken string, clientId string) *errors.ServiceError {
					return nil
				},
				GetKafkaClientSecretFunc: func(accessToken string, clientId string) (string, *errors.ServiceError) {
					return "", nil
				},
			},
			args: args{
				clientID:          testClientID,
				tokenProviderFail: true,
			},
			expectGetTokenToBeCalled: true,
			wantErr:                  errors.NewWithCause(errors.ErrorGeneral, nil, "error getting access token\n caused by: failure retrieving token"),
		},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			getTokenCalled := false

			proxy := keycloakServiceProxy{
				getToken: testTokenProvider(token, &getTokenCalled, tt.args.tokenProviderFail),
				service:  tt.mock,
			}

			err := proxy.DeleteServiceAccountInternal(tt.args.clientID)
			if tt.args.tokenProviderFail {
				g.Expect(err).To(gomega.HaveOccurred())
				g.Expect(tt.mock.calls.DeleteServiceAccountInternal).To(gomega.HaveLen(0))
				g.Expect(err.Error()).To(gomega.Equal(tt.wantErr.Error()))
				g.Expect(err.Code).To(gomega.Equal(tt.wantErr.Code))
			} else {
				g.Expect(err).ToNot(gomega.HaveOccurred())
				g.Expect(tt.mock.calls.DeleteServiceAccountInternal).To(gomega.HaveLen(1))
				g.Expect(tt.mock.calls.DeleteServiceAccountInternal[0].ClientId).To(gomega.Equal(tt.args.clientID))
			}

			g.Expect(getTokenCalled).To(gomega.Equal(tt.expectGetTokenToBeCalled))
		})
	}
}
