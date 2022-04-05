package sso

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/redhatsso"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
)

var _ KeycloakServiceBuilderSelector = &keycloakServiceBuilderSelector{}
var _ KeycloakServiceBuilder = &keycloakServiceBuilder{}

type KeycloakServiceBuilder interface {
	WithRealmConfig(realmConfig *keycloak.KeycloakRealmConfig) KeycloakServiceBuilder
	Build() KeycloakService
}

type keycloakServiceBuilder struct {
	config      *keycloak.KeycloakConfig
	realmConfig *keycloak.KeycloakRealmConfig
}

// Build returns an instance of KeycloakService ready to be used.
// If a custom realm is configured (WithRealmConfig called), then always Keycloak provider is used
// irrespective of the `builder.config.SelectSSOProvider` value
func (builder *keycloakServiceBuilder) Build() KeycloakService {
	notNilPredicate := func(x interface{}) bool {
		return x.(*keycloak.KeycloakRealmConfig) != nil
	}

	// Temporary: if a realm configuration different from the one into the config is specified
	// we always instantiate MAS_SSO irrespective of the selected provider
	if builder.config.SelectSSOProvider == keycloak.MAS_SSO ||
		builder.realmConfig != nil {
		_, realmConfig := arrays.FindFirst(notNilPredicate, builder.realmConfig, builder.config.KafkaRealm)
		return newKeycloakService(builder.config, realmConfig.(*keycloak.KeycloakRealmConfig))
	} else {
		_, realmConfig := arrays.FindFirst(notNilPredicate, builder.realmConfig, builder.config.RedhatSSORealm)
		client := redhatsso.NewSSOClient(builder.config, realmConfig.(*keycloak.KeycloakRealmConfig))
		return &keycloakServiceProxy{
			accessTokenProvider: client,
			service: &redhatssoService{
				client: client,
			},
		}
	}
}

func (builder *keycloakServiceBuilder) WithRealmConfig(realmConfig *keycloak.KeycloakRealmConfig) KeycloakServiceBuilder {
	builder.realmConfig = realmConfig
	return builder
}

type KeycloakServiceBuilderSelector interface {
	WithConfiguration(config *keycloak.KeycloakConfig) KeycloakServiceBuilder
}

type keycloakServiceBuilderSelector struct {
}

func (s *keycloakServiceBuilderSelector) WithConfiguration(config *keycloak.KeycloakConfig) KeycloakServiceBuilder {
	return &keycloakServiceBuilder{
		config: config,
	}
}
