package ocm

import (
	"context"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

// authorizationMock returns allowed=true for every request
type authorizationMock service

func (a authorizationMock) FindSubscriptions(ctx context.Context, query string) (*amsv1.SubscriptionsListResponse, error) {
	return nil, nil
}

var _ OCMAuthorization = &authorizationMock{}

func (a authorizationMock) SelfAccessReview(ctx context.Context, action, resourceType, organizationID, subscriptionID, clusterID string) (allowed bool, err error) {
	return true, nil
}

func (a authorizationMock) AccessReview(ctx context.Context, username, action, resourceType, organizationID, subscriptionID, clusterID string) (allowed bool, err error) {
	return true, nil
}
