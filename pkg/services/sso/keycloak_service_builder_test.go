package sso

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/onsi/gomega"
)

var (
	config = &keycloak.KeycloakConfig{
		EnableAuthenticationOnKafka: true,
		BaseURL:                     "base_url",
		SelectSSOProvider:           keycloak.REDHAT_SSO,
	}
	realmConfig = &keycloak.KeycloakRealmConfig{
		ClientID:     "clientId",
		ClientSecret: "clientSecret",
	}
)

func Test_keycloakServiceBuilder_Build_ForKFM(t *testing.T) {
	type fields struct {
		config      *keycloak.KeycloakConfig
		realmConfig *keycloak.KeycloakRealmConfig
	}
	client := keycloak.NewClient(config, realmConfig)
	tests := []struct {
		name   string
		fields fields
		want   KeycloakService
	}{
		{
			name: "should return an instance of the keycloak service",
			fields: fields{
				config:      config,
				realmConfig: realmConfig,
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
			service := NewKeycloakServiceBuilder().
				ForKFM().
				WithConfiguration(tt.fields.config).
				WithRealmConfig(tt.fields.realmConfig).
				Build()
			g.Expect(service).ToNot(gomega.BeNil())
			g.Expect(service.GetConfig()).To(gomega.Equal(tt.want.GetConfig()))
			g.Expect(service.GetRealmConfig()).To(gomega.Equal(tt.want.GetRealmConfig()))
		})
	}
}

func Test_osdKeycloackServiceBuilder_Build_ForOSD(t *testing.T) {
	type fields struct {
		config      *keycloak.KeycloakConfig
		realmConfig *keycloak.KeycloakRealmConfig
	}

	client := keycloak.NewClient(config, realmConfig)

	tests := []struct {
		name   string
		fields fields
		want   OSDKeycloakService
	}{
		{
			name: "should return an instance of the OSD keycloak service",
			fields: fields{
				config:      config,
				realmConfig: realmConfig,
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
			service := NewKeycloakServiceBuilder().
				ForOSD().
				WithConfiguration(tt.fields.config).
				WithRealmConfig(tt.fields.realmConfig).
				Build()

			g.Expect(service).ToNot(gomega.BeNil())
			g.Expect(service.GetConfig()).To(gomega.Equal(tt.want.GetConfig()))
			g.Expect(service.GetRealmConfig()).To(gomega.Equal(tt.want.GetRealmConfig()))
		})
	}
}
