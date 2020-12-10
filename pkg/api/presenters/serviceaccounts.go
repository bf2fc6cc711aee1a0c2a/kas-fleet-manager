package presenters

import (
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
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
		ClientID:     account.ClientID,
		ClientSecret: account.ClientSecret,
		Name:         account.Name,
		Description:  account.Description,
		Id:           reference.Id,
		Kind:         reference.Kind,
		Href:         reference.Href,
	}
}

func PresentServiceAccountListItem(account *api.ServiceAccount) openapi.ServiceAccountListItem {
	ref := PresentReference(account.ID, account)
	return openapi.ServiceAccountListItem{
		Id:          ref.Id,
		Kind:        ref.Kind,
		Href:        ref.Href,
		ClientID:    account.ClientID,
		Name:        account.Name,
		Description: account.Description,
	}
}
