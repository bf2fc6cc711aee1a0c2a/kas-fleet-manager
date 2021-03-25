package services

import (
	"context"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

type AuthorizationService interface {
	CheckAccessReviewByKafkaID(ctx context.Context, action string, kafkaId string) (bool, *errors.ServiceError)
	ListAllowedKafkaInstance(ctx context.Context, productID string ) ([]string, *errors.ServiceError)
	IsQuotaReserved(ctx context.Context, productID string) (bool, *errors.ServiceError)
}

type authorizationService struct {
	connectionFactory *db.ConnectionFactory
	authzClient ocm.OCMAuthorization
}

var _ AuthorizationService = &authorizationService{}

//get/delete/create
func (a authorizationService) CheckAccessReviewByKafkaID(ctx context.Context, action string, kafkaId string) (bool, *errors.ServiceError) {
	isAllowed, err := a.authzClient.SelfAccessReview(ctx, action,"Cluster","", "" , kafkaId)
	if err != nil {
		return false, errors.GeneralError("%v", err)
	}
	return isAllowed, nil
}

func (a authorizationService) ListAllowedKafkaInstance(ctx context.Context, productID string) ([]string, *errors.ServiceError){
	query := fmt.Sprintf("plan_id is '%s' and status='%s'", productID, "Active")
	subs, err := a.authzClient.FindSubscriptions(ctx, query)
	if err != nil {
		return nil, errors.GeneralError("%v", err)
	}
	i := subs.Total()
	var listCluster = make([]string, i)
	subs.Items().Range(func(index int, item *amsv1.Subscription) bool {
		listCluster[index] = item.ClusterID()
		return true
	})
	return listCluster, nil
}


func (a authorizationService) IsQuotaReserved(ctx context.Context, productID string) (bool, *errors.ServiceError) {
	query := fmt.Sprintf("plan_id is '%s' and status='%s'", productID, "Active")
	subs, err :=  a.authzClient.FindSubscriptions(ctx, query)
	if err != nil {
		return false, errors.GeneralError("failed to check if quotas is available: %v", err)
	}
	if subs.Total() > 0 {
		return true, nil
	}
	return false, nil
}
