package sso

import (
	"context"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/redhatsso"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

type Provider string

const (
	KeycloakProvider  Provider = "KEYCLOAK"
	RedhatSSOProvider Provider = "REDHAT"
)

type CompleteServiceAccountRequest struct {
	Owner          string
	OwnerAccountId string
	OrgId          string
	ClientId       string
	Name           string
	Description    string
}

//go:generate moq -out keycloakservice_moq.go . KeycloakService
type KeycloakService interface {
	RegisterKafkaClientInSSO(kafkaNamespace string, orgId string) (string, *errors.ServiceError)
	RegisterOSDClusterClientInSSO(clusterId string, clusterOathCallbackURI string) (string, *errors.ServiceError)
	DeRegisterClientInSSO(kafkaNamespace string) *errors.ServiceError
	GetConfig() *keycloak.KeycloakConfig
	GetRealmConfig() *keycloak.KeycloakRealmConfig
	IsKafkaClientExist(clientId string) *errors.ServiceError
	CreateServiceAccount(serviceAccountRequest *api.ServiceAccountRequest, ctx context.Context) (*api.ServiceAccount, *errors.ServiceError)
	DeleteServiceAccount(ctx context.Context, clientId string) *errors.ServiceError
	ResetServiceAccountCredentials(ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError)
	ListServiceAcc(ctx context.Context, first int, max int) ([]api.ServiceAccount, *errors.ServiceError)
	RegisterKasFleetshardOperatorServiceAccount(agentClusterId string) (*api.ServiceAccount, *errors.ServiceError)
	DeRegisterKasFleetshardOperatorServiceAccount(agentClusterId string) *errors.ServiceError
	GetServiceAccountById(ctx context.Context, id string) (*api.ServiceAccount, *errors.ServiceError)
	GetServiceAccountByClientId(ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError)
	RegisterConnectorFleetshardOperatorServiceAccount(agentClusterId string) (*api.ServiceAccount, *errors.ServiceError)
	DeRegisterConnectorFleetshardOperatorServiceAccount(agentClusterId string) *errors.ServiceError
	GetKafkaClientSecret(clientId string) (string, *errors.ServiceError)
	CreateServiceAccountInternal(request CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError)
	DeleteServiceAccountInternal(clientId string) *errors.ServiceError
}

type keycloakServiceInternal interface {
	RegisterKafkaClientInSSO(accessToken string, kafkaNamespace string, orgId string) (string, *errors.ServiceError)
	RegisterOSDClusterClientInSSO(accessToken string, clusterId string, clusterOathCallbackURI string) (string, *errors.ServiceError)
	DeRegisterClientInSSO(accessToken string, kafkaNamespace string) *errors.ServiceError
	GetConfig() *keycloak.KeycloakConfig
	GetRealmConfig() *keycloak.KeycloakRealmConfig
	IsKafkaClientExist(accessToken string, clientId string) *errors.ServiceError
	CreateServiceAccount(accessToken string, serviceAccountRequest *api.ServiceAccountRequest, ctx context.Context) (*api.ServiceAccount, *errors.ServiceError)
	DeleteServiceAccount(accessToken string, ctx context.Context, clientId string) *errors.ServiceError
	ResetServiceAccountCredentials(accessToken string, ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError)
	ListServiceAcc(accessToken string, ctx context.Context, first int, max int) ([]api.ServiceAccount, *errors.ServiceError)
	RegisterKasFleetshardOperatorServiceAccount(accessToken string, agentClusterId string) (*api.ServiceAccount, *errors.ServiceError)
	DeRegisterKasFleetshardOperatorServiceAccount(accessToken string, agentClusterId string) *errors.ServiceError
	GetServiceAccountById(accessToken string, ctx context.Context, id string) (*api.ServiceAccount, *errors.ServiceError)
	GetServiceAccountByClientId(accessToken string, ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError)
	RegisterConnectorFleetshardOperatorServiceAccount(accessToken string, agentClusterId string) (*api.ServiceAccount, *errors.ServiceError)
	DeRegisterConnectorFleetshardOperatorServiceAccount(accessToken string, agentClusterId string) *errors.ServiceError
	GetKafkaClientSecret(accessToken string, clientId string) (string, *errors.ServiceError)
	CreateServiceAccountInternal(accessToken string, request CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError)
	DeleteServiceAccountInternal(accessToken string, clientId string) *errors.ServiceError
}

func NewKeycloakServiceWithClient(client keycloak.KcClient) KeycloakService {
	return &keycloakServiceProxy{
		accessTokenProvider: client,
		service: &masService{
			kcClient: client,
		},
	}
}

func NewKeycloakService(config *keycloak.KeycloakConfig, realmConfig *keycloak.KeycloakRealmConfig) KeycloakService {
	client := keycloak.NewClient(config, realmConfig)
	return &keycloakServiceProxy{
		accessTokenProvider: client,
		service: &masService{
			kcClient: client,
		},
	}
}

// TODO: this should be refactored to a builder pattern so that the `interface{}` won't be needed anymore
func NewKeycloakServiceByType(provider Provider, config interface{}, realmConfig *keycloak.KeycloakRealmConfig) (KeycloakService, error) {
	if provider == KeycloakProvider {
		if cfg, ok := config.(*keycloak.KeycloakConfig); ok {
			client := keycloak.NewClient(cfg, realmConfig)
			return &keycloakServiceProxy{
				accessTokenProvider: client,
				service: &masService{
					kcClient: client,
				},
			}, nil
		} else {
			return nil, fmt.Errorf("invalid configuration for provider %s", provider)
		}
	}
	if provider == RedhatSSOProvider {
		if cfg, ok := config.(*redhatsso.RedhatSSOConfig); ok {
			client := redhatsso.NewSSOClient(cfg)
			return &keycloakServiceProxy{
				accessTokenProvider: client,
				service: &redhatssoService{
					client: client,
				},
			}, nil
		} else {
			return nil, fmt.Errorf("invalid configuration for provider %s", provider)
		}
	}
	return nil, fmt.Errorf("invalid provider %s", provider)
}
