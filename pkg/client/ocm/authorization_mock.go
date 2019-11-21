package ocm

// authorizationMock returns allowed=true for every request
type authorizationMock service

var _ OCMAuthorization = &authorizationMock{}

func (a authorizationMock) SelfAccessReview(action, resourceType, organizationID, subscriptionID, clusterID string) (allowed bool, err error) {
	return true, nil
}

func (a authorizationMock) AccessReview(username, action, resourceType, organizationID, subscriptionID, clusterID string) (allowed bool, err error) {
	return true, nil
}
