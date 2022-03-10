package sso

import (
	"context"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
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

func NewKeycloakServiceWithClient(client keycloak.KcClient) *masService {
	return &masService{
		kcClient: client,
	}
}

func NewKeycloakService(config *keycloak.KeycloakConfig, realmConfig *keycloak.KeycloakRealmConfig) *masService {
	client := keycloak.NewClient(config, realmConfig)
	return &masService{
		kcClient: client,
	}
}
