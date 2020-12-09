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

func PresentServiceAccountRequest(account *api.ServiceAccount) *openapi.ServiceAccount {
	reference := PresentReference(account.ClientID, account)
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
