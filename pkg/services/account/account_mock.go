package account

import (
	"fmt"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"time"
)

const (
	mockOrgIDTemplate        = "mock-id-%d"
	mockExternalIDTemplate   = "mock-extid-%d"
	mockEbsAccountIDTemplate = "mock-ebs-%d"
	mockOrgNameTemplate      = "mock-org-%d"
)

// mock returns allowed=true for every request
type mock struct{}

var _ AccountService = &mock{}

func NewMockAccountService() AccountService {
	return &mock{}
}

func (a mock) SearchOrganizations(filter string) (*v1.OrganizationList, error) {
	return buildMockOrganizationList(10), nil
}

func (a mock) GetOrganization(filter string) (*v1.Organization, error) {
	orgs, _ := a.SearchOrganizations(filter)
	return orgs.Get(0), nil
}

func buildMockOrganizationList(count int) *v1.OrganizationList {
	var mockOrgs []*v1.OrganizationBuilder

	for i := 0; i < count; i++ {
		mockOrgs = append(mockOrgs,
			v1.NewOrganization().
				ID(fmt.Sprintf(mockOrgIDTemplate, i)).
				ExternalID(fmt.Sprintf(mockExternalIDTemplate, i)).
				EbsAccountID(fmt.Sprintf(mockEbsAccountIDTemplate, i)).
				CreatedAt(time.Now()).
				Name(fmt.Sprintf(mockOrgNameTemplate, i)).
				UpdatedAt(time.Now()))
	}

	ret, _ := v1.NewOrganizationList().Items(mockOrgs...).Build()

	return ret
}
