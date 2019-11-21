package ocm

import (
	"github.com/openshift-online/ocm-sdk-go"
	azv1 "github.com/openshift-online/ocm-sdk-go/authorizations/v1"
)

type OCMAuthorization interface {
	SelfAccessReview(ctx context.Context, action, resourceType, organizationID, subscriptionID, clusterID string) (allowed bool, err error)
	AccessReview(ctx context.Context, username, action, resourceType, organizationID, subscriptionID, clusterID string) (allowed bool, err error)
}

type authorization service

var _ OCMAuthorization = &authorization{}

func (a authorization) SelfAccessReview(ctx context.Context, action, resourceType, organizationID, subscriptionID, clusterID string) (allowed bool, err error) {
	con := a.client.Connection
	selfAccessReview := con.Authorizations().V1().SelfAccessReview()

	request, err := azv1.NewSelfAccessReviewRequest().
		Action(action).
		Resourcetype(resourceType).
		OrganizationID(organizationID).
		ClusterID(clusterID).
		SubscriptionID(subscriptionID).
		Build()
	if err != nil {
		return err
	}

	response, err := selfAccessReview.Post().
		Request(request).
		SendContext(ctx)
	if err != nil {
		return err
	}

	return response.Allowed(), nil
}

func (a authorization) AccessReview(ctx context.Context, username, action, resourceType, organizationID, subscriptionID, clusterID string) (allowed bool, err error) {
	con := a.client.Connection
	accessReview := con.Authorizations().V1().AccessReview()

	request, err := azv1.NewAccessReviewRequest().
		AccountUsername(username).
		Action(action).
		Resourcetype(resourceType).
		OrganizationID(organizationID).
		ClusterID(clusterID).
		SubscriptionID(subscriptionID).
		Build()
	if err != nil {
		return err
	}

	response, err := accessReview.Post().
		Request(request).
		SendContext(ctx)
	if err != nil {
		return err
	}

	return response.Allowed(), nil
}
