package authorization

import (
	"context"
)

// mock returns allowed=true for every request
type mock struct{}

var _ Authorization = &mock{}

func NewMockAuthorization() Authorization {
	return &mock{}
}

func (a mock) SelfAccessReview(ctx context.Context, action, resourceType, organizationID, subscriptionID, clusterID string) (allowed bool, err error) {
	return true, nil
}

func (a mock) AccessReview(ctx context.Context, username, action, resourceType, organizationID, subscriptionID, clusterID string) (allowed bool, err error) {
	return true, nil
}
