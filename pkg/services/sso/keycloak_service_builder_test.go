package sso

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	. "github.com/onsi/gomega"
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
				accessTokenProvider: client,
				service: &masService{
					kcClient: client,
				},
			},
		},
	}
	g := NewWithT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewKeycloakServiceBuilder().
				ForKFM().
				WithConfiguration(tt.fields.config).
				WithRealmConfig(tt.fields.realmConfig).
				Build()
			g.Expect(service).ToNot(BeNil())
			g.Expect(service.GetConfig()).To(Equal(tt.want.GetConfig()))
			g.Expect(service.GetRealmConfig()).To(Equal(tt.want.GetRealmConfig()))
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
				accessTokenProvider: client,
				service: &masService{
					kcClient: client,
				},
			},
		},
	}
	g := NewWithT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewKeycloakServiceBuilder().
				ForOSD().
				WithConfiguration(tt.fields.config).
				WithRealmConfig(tt.fields.realmConfig).
				Build()

			g.Expect(service).ToNot(BeNil())
			g.Expect(service.GetConfig()).To(Equal(tt.want.GetConfig()))
			g.Expect(service.GetRealmConfig()).To(Equal(tt.want.GetRealmConfig()))
		})
	}
}
