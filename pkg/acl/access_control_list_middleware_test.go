package acl_test

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/acl"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/golang/glog"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/dgrijalva/jwt-go"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"

	. "github.com/onsi/gomega"
)

const (
	jwtKeyFile = "test/support/jwt_private_key.pem"
	jwtCAFile  = "test/support/jwt_ca.pem"
)

var env *environments.Env
var ocmConfig *ocm.OCMConfig

func TestMain(m *testing.M) {
	var err error
	env, err = environments.New(environments.GetEnvironmentStrFromEnv(),
		kafka.ConfigProviders(),
	)
	if err != nil {
		glog.Fatalf("error initializing: %v", err)
	}
	env.MustResolveAll(&ocmConfig)
	os.Exit(m.Run())
}

func Test_AccessControlListMiddleware_AccessControlListsDisabled(t *testing.T) {
	authHelper, err := auth.NewAuthHelper(jwtKeyFile, jwtCAFile, ocmConfig.TokenIssuerURL)
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
		arg                  *acl.AccessControlListConfig
		ctx                  context.Context
		filterByOrganisation bool
	}{
		{
			name: "returns 200 Ok response for internal users who belong to an organisation listed in the allow list",
			arg: &acl.AccessControlListConfig{
				EnableDenyList: false,
				AllowList: acl.AllowListConfiguration{
					AllowAnyRegisteredUsers: true,
					Organisations: acl.OrganisationList{
						acl.Organisation{
							Id:              "org-id-0",
							AllowedAccounts: acl.AllowedAccounts{acl.AllowedAccount{Username: "username"}},
						},
					},
				},
			},
			filterByOrganisation: true,
			ctx:                  getAuthenticatedContext(authHelper, t, acc, nil),
		},
		{
			name: "returns 200 Ok response for internal users who are listed in the allow list",
			arg: &acl.AccessControlListConfig{
				EnableDenyList: false,
				AllowList: acl.AllowListConfiguration{
					AllowAnyRegisteredUsers: true,
					ServiceAccounts: acl.AllowedAccounts{
						acl.AllowedAccount{Username: "username"},
					},
				},
			},
			filterByOrganisation: true,
			ctx:                  getAuthenticatedContext(authHelper, t, acc, nil),
		},
		{
			name: "returns 200 OK response for internal users who do not have access through their organisation but as a service account in the allow list",
			arg: &acl.AccessControlListConfig{
				EnableDenyList: false,
				AllowList: acl.AllowListConfiguration{
					AllowAnyRegisteredUsers: true,
					Organisations: acl.OrganisationList{
						acl.Organisation{
							Id:              "org-id-0",
							AllowAll:        false,
							AllowedAccounts: acl.AllowedAccounts{},
						},
					},
					ServiceAccounts: acl.AllowedAccounts{
						acl.AllowedAccount{Username: "username"},
					},
				},
			},
			filterByOrganisation: false,
			ctx:                  getAuthenticatedContext(authHelper, t, svcAcc, nil),
		},
		{
			name: "returns 200 Ok response for external users",
			arg: &acl.AccessControlListConfig{
				EnableDenyList: false,
				AllowList: acl.AllowListConfiguration{
					AllowAnyRegisteredUsers: true,
				},
			},
			filterByOrganisation: true,
			ctx:                  getAuthenticatedContext(authHelper, t, acc, nil),
		},
		{
			name: "returns 200 Ok response for external users who are not included in their organisation in the allow list",
			arg: &acl.AccessControlListConfig{
				EnableDenyList: false,
				AllowList: acl.AllowListConfiguration{
					AllowAnyRegisteredUsers: true,
					Organisations: acl.OrganisationList{
						acl.Organisation{
							Id:              "org-id-0",
							AllowAll:        false,
							AllowedAccounts: acl.AllowedAccounts{},
						},
					},
				},
			},
			filterByOrganisation: true,
			ctx:                  getAuthenticatedContext(authHelper, t, acc, nil),
		},
		{
			name: "returns 200 Ok response for external service accounts",
			arg: &acl.AccessControlListConfig{
				EnableDenyList: false,
				AllowList: acl.AllowListConfiguration{
					AllowAnyRegisteredUsers: true,
				},
			},
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

			middleware := acl.NewAccessControlListMiddleware(tt.arg)
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
	authHelper, err := auth.NewAuthHelper(jwtKeyFile, jwtCAFile, ocmConfig.TokenIssuerURL)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		arg  *acl.AccessControlListConfig
		want *errors.ServiceError
	}{
		{
			name: "returns 403 Forbidden response when user is not allowed to access service",
			arg: &acl.AccessControlListConfig{
				EnableDenyList: true,
				DenyList:       acl.DeniedUsers{"username"},
				AllowList: acl.AllowListConfiguration{
					AllowAnyRegisteredUsers: false,
					Organisations: acl.OrganisationList{
						acl.Organisation{
							Id:       "org-id-0",
							AllowAll: true,
						},
					},
					ServiceAccounts: acl.AllowedAccounts{
						acl.AllowedAccount{Username: "username"},
					},
				},
			},
		},
		{
			name: "returns 403 Forbidden response when user is not allowed to access service for the given organisation with allowed users",
			arg: &acl.AccessControlListConfig{
				AllowList: acl.AllowListConfiguration{
					AllowAnyRegisteredUsers: false,
					Organisations: acl.OrganisationList{
						acl.Organisation{
							Id:              "org-id-0",
							AllowedAccounts: acl.AllowedAccounts{acl.AllowedAccount{Username: "another-username"}},
						},
					},
				},
			},
		},
		{
			name: "returns 403 Forbidden response when user is not allowed to access service for the given organisation with empty allowed users and no users is allowed to access the service",
			arg: &acl.AccessControlListConfig{
				AllowList: acl.AllowListConfiguration{
					AllowAnyRegisteredUsers: false,
					Organisations: acl.OrganisationList{
						acl.Organisation{
							Id:              "org-id-0",
							AllowAll:        false,
							AllowedAccounts: acl.AllowedAccounts{},
						},
					},
				},
			},
		},
		{
			name: "returns 403 Forbidden response when user organisation is not listed and user is not present in allowed service accounts list",
			arg: &acl.AccessControlListConfig{
				AllowList: acl.AllowListConfiguration{
					AllowAnyRegisteredUsers: false,
					Organisations: acl.OrganisationList{
						acl.Organisation{
							Id:       "org-id-3",
							AllowAll: false,
						},
					},
					ServiceAccounts: acl.AllowedAccounts{
						acl.AllowedAccount{Username: "allowed-user-1"},
						acl.AllowedAccount{Username: "allowed-user-2"},
					},
				},
			},
		},
		{
			name: "returns 403 Forbidden response when is not allowed to access the service through users organisation or the service accounts allow list",
			arg: &acl.AccessControlListConfig{
				AllowList: acl.AllowListConfiguration{
					AllowAnyRegisteredUsers: false,
					Organisations: acl.OrganisationList{
						acl.Organisation{
							Id:              "org-id-0",
							AllowAll:        false,
							AllowedAccounts: acl.AllowedAccounts{},
						},
					},
					ServiceAccounts: acl.AllowedAccounts{
						acl.AllowedAccount{Username: "allowed-user-1"},
						acl.AllowedAccount{Username: "allowed-user-2"},
					},
				},
			},
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

			middleware := acl.NewAccessControlListMiddleware(tt.arg)
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
	authHelper, err := auth.NewAuthHelper(jwtKeyFile, jwtCAFile, ocmConfig.TokenIssuerURL)
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
		arg                  *acl.AccessControlListConfig
		ctx                  context.Context
		filterByOrganisation bool
	}{
		{
			name: "returns 200 Ok response when user is allowed to access service for the given organisation with allowed users",
			arg: &acl.AccessControlListConfig{
				AllowList: acl.AllowListConfiguration{
					AllowAnyRegisteredUsers: false,
					Organisations: acl.OrganisationList{
						acl.Organisation{
							Id:              "org-id-0",
							AllowedAccounts: acl.AllowedAccounts{acl.AllowedAccount{Username: "username"}},
						},
					},
				},
			},
			filterByOrganisation: true,
			ctx:                  getAuthenticatedContext(authHelper, t, acc, nil),
		},
		{
			name: "returns 200 OK response when user is allowed to access service for the given organisation with empty allowed users and all users are allowed to access the service",
			arg: &acl.AccessControlListConfig{
				AllowList: acl.AllowListConfiguration{
					AllowAnyRegisteredUsers: false,
					Organisations: acl.OrganisationList{
						acl.Organisation{
							Id:              "org-id-0",
							AllowAll:        true,
							AllowedAccounts: acl.AllowedAccounts{},
						},
					},
				},
			},
			filterByOrganisation: true,
			ctx:                  getAuthenticatedContext(authHelper, t, acc, nil),
		},
		{
			name: "returns 200 OK response when is not allowed to access the service through users organisation but through the service accounts allow list",
			arg: &acl.AccessControlListConfig{
				AllowList: acl.AllowListConfiguration{
					AllowAnyRegisteredUsers: false,
					Organisations: acl.OrganisationList{
						acl.Organisation{
							Id:              "org-id-0",
							AllowAll:        false,
							AllowedAccounts: acl.AllowedAccounts{},
						},
					},
					ServiceAccounts: acl.AllowedAccounts{
						acl.AllowedAccount{Username: "allowed-user-1"},
						acl.AllowedAccount{Username: "svc-acc-username"},
					},
				},
			},
			filterByOrganisation: false,
			ctx:                  getAuthenticatedContext(authHelper, t, svcAcc, nil),
		},
		{
			name: "returns 200 Ok response when token used is retrieved from sso.redhat.com and user is allowed access to the service",
			arg: &acl.AccessControlListConfig{
				AllowList: acl.AllowListConfiguration{
					AllowAnyRegisteredUsers: false,
					Organisations: acl.OrganisationList{
						acl.Organisation{
							Id:              "org-id-0",
							AllowedAccounts: acl.AllowedAccounts{acl.AllowedAccount{Username: "username"}},
						},
					},
				},
			},
			filterByOrganisation: true,
			ctx: getAuthenticatedContext(authHelper, t, acc, jwt.MapClaims{
				"username":           nil,
				"preferred_username": acc.Username(),
			}),
		},
		{
			name: "returns 200 OK response when user is allowed access as a service account",
			arg: &acl.AccessControlListConfig{
				AllowList: acl.AllowListConfiguration{
					AllowAnyRegisteredUsers: false,
					ServiceAccounts: acl.AllowedAccounts{
						acl.AllowedAccount{Username: "allowed-user-1"},
						acl.AllowedAccount{Username: "svc-acc-username"},
					},
				},
			},
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

			middleware := acl.NewAccessControlListMiddleware(tt.arg)
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
