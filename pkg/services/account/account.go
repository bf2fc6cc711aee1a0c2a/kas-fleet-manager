package account

import (
	sdk "github.com/openshift-online/ocm-sdk-go"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

type AccountService interface {
	SearchOrganizations(filter string) (*v1.OrganizationList, error)
	GetOrganization(filter string) (*v1.Organization, error)
}

type accountService struct {
	connection *sdk.Connection
}

var _ AccountService = &accountService{}

func NewAccountService(connection *sdk.Connection) AccountService {
	return &accountService{
		connection,
	}
}

func (as *accountService) SearchOrganizations(filter string) (*v1.OrganizationList, error) {
	res, err := as.connection.AccountsMgmt().V1().Organizations().List().Search(filter).Send()
	if err != nil {
		return nil, err
	}

	return res.Items(), nil
}

func (as *accountService) GetOrganization(filter string) (*v1.Organization, error) {
	organizationList, err := as.SearchOrganizations(filter)
	if err != nil {
		return nil, err
	}

	if organizationList.Len() == 0 {
		return nil, nil
	}
	return organizationList.Get(0), nil
}
