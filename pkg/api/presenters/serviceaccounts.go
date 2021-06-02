package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
)

func ConvertServiceAccountRequest(account openapi.ServiceAccountRequest) *api.ServiceAccountRequest {
	return &api.ServiceAccountRequest{
		Name:        account.Name,
		Description: account.Description,
	}
}

func PresentServiceAccount(account *api.ServiceAccount) *openapi.ServiceAccount {
	reference := PresentReference(account.ID, account)
	return &openapi.ServiceAccount{
		ClientId:               account.ClientID,
		ClientSecret:           account.ClientSecret,
		DeprecatedClientID:     account.ClientID,
		DeprecatedClientSecret: account.ClientSecret,
		Name:                   account.Name,
		Description:            account.Description,
		Owner:                  account.Owner,
		CreatedAt:              account.CreatedAt,
		Id:                     reference.Id,
		Kind:                   reference.Kind,
		Href:                   reference.Href,
	}
}

func PresentServiceAccountListItem(account *api.ServiceAccount) openapi.ServiceAccountListItem {
	ref := PresentReference(account.ID, account)
	return openapi.ServiceAccountListItem{
		Id:                 ref.Id,
		Kind:               ref.Kind,
		Href:               ref.Href,
		ClientId:           account.ClientID,
		Name:               account.Name,
		Owner:              account.Owner,
		Description:        account.Description,
		CreatedAt:          account.CreatedAt,
		DeprecatedClientID: account.ClientID,
	}
}
