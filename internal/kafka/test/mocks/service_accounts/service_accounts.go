package mocks

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	v1 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/managedkafkas.managedkafka.bf2.org/v1"
)

const (
	name                   = "test-name"
	description            = "test-description"
	id                     = "test-id"
	clientId               = "test-client-id"
	clientSecret           = "test-client-secret"
	owner                  = "test-owner"
	kindServiceAccount     = "ServiceAccount"
	baseUrlSsoProvider     = "https://base_url_sso_provider"
	tokenUrlSsoProvider    = "https://token_url_sso_provider"
	jwksSsoProvider        = "test-jwks"
	tokenIssuerSsoProvider = "test-issuer"
	principal              = "testPrincipal"
	password               = "testPassword"
)

func BuildServiceAccountRequest(modifyFn func(ServiceAccountRequest public.ServiceAccountRequest)) public.ServiceAccountRequest {
	serviceAccountRequest := public.ServiceAccountRequest{
		Name:        name,
		Description: description,
	}
	if modifyFn != nil {
		modifyFn(serviceAccountRequest)
	}
	return serviceAccountRequest
}

func BuildApiServiceAccountRequest(modifyFn func(ServiceAccountRequest api.ServiceAccountRequest)) *api.ServiceAccountRequest {
	serviceAccountRequest := api.ServiceAccountRequest{
		Name:        name,
		Description: description,
	}
	if modifyFn != nil {
		modifyFn(serviceAccountRequest)
	}
	return &serviceAccountRequest
}

func BuildServiceAccount(modifyFn func(ServiceAccount public.ServiceAccount)) *public.ServiceAccount {
	serviceAccount := public.ServiceAccount{
		Id:           id,
		Kind:         kindServiceAccount,
		Name:         name,
		Description:  description,
		ClientId:     clientId,
		ClientSecret: clientSecret,
		// Deprecated
		DeprecatedOwner: owner,
		CreatedBy:       owner,
		Href:            fmt.Sprintf("/api/kafkas_mgmt/v1/service_accounts/%s", id),
	}
	if modifyFn != nil {
		modifyFn(serviceAccount)
	}
	return &serviceAccount
}

func BuildApiServiceAccount(modifyFn func(ServiceAccount api.ServiceAccount)) *api.ServiceAccount {
	serviceAccount := api.ServiceAccount{
		ID:           id,
		Name:         name,
		Description:  description,
		ClientID:     clientId,
		ClientSecret: clientSecret,
		Owner:        owner,
	}
	if modifyFn != nil {
		modifyFn(serviceAccount)
	}
	return &serviceAccount
}

func BuildServiceAccountListItem(modifyFn func(ServiceAccount public.ServiceAccountListItem)) public.ServiceAccountListItem {
	serviceAccountListItem := public.ServiceAccountListItem{
		Id:              id,
		Name:            name,
		Href:            fmt.Sprintf("/api/kafkas_mgmt/v1/service_accounts/%s", id),
		Kind:            kindServiceAccount,
		Description:     description,
		ClientId:        clientId,
		DeprecatedOwner: owner,
		CreatedBy:       owner,
	}
	if modifyFn != nil {
		modifyFn(serviceAccountListItem)
	}
	return serviceAccountListItem
}

func BuildApiSsoProvier(modifyFn func(SsoProvider api.SsoProvider)) *api.SsoProvider {
	apiSsoProvider := api.SsoProvider{
		BaseUrl:     baseUrlSsoProvider,
		TokenUrl:    tokenUrlSsoProvider,
		Jwks:        jwksSsoProvider,
		ValidIssuer: tokenIssuerSsoProvider,
	}
	if modifyFn != nil {
		modifyFn(apiSsoProvider)
	}
	return &apiSsoProvider
}

func BuildSsoProvider(modifyFn func(SsoProvider public.SsoProvider)) public.SsoProvider {
	ssoProvider := public.SsoProvider{
		BaseUrl:     baseUrlSsoProvider,
		TokenUrl:    tokenUrlSsoProvider,
		Jwks:        jwksSsoProvider,
		ValidIssuer: tokenIssuerSsoProvider,
	}
	if modifyFn != nil {
		modifyFn(ssoProvider)
	}
	return ssoProvider
}

func GetServiceAccounts() []v1.ServiceAccount {
	return []v1.ServiceAccount{
		{
			Name:      name,
			Principal: principal,
			Password:  password,
		},
	}
}

func GetManagedKafkaAllOfSpecServiceAccounts() []private.ManagedKafkaAllOfSpecServiceAccounts {
	return []private.ManagedKafkaAllOfSpecServiceAccounts{
		{
			Name:      name,
			Principal: principal,
			Password:  password,
		},
	}
}
