package acl

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/dgrijalva/jwt-go"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"

	. "github.com/onsi/gomega"
)

const (
	jwtKeyFile = "test/support/jwt_private_key.pem"
	jwtCAFile  = "test/support/jwt_ca.pem"
)

func Test_AccessControlListMiddleware_AccessControlListsDisabled(t *testing.T) {
	authHelper, err := auth.NewAuthHelper(jwtKeyFile, jwtCAFile, environments.Environment().Config.OCM.TokenIssuerURL)
	if err != nil {
		t.Fatal(err)
	}

	acc, err := authHelper.NewAccount("username", "test-user", "", "org-id-0")
	if err != nil {
		t.Fatal(err)
	}

	svcAcc, err := authHelper.NewAccount("svc-acc-username", "svc-acc-test-user", "", "")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name                 string
		arg                  services.ConfigService
		ctx                  context.Context
		filterByOrganisation bool
	}{
		{
			name: "returns 200 Ok response for internal users who belong to an organisation listed in the allow list",
			arg: services.NewConfigService(
				config.ApplicationConfig{
					AccessControlList: &config.AccessControlListConfig{
						EnableDenyList: false,
						AllowList: config.AllowListConfiguration{
							AllowAnyRegisteredUsers: true,
							Organisations: config.OrganisationList{
								config.Organisation{
									Id:              "org-id-0",
									AllowedAccounts: config.AllowedAccounts{config.AllowedAccount{Username: "username"}},
								},
							},
						},
					},
				},
			),
			filterByOrganisation: true,
			ctx:                  getAuthenticatedContext(authHelper, t, acc, nil),
		},
		{
			name: "returns 200 Ok response for internal users who are listed in the allow list",
			arg: services.NewConfigService(
				config.ApplicationConfig{
					AccessControlList: &config.AccessControlListConfig{
						EnableDenyList: false,
						AllowList: config.AllowListConfiguration{
							AllowAnyRegisteredUsers: true,
							ServiceAccounts: config.AllowedAccounts{
								config.AllowedAccount{Username: "username"},
							},
						},
					},
				},
			),
			filterByOrganisation: true,
			ctx:                  getAuthenticatedContext(authHelper, t, acc, nil),
		},
		{
			name: "returns 200 OK response for internal users who do not have access through their organisation but as a service account in the allow list",
			arg: services.NewConfigService(
				config.ApplicationConfig{
					AccessControlList: &config.AccessControlListConfig{
						EnableDenyList: false,
						AllowList: config.AllowListConfiguration{
							AllowAnyRegisteredUsers: true,
							Organisations: config.OrganisationList{
								config.Organisation{
									Id:              "org-id-0",
									AllowAll:        false,
									AllowedAccounts: config.AllowedAccounts{},
								},
							},
							ServiceAccounts: config.AllowedAccounts{
								config.AllowedAccount{Username: "username"},
							},
						},
					},
				},
			),
			filterByOrganisation: false,
			ctx:                  getAuthenticatedContext(authHelper, t, svcAcc, nil),
		},
		{
			name: "returns 200 Ok response for external users",
			arg: services.NewConfigService(
				config.ApplicationConfig{
					AccessControlList: &config.AccessControlListConfig{
						EnableDenyList: false,
						AllowList: config.AllowListConfiguration{
							AllowAnyRegisteredUsers: true,
						},
					},
				},
			),
			filterByOrganisation: true,
			ctx:                  getAuthenticatedContext(authHelper, t, acc, nil),
		},
		{
			name: "returns 200 Ok response for external users who are not included in their organisation in the allow list",
			arg: services.NewConfigService(
				config.ApplicationConfig{
					AccessControlList: &config.AccessControlListConfig{
						EnableDenyList: false,
						AllowList: config.AllowListConfiguration{
							AllowAnyRegisteredUsers: true,
							Organisations: config.OrganisationList{
								config.Organisation{
									Id:              "org-id-0",
									AllowAll:        false,
									AllowedAccounts: config.AllowedAccounts{},
								},
							},
						},
					},
				},
			),
			filterByOrganisation: true,
			ctx:                  getAuthenticatedContext(authHelper, t, acc, nil),
		},
		{
			name: "returns 200 Ok response for external service accounts",
			arg: services.NewConfigService(
				config.ApplicationConfig{
					AccessControlList: &config.AccessControlListConfig{
						EnableDenyList: false,
						AllowList: config.AllowListConfiguration{
							AllowAnyRegisteredUsers: true,
						},
					},
				},
			),
			filterByOrganisation: false,
			ctx:                  getAuthenticatedContext(authHelper, t, svcAcc, nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)

			req, err := http.NewRequest("GET", "/api/kafkas_mgmt/kafkas", nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()

			middleware := NewAccessControlListMiddleware(tt.arg)
			handler := middleware.Authorize(http.HandlerFunc(NextHandler))

			req = req.WithContext(tt.ctx)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))

			// verify that the context is set with whether the user is allowed as a service account or not
			ctxAfterMiddleware := req.Context()
			Expect(auth.GetFilterByOrganisationFromContext(ctxAfterMiddleware)).To(Equal(tt.filterByOrganisation))
		})
	}
}

func Test_AccessControlListMiddleware_UserHasNoAccess(t *testing.T) {
	authHelper, err := auth.NewAuthHelper(jwtKeyFile, jwtCAFile, environments.Environment().Config.OCM.TokenIssuerURL)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		arg  services.ConfigService
	}{
		{
			name: "returns 403 Forbidden response when user is not allowed to access service",
			arg: services.NewConfigService(
				config.ApplicationConfig{
					AccessControlList: &config.AccessControlListConfig{
						EnableDenyList: true,
						DenyList:       config.DeniedUsers{"username"},
						AllowList: config.AllowListConfiguration{
							AllowAnyRegisteredUsers: false,
							Organisations: config.OrganisationList{
								config.Organisation{
									Id:       "org-id-0",
									AllowAll: true,
								},
							},
							ServiceAccounts: config.AllowedAccounts{
								config.AllowedAccount{Username: "username"},
							},
						},
					},
				},
			),
		},
		{
			name: "returns 403 Forbidden response when user is not allowed to access service for the given organisation with allowed users",
			arg: services.NewConfigService(
				config.ApplicationConfig{
					AccessControlList: &config.AccessControlListConfig{
						AllowList: config.AllowListConfiguration{
							AllowAnyRegisteredUsers: false,
							Organisations: config.OrganisationList{
								config.Organisation{
									Id:              "org-id-0",
									AllowedAccounts: config.AllowedAccounts{config.AllowedAccount{Username: "another-username"}},
								},
							},
						},
					},
				},
			),
		},
		{
			name: "returns 403 Forbidden response when user is not allowed to access service for the given organisation with empty allowed users and no users is allowed to access the service",
			arg: services.NewConfigService(
				config.ApplicationConfig{
					AccessControlList: &config.AccessControlListConfig{
						AllowList: config.AllowListConfiguration{
							AllowAnyRegisteredUsers: false,
							Organisations: config.OrganisationList{
								config.Organisation{
									Id:              "org-id-0",
									AllowAll:        false,
									AllowedAccounts: config.AllowedAccounts{},
								},
							},
						},
					},
				},
			),
		},
		{
			name: "returns 403 Forbidden response when user organisation is not listed and user is not present in allowed service accounts list",
			arg: services.NewConfigService(
				config.ApplicationConfig{
					AccessControlList: &config.AccessControlListConfig{
						AllowList: config.AllowListConfiguration{
							AllowAnyRegisteredUsers: false,
							Organisations: config.OrganisationList{
								config.Organisation{
									Id:       "org-id-3",
									AllowAll: false,
								},
							},
							ServiceAccounts: config.AllowedAccounts{
								config.AllowedAccount{Username: "allowed-user-1"},
								config.AllowedAccount{Username: "allowed-user-2"},
							},
						},
					},
				},
			),
		},
		{
			name: "returns 403 Forbidden response when is not allowed to access the service through users organisation or the service accounts allow list",
			arg: services.NewConfigService(
				config.ApplicationConfig{
					AccessControlList: &config.AccessControlListConfig{
						AllowList: config.AllowListConfiguration{
							AllowAnyRegisteredUsers: false,
							Organisations: config.OrganisationList{
								config.Organisation{
									Id:              "org-id-0",
									AllowAll:        false,
									AllowedAccounts: config.AllowedAccounts{},
								},
							},
							ServiceAccounts: config.AllowedAccounts{
								config.AllowedAccount{Username: "allowed-user-1"},
								config.AllowedAccount{Username: "allowed-user-2"},
							},
						},
					},
				},
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)

			req, err := http.NewRequest("GET", "/api/kafkas_mgmt/kafkas", nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()

			middleware := NewAccessControlListMiddleware(tt.arg)
			handler := middleware.Authorize(http.HandlerFunc(NextHandler))

			// create a jwt and set it in the context
			ctx := req.Context()
			acc, err := authHelper.NewAccount("username", "test-user", "", "org-id-0")
			if err != nil {
				t.Fatal(err)
			}

			token, err := authHelper.CreateJWTWithClaims(acc, nil)
			if err != nil {
				t.Fatal(err)
			}

			ctx = auth.SetTokenInContext(ctx, token)
			req = req.WithContext(ctx)
			handler.ServeHTTP(rr, req)

			body, err := ioutil.ReadAll(rr.Body)
			if err != nil {
				t.Fatal(err)
			}

			Expect(rr.Code).To(Equal(http.StatusForbidden))
			Expect(rr.Header().Get("Content-Type")).To(Equal("application/json"))

			var data map[string]string
			err = json.Unmarshal(body, &data)
			if err != nil {
				t.Fatal(err)
			}
			Expect(data["kind"]).To(Equal("Error"))
			Expect(data["reason"]).To(Equal("User 'username' is not authorized to access the service."))
			// verify that context about user being allowed as service account is set to false always
			ctxAfterMiddleware := req.Context()
			Expect(auth.GetFilterByOrganisationFromContext(ctxAfterMiddleware)).To(Equal(false))
		})
	}
}

func Test_AccessControlListMiddleware_UserHasAccessViaAllowList(t *testing.T) {
	authHelper, err := auth.NewAuthHelper(jwtKeyFile, jwtCAFile, environments.Environment().Config.OCM.TokenIssuerURL)
	if err != nil {
		t.Fatal(err)
	}

	acc, err := authHelper.NewAccount("username", "test-user", "", "org-id-0")
	if err != nil {
		t.Fatal(err)
	}

	svcAcc, err := authHelper.NewAccount("svc-acc-username", "svc-acc-test-user", "", "")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name                 string
		arg                  services.ConfigService
		ctx                  context.Context
		filterByOrganisation bool
	}{
		{
			name: "returns 200 Ok response when user is allowed to access service for the given organisation with allowed users",
			arg: services.NewConfigService(
				config.ApplicationConfig{
					AccessControlList: &config.AccessControlListConfig{
						AllowList: config.AllowListConfiguration{
							AllowAnyRegisteredUsers: false,
							Organisations: config.OrganisationList{
								config.Organisation{
									Id:              "org-id-0",
									AllowedAccounts: config.AllowedAccounts{config.AllowedAccount{Username: "username"}},
								},
							},
						},
					},
				},
			),
			filterByOrganisation: true,
			ctx:                  getAuthenticatedContext(authHelper, t, acc, nil),
		},
		{
			name: "returns 200 OK response when user is allowed to access service for the given organisation with empty allowed users and all users are allowed to access the service",
			arg: services.NewConfigService(
				config.ApplicationConfig{
					AccessControlList: &config.AccessControlListConfig{
						AllowList: config.AllowListConfiguration{
							AllowAnyRegisteredUsers: false,
							Organisations: config.OrganisationList{
								config.Organisation{
									Id:              "org-id-0",
									AllowAll:        true,
									AllowedAccounts: config.AllowedAccounts{},
								},
							},
						},
					},
				},
			),
			filterByOrganisation: true,
			ctx:                  getAuthenticatedContext(authHelper, t, acc, nil),
		},
		{
			name: "returns 200 OK response when is not allowed to access the service through users organisation but through the service accounts allow list",
			arg: services.NewConfigService(
				config.ApplicationConfig{
					AccessControlList: &config.AccessControlListConfig{
						AllowList: config.AllowListConfiguration{
							AllowAnyRegisteredUsers: false,
							Organisations: config.OrganisationList{
								config.Organisation{
									Id:              "org-id-0",
									AllowAll:        false,
									AllowedAccounts: config.AllowedAccounts{},
								},
							},
							ServiceAccounts: config.AllowedAccounts{
								config.AllowedAccount{Username: "allowed-user-1"},
								config.AllowedAccount{Username: "svc-acc-username"},
							},
						},
					},
				},
			),
			filterByOrganisation: false,
			ctx:                  getAuthenticatedContext(authHelper, t, svcAcc, nil),
		},
		{
			name: "returns 200 Ok response when token used is retrieved from sso.redhat.com and user is allowed access to the service",
			arg: services.NewConfigService(
				config.ApplicationConfig{
					AccessControlList: &config.AccessControlListConfig{
						AllowList: config.AllowListConfiguration{
							AllowAnyRegisteredUsers: false,
							Organisations: config.OrganisationList{
								config.Organisation{
									Id:              "org-id-0",
									AllowedAccounts: config.AllowedAccounts{config.AllowedAccount{Username: "username"}},
								},
							},
						},
					},
				},
			),
			filterByOrganisation: true,
			ctx: getAuthenticatedContext(authHelper, t, acc, jwt.MapClaims{
				"username":           nil,
				"preferred_username": acc.Username(),
			}),
		},
		{
			name: "returns 200 OK response when user is allowed access as a service account",
			arg: services.NewConfigService(
				config.ApplicationConfig{
					AccessControlList: &config.AccessControlListConfig{
						AllowList: config.AllowListConfiguration{
							AllowAnyRegisteredUsers: false,
							ServiceAccounts: config.AllowedAccounts{
								config.AllowedAccount{Username: "allowed-user-1"},
								config.AllowedAccount{Username: "svc-acc-username"},
							},
						},
					},
				},
			),
			filterByOrganisation: false,
			ctx:                  getAuthenticatedContext(authHelper, t, svcAcc, nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)

			req, err := http.NewRequest("GET", "/api/kafkas_mgmt/kafkas", nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()

			middleware := NewAccessControlListMiddleware(tt.arg)
			handler := middleware.Authorize(http.HandlerFunc(NextHandler))

			req = req.WithContext(tt.ctx)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))

			// verify that the context is set with whether the user is allowed as a service account or not
			ctxAfterMiddleware := req.Context()
			Expect(auth.GetFilterByOrganisationFromContext(ctxAfterMiddleware)).To(Equal(tt.filterByOrganisation))
		})
	}
}

// NextHandler is a dummy handler that returns OK when AllowList middleware has passed
func NextHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK) //nolint
	_, err := io.WriteString(w, "OK")
	if err != nil {
		panic(err)
	}
}

func getAuthenticatedContext(authHelper *auth.AuthHelper, t *testing.T, acc *v1.Account, claims jwt.MapClaims) context.Context {
	// create a jwt and set it in the context
	token, err := authHelper.CreateJWTWithClaims(acc, claims)
	if err != nil {
		t.Fatal(err)
	}

	return auth.SetTokenInContext(context.Background(), token)
}
