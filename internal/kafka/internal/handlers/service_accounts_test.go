package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sso"
	"github.com/gorilla/mux"
	"github.com/onsi/gomega"
)

var (
	createServiceAccountRequest = `{"name": "my-app-sa","description": "service account for my app"}`
)

func TestNewServiceAccountHandler(t *testing.T) {
	type args struct {
		service sso.KafkaKeycloakService
	}
	tests := []struct {
		name string
		args args
		want *serviceAccountsHandler
	}{
		{
			name: "should return a NewServiceAccountHandler",
			args: args{
				service: &sso.KeycloakServiceMock{},
			},
			want: &serviceAccountsHandler{
				service: &sso.KeycloakServiceMock{},
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(NewServiceAccountHandler(tt.args.service)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_serviceAccountsHandler_ListServiceAccounts(t *testing.T) {
	type fields struct {
		service sso.KeycloakService
	}
	type args struct {
		url string
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
	}{
		{
			name: "should successfully list service accounts",
			fields: fields{
				service: &sso.KeycloakServiceMock{
					ListServiceAccFunc: func(ctx context.Context, first, max int) ([]api.ServiceAccount, *errors.ServiceError) {
						return []api.ServiceAccount{
							{
								ClientID: "client-id",
							},
							{
								ClientID: "client-id",
							},
						}, nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{}
					},
				},
			},
			args: args{
				url: "/api/kafkas_mgmt/v1/service_accounts",
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "should return an error because it fails to list service accounts",
			fields: fields{
				service: &sso.KeycloakServiceMock{
					ListServiceAccFunc: func(ctx context.Context, first, max int) ([]api.ServiceAccount, *errors.ServiceError) {
						return nil, errors.GeneralError("failed to list service accounts")
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{}
					},
				},
			},
			args: args{
				url: "/api/kafkas_mgmt/v1/service_accounts",
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			req, rw := GetHandlerParams("GET", tt.args.url, nil, t)

			h := NewServiceAccountHandler(tt.fields.service)
			h.ListServiceAccounts(rw, req)
			resp := rw.Result()
			resp.Body.Close()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
		})
	}
}

func Test_serviceAccountsHandler_CreateServiceAccount(t *testing.T) {
	type fields struct {
		service sso.KeycloakService
	}
	type args struct {
		url  string
		body []byte
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
	}{
		{
			name: "should return status code 202 if the request was accepted successfully",
			fields: fields{
				service: &sso.KeycloakServiceMock{CreateServiceAccountFunc: func(serviceAccountRequest *api.ServiceAccountRequest, ctx context.Context) (*api.ServiceAccount, *errors.ServiceError) {
					return &api.ServiceAccount{}, nil
				}},
			},
			args: args{
				url:  "/api/kafkas_mgmt/v1/service_accounts",
				body: []byte(createServiceAccountRequest),
			},
			wantStatusCode: http.StatusAccepted,
		},
		{
			name: "should return status code 500 if it fails to create the service account",
			fields: fields{
				service: &sso.KeycloakServiceMock{CreateServiceAccountFunc: func(serviceAccountRequest *api.ServiceAccountRequest, ctx context.Context) (*api.ServiceAccount, *errors.ServiceError) {
					return nil, errors.GeneralError("error creating service account")
				}},
			},
			args: args{
				url:  "/api/kafkas_mgmt/v1/service_accounts",
				body: []byte(createServiceAccountRequest),
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			req, rw := GetHandlerParams("POST", tt.args.url, bytes.NewBuffer(tt.args.body), t)

			h := NewServiceAccountHandler(tt.fields.service)
			h.CreateServiceAccount(rw, req)
			resp := rw.Result()
			resp.Body.Close()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
		})
	}
}

func Test_serviceAccountsHandler_DeleteServiceAccount(t *testing.T) {
	type fields struct {
		service sso.KeycloakService
	}
	type args struct {
		url string
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
	}{
		{
			name: "should return status code 204 if it deletes the service account successfully",
			fields: fields{
				service: &sso.KeycloakServiceMock{
					DeleteServiceAccountFunc: func(ctx context.Context, clientId string) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				url: "/api/kafkas_mgmt/v1/service_accounts/{id}",
			},
			wantStatusCode: http.StatusNoContent,
		},
		{
			name: "should return status code 500 if it fails to delete the service account",
			fields: fields{
				service: &sso.KeycloakServiceMock{
					DeleteServiceAccountFunc: func(ctx context.Context, clientId string) *errors.ServiceError {
						return errors.GeneralError("failed to delete the service account")
					},
				},
			},
			args: args{
				url: "/api/kafkas_mgmt/v1/service_accounts/{id}",
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			req, rw := GetHandlerParams("DELETE", tt.args.url, nil, t)
			req = mux.SetURLVars(req, map[string]string{"id": "b5843c4b-a702-100d-fc77-70e9b20e554f"})

			h := NewServiceAccountHandler(tt.fields.service)
			h.DeleteServiceAccount(rw, req)
			resp := rw.Result()
			resp.Body.Close()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
		})
	}
}

func Test_serviceAccountsHandler_ResetServiceAccountCredential(t *testing.T) {
	type fields struct {
		service sso.KeycloakService
	}
	type args struct {
		url string
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
	}{
		{
			name: "should return status code 200 if it successfully resets their service accounts credentials",
			fields: fields{
				service: &sso.KeycloakServiceMock{
					ResetServiceAccountCredentialsFunc: func(ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError) {
						return &api.ServiceAccount{}, nil
					},
				},
			},
			args: args{
				url: "/api/kafkas_mgmt/v1/service_accounts/{id}/reset_credentials",
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "should return status code 500 if it does not reset the service account credentials",
			fields: fields{
				service: &sso.KeycloakServiceMock{
					ResetServiceAccountCredentialsFunc: func(ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError) {
						return nil, errors.GeneralError("failed to reset the service account credentials")
					},
				},
			},
			args: args{
				url: "/api/kafkas_mgmt/v1/service_accounts/{id}/reset_credentials",
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			req, rw := GetHandlerParams("POST", tt.args.url, nil, t)
			req = mux.SetURLVars(req, map[string]string{"id": "b5843c4b-a702-100d-fc77-70e9b20e554f"})

			h := NewServiceAccountHandler(tt.fields.service)
			h.ResetServiceAccountCredential(rw, req)
			resp := rw.Result()
			resp.Body.Close()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
		})
	}
}

func Test_serviceAccountsHandler_GetServiceAccountByClientId(t *testing.T) {
	type fields struct {
		service sso.KeycloakService
	}
	type args struct {
		url string
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
	}{
		{
			name: "should return status code 200 if it can get the client by the client id",
			fields: fields{
				service: &sso.KeycloakServiceMock{
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							SelectSSOProvider: keycloak.MAS_SSO,
						}
					},
					GetServiceAccountByClientIdFunc: func(ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError) {
						return &api.ServiceAccount{}, nil
					},
				},
			},
			args: args{
				url: "/api/kafkas_mgmt/v1/service_accounts/{client_id}",
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "should return status code 200 if it can get the client by the client id",
			fields: fields{
				service: &sso.KeycloakServiceMock{
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							SelectSSOProvider: keycloak.MAS_SSO,
						}
					},
					GetServiceAccountByClientIdFunc: func(ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError) {
						return nil, errors.GeneralError("failed to get service acccont")
					},
				},
			},
			args: args{
				url: "/api/kafkas_mgmt/v1/service_accounts/{client_id}",
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			req, rw := GetHandlerParams("GET", tt.args.url, nil, t)

			req.Form = url.Values{}
			req.Form.Add("client_id", "srvc-acct-7f4f2226-f0cc-7f40-8d74-9b38934d2be0")

			h := NewServiceAccountHandler(tt.fields.service)
			h.GetServiceAccountByClientId(rw, req)
			resp := rw.Result()
			resp.Body.Close()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
		})
	}
}

func Test_serviceAccountsHandler_GetServiceAccountById(t *testing.T) {
	type fields struct {
		service sso.KeycloakService
	}
	type args struct {
		url string
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
	}{
		{
			name: "should return status code 200 if it successfully gets the service account by id",
			fields: fields{
				service: &sso.KeycloakServiceMock{
					GetServiceAccountByIdFunc: func(ctx context.Context, id string) (*api.ServiceAccount, *errors.ServiceError) {
						return &api.ServiceAccount{}, nil
					},
				},
			},
			args: args{
				url: "/api/kafkas_mgmt/v1/service_accounts/{id}",
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "should return status code 500 if it fails to get the service account by id",
			fields: fields{
				service: &sso.KeycloakServiceMock{
					GetServiceAccountByIdFunc: func(ctx context.Context, id string) (*api.ServiceAccount, *errors.ServiceError) {
						return nil, errors.GeneralError("failed to get service acccont")
					},
				},
			},
			args: args{
				url: "/api/kafkas_mgmt/v1/service_accounts/{id}",
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			req, rw := GetHandlerParams("GET", tt.args.url, nil, t)
			req = mux.SetURLVars(req, map[string]string{"id": "b5843c4b-a702-100d-fc77-70e9b20e554f"})

			h := NewServiceAccountHandler(tt.fields.service)
			h.GetServiceAccountById(rw, req)
			resp := rw.Result()
			resp.Body.Close()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
		})
	}
}

func Test_serviceAccountsHandler_GetSsoProviders(t *testing.T) {
	type fields struct {
		service sso.KeycloakService
	}
	type args struct {
		url string
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
	}{
		{
			name: "should return status code 200 if it successfully retrieves the sso provider",
			fields: fields{
				service: &sso.KeycloakServiceMock{
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{}
					},
					GetRealmConfigFunc: func() *keycloak.KeycloakRealmConfig {
						return &keycloak.KeycloakRealmConfig{}
					},
				},
			},
			args: args{
				url: "/api/kafkas_mgmt/v1/sso_providers",
			},
			wantStatusCode: http.StatusOK,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			req, rw := GetHandlerParams("GET", tt.args.url, nil, t)

			h := NewServiceAccountHandler(tt.fields.service)
			h.GetSsoProviders(rw, req)
			resp := rw.Result()
			resp.Body.Close()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
		})
	}
}
