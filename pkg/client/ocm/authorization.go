package ocm

import (
	"context"
	"fmt"
	sdkClient "github.com/openshift-online/ocm-sdk-go"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/authentication"

	azv1 "github.com/openshift-online/ocm-sdk-go/authorizations/v1"
)

type OCMAuthorization interface {
	SelfAccessReview(ctx context.Context, action, resourceType, organizationID, subscriptionID, clusterID string) (allowed bool, err error)
	AccessReview(ctx context.Context, username, action, resourceType, organizationID, subscriptionID, clusterID string) (allowed bool, err error)
	FindSubscriptions(ctx context.Context, query string) (*amsv1.SubscriptionsListResponse, error)
}

type service struct {
	client *Client //nolint
}

type authorization service

var _ OCMAuthorization = &authorization{}

func (a authorization) SelfAccessReview(ctx context.Context, action, resourceType, organizationID, subscriptionID, clusterID string) (allowed bool, err error) {
	con := a.client.Connection
	selfAccessReview := con.Authorizations().V1().SelfAccessReview()

	request, err := azv1.NewSelfAccessReviewRequest().
		Action(action).
		ResourceType(resourceType).
		OrganizationID(organizationID).
		ClusterID(clusterID).
		SubscriptionID(subscriptionID).
		Build()
	if err != nil {
		return false, err
	}

	postResp, err := selfAccessReview.Post().
		Request(request).
		SendContext(ctx)
	if err != nil {
		return false, err
	}
	response, ok := postResp.GetResponse()
	if !ok {
		return false, fmt.Errorf("Empty response from authorization post request")
	}

	return response.Allowed(), nil
}

func (a authorization) AccessReview(ctx context.Context, username, action, resourceType, organizationID, subscriptionID, clusterID string) (allowed bool, err error) {
	con := a.client.Connection
	accessReview := con.Authorizations().V1().AccessReview()

	request, err := azv1.NewAccessReviewRequest().
		AccountUsername(username).
		Action(action).
		ResourceType(resourceType).
		OrganizationID(organizationID).
		ClusterID(clusterID).
		SubscriptionID(subscriptionID).
		Build()
	if err != nil {
		return false, err
	}

	postResp, err := accessReview.Post().
		Request(request).
		SendContext(ctx)
	if err != nil {
		return false, err
	}

	response, ok := postResp.GetResponse()
	if !ok {
		return false, fmt.Errorf("Empty response from authorization post request")
	}

	return response.Allowed(), nil
}

func (a authorization) FindSubscriptions(ctx context.Context,query string) (*amsv1.SubscriptionsListResponse, error) {
	con, err := a.connection(ctx)
	r, err := con.AccountsMgmt().V1().Subscriptions().List().Search(query).Fields("cluster_id").Send()
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (a authorization) connection(ctx context.Context) (*sdkClient.Connection, error) {
	token, err := authentication.TokenFromContext(ctx)
	if err != nil {
		return nil, err
	}
	con, err := a.client.NewConnWithUserToken(token.Raw)
	if err != nil {
		return nil, err
	}
	return  con, err
}