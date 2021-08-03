package authorization

import (
	"context"
	"fmt"
	"net/http"

	sdkClient "github.com/openshift-online/ocm-sdk-go"

	azv1 "github.com/openshift-online/ocm-sdk-go/authorizations/v1"
)

type Authorization interface {
	SelfAccessReview(ctx context.Context, action, resourceType, organizationID, subscriptionID, clusterID string) (allowed bool, err error)
	AccessReview(ctx context.Context, username, action, resourceType, organizationID, subscriptionID, clusterID string) (allowed bool, err error)
	CheckUsernameValid(username string) (bool, error)
}

type authorization struct {
	client *sdkClient.Connection
}

var _ Authorization = &authorization{}

func NewOCMAuthorization(client *sdkClient.Connection) Authorization {
	return &authorization{
		client: client,
	}
}

func (a authorization) SelfAccessReview(ctx context.Context, action, resourceType, organizationID, subscriptionID, clusterID string) (allowed bool, err error) {
	con := a.client
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
	con := a.client
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

func (a authorization) CheckUsernameValid(username string) (bool, error) {
	resp, err := a.client.AccountsMgmt().V1().Accounts().List().
		Parameter("page", 1).
		Parameter("size", 1).
		Parameter("search", fmt.Sprintf("username = '%s'", username)).
		Send()

	return resp.Status() == http.StatusOK && resp.Size() > 0 && !resp.Items().Get(0).Banned(), err
}
