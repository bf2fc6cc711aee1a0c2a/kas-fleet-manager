package ocm

import (
	"context"
	"fmt"
	sdkClient "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/authentication"
	amv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	azv1 "github.com/openshift-online/ocm-sdk-go/authorizations/v1"
)

const (
	ActiveStatus = "Active"
	Rhosak       = "RHOSAK"
	RhosakTrial  = "RHOSAKTrial"
)

//go:generate moq -out authorization_moq.go . OCMAuthorization:authorizationMock
type OCMAuthorization interface {
	SelfAccessReview(ctx context.Context, action, resourceType, organizationID, subscriptionID, clusterID string) (allowed bool, err error)
	AccessReview(ctx context.Context, username, action, resourceType, organizationID, subscriptionID, clusterID string) (allowed bool, err error)
	IsUserHasValidSubs(ctx context.Context) (bool, error)
	ListKafkaClusterWithActiveSubscription(ctx context.Context) ([]string, error)
	GetUserId(ctx context.Context) (string, error)
}

type authorization struct {
	client *Client //nolint
}

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

func (a authorization) IsUserHasValidSubs(ctx context.Context) (bool, error) {
	con, err := a.Connection(ctx)
	if err != nil {
		return false, err
	}
	query := fmt.Sprintf("(plan_id is '%s') or (plan_id is '%s')  and ( status='%s')", RhosakTrial, Rhosak, ActiveStatus)
	subResp, err := con.AccountsMgmt().V1().Subscriptions().List().Search(query).SendContext(ctx)
	if err != nil {
		return false, err
	}
	if subResp.Total() > 0 {
		return true, nil
	}
	return false, nil
}

func (a authorization) ListKafkaClusterWithActiveSubscription(ctx context.Context) ([]string, error) {
	query := fmt.Sprintf("(plan_id is '%s') or (plan_id is '%s') and ( status='%s')", RhosakTrial,Rhosak,ActiveStatus)
	con, err := a.Connection(ctx)
	if err != nil {
		return nil, err
	}
	subResp, err := con.AccountsMgmt().V1().Subscriptions().List().Search(query).SendContext(ctx)
	if err != nil {
		return nil, err
	}
	i := subResp.Total()
	var listCluster = make([]string, i)

	//allowed to list
	subResp.Items().Range(func(index int, item *amv1.Subscription) bool {
		listCluster[index] = item.ClusterID()
		return true
	})
	return listCluster, nil
}

func (a authorization) Connection(ctx context.Context) (*sdkClient.Connection, error) {
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

func (a authorization) GetUserId(ctx context.Context) (string, error){
	con, err := a.Connection(ctx)
	if err != nil {
		return "", err
	}
	accountResource, err := con.AccountsMgmt().V1().CurrentAccount().Get().SendContext(ctx)
	if err != nil {
		return "", err
	}
	userId, _ := accountResource.Body().GetID()
	return userId, nil
}
