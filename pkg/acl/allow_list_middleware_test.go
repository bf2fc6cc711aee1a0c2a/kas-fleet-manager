package acl

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/auth"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"

	. "github.com/onsi/gomega"
)

func Test_AllowListMiddleware_Disabled(t *testing.T) {
	RegisterTestingT(t)
	req, err := http.NewRequest("GET", "/api/managed-services/kafkas", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	middleware := NewAllowListMiddleware(services.NewConfigService(
		config.ProviderConfiguration{},
		config.AllowListConfig{
			EnableAllowList: false,
		}, config.ClusterCreationConfig{},
		config.ObservabilityConfiguration{},
	))
	handler := middleware.Authorize(http.HandlerFunc(NextHandler))

	handler.ServeHTTP(rr, req)

	Expect(rr.Code).To(Equal(http.StatusOK))
}

func Test_AllowListMiddleware_UserHasNoAccess(t *testing.T) {
	tests := []struct {
		name string
		arg  services.ConfigService
	}{
		{
			name: "returns 403 Forbidden response when user is not allowed to access service for the given organisation with allowed users",
			arg: services.NewConfigService(
				config.ProviderConfiguration{},
				config.AllowListConfig{
					EnableAllowList: true,
					AllowList: config.AllowListConfiguration{
						Organisations: config.OrganisationList{
							config.Organisation{
								Id:              "org-id-0",
								AllowedAccounts: config.AllowedAccounts{config.AllowedAccount{Username: "another-username"}},
							},
						},
					},
				},
				config.ClusterCreationConfig{},
				config.ObservabilityConfiguration{},
			),
		},
		{
			name: "returns 403 Forbidden response when user is not allowed to access service for the given organisation with empty allowed users and no users is allowed to access the service",
			arg: services.NewConfigService(
				config.ProviderConfiguration{},
				config.AllowListConfig{
					EnableAllowList: true,
					AllowList: config.AllowListConfiguration{
						Organisations: config.OrganisationList{
							config.Organisation{
								Id:              "org-id-0",
								AllowAll:        false,
								AllowedAccounts: config.AllowedAccounts{},
							},
						},
					},
				},
				config.ClusterCreationConfig{},
				config.ObservabilityConfiguration{},
			),
		},
		{
			name: "returns 403 Forbidden response when user organisation is not listed and user is not present in allowed service accounts list",
			arg: services.NewConfigService(
				config.ProviderConfiguration{},
				config.AllowListConfig{
					EnableAllowList: true,
					AllowList: config.AllowListConfiguration{
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
				config.ClusterCreationConfig{},
				config.ObservabilityConfiguration{},
			),
		},
		{
			name: "returns 403 Forbidden response when is not allowed to access the service through users organisation or the service accounts allow list",
			arg: services.NewConfigService(
				config.ProviderConfiguration{},
				config.AllowListConfig{
					EnableAllowList: true,
					AllowList: config.AllowListConfiguration{
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
				config.ClusterCreationConfig{},
				config.ObservabilityConfiguration{},
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)

			req, err := http.NewRequest("GET", "/api/managed-services/kafkas", nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()

			middleware := NewAllowListMiddleware(tt.arg)
			handler := middleware.Authorize(http.HandlerFunc(NextHandler))

			// set username and organisation id within the current context
			ctx := req.Context()
			ctx = auth.SetUsernameContext(ctx, "username")
			ctx = auth.SetOrgIdContext(ctx, "org-id-0")

			req = req.WithContext(ctx)
			handler.ServeHTTP(rr, req)

			body, err := ioutil.ReadAll(rr.Body)
			if err != nil {
				t.Fatal(err)
			}
			var data map[string]string
			err = json.Unmarshal(body, &data)
			if err != nil {
				t.Fatal(err)
			}
			Expect(rr.Code).To(Equal(http.StatusForbidden))
			Expect(rr.Header().Get("Content-Type")).To(Equal("application/json"))
			Expect(data["kind"]).To(Equal("Error"))
			Expect(data["reason"]).To(Equal("User 'username' is not authorized to access the service."))
		})
	}

}

func Test_AllowListMiddleware_UserHasAccess(t *testing.T) {
	tests := []struct {
		name string
		arg  services.ConfigService
	}{
		{
			name: "returns 200 Ok response when user is allowed to access service for the given organisation with allowed users",
			arg: services.NewConfigService(
				config.ProviderConfiguration{},
				config.AllowListConfig{
					EnableAllowList: true,
					AllowList: config.AllowListConfiguration{
						Organisations: config.OrganisationList{
							config.Organisation{
								Id:              "org-id-0",
								AllowedAccounts: config.AllowedAccounts{config.AllowedAccount{Username: "username"}},
							},
						},
					},
				},
				config.ClusterCreationConfig{},
				config.ObservabilityConfiguration{},
			),
		},
		{
			name: "returns 200 OK response when user is allowed to access service for the given organisation with empty allowed users and all users are allowed to access the service",
			arg: services.NewConfigService(
				config.ProviderConfiguration{},
				config.AllowListConfig{
					EnableAllowList: true,
					AllowList: config.AllowListConfiguration{
						Organisations: config.OrganisationList{
							config.Organisation{
								Id:              "org-id-0",
								AllowAll:        true,
								AllowedAccounts: config.AllowedAccounts{},
							},
						},
					},
				},
				config.ClusterCreationConfig{},
				config.ObservabilityConfiguration{}),
		},
		{
			name: "returns 200 OK response when is not allowed to access the service through users organisation but through the service accounts allow list",
			arg: services.NewConfigService(
				config.ProviderConfiguration{},
				config.AllowListConfig{
					EnableAllowList: true,
					AllowList: config.AllowListConfiguration{
						Organisations: config.OrganisationList{
							config.Organisation{
								Id:              "org-id-0",
								AllowAll:        false,
								AllowedAccounts: config.AllowedAccounts{},
							},
						},
						ServiceAccounts: config.AllowedAccounts{
							config.AllowedAccount{Username: "allowed-user-1"},
							config.AllowedAccount{Username: "username"},
						},
					},
				},
				config.ClusterCreationConfig{},
				config.ObservabilityConfiguration{},
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)

			req, err := http.NewRequest("GET", "/api/managed-services/kafkas", nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()

			middleware := NewAllowListMiddleware(tt.arg)
			handler := middleware.Authorize(http.HandlerFunc(NextHandler))

			// set username and organisation id within the current context
			ctx := req.Context()
			ctx = auth.SetUsernameContext(ctx, "username")
			ctx = auth.SetOrgIdContext(ctx, "org-id-0")

			req = req.WithContext(ctx)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))
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
