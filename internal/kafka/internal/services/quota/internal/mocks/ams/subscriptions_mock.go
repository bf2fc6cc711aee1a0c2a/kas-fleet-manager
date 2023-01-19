package ams

import (
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	v12 "github.com/openshift-online/ocm-sdk-go/authorizations/v1"
	"regexp"
)

func newSubscription(subscriptionID string, clusterID string, planID string) *v1.Subscription {
	subscription, _ := v1.NewSubscription().
		ID(subscriptionID).
		ClusterID(clusterID).
		Status(string(v12.SubscriptionStatusActive)).
		Plan(v1.NewPlan().ID(planID).Name(planID)).Build()
	return subscription
}

var subscriptions = map[string]*v1.Subscription{
	"subscription-1":  newSubscription("subscription-1", "cluster-1", "RHOSAK"),
	"subscription-2":  newSubscription("subscription-2", "cluster-2", "RHOSAK"),
	"subscription-3":  newSubscription("subscription-3", "cluster-3", "RHOSAKTrial"),
	"subscription-4":  newSubscription("subscription-4", "cluster-4", "RHOSAK"),
	"subscription-5":  newSubscription("subscription-5", "cluster-5", "RHOSAK"),
	"subscription-6":  newSubscription("subscription-6", "cluster-6", "RHOSAKTrial"),
	"subscription-7":  newSubscription("subscription-7", "cluster-7", "RHOSAKEval"),
	"subscription-8":  newSubscription("subscription-8", "cluster-8", "RHOSAKEval"),
	"subscription-9":  newSubscription("subscription-9", "cluster-9", "RHOSAKEval"),
	"subscription-10": newSubscription("subscription-10", "cluster-10", "RHOSAK"),
}

func FindSubscriptions(query string) []*v1.Subscription {
	r := regexp.MustCompile("^cluster_id='(?P<clusterid>.*)'$")

	switch true {
	case r.MatchString(query):
		clusterID := r.FindStringSubmatch(query)[1]

		for _, v := range subscriptions {
			if v.ClusterID() == clusterID {
				return []*v1.Subscription{v}
			}
		}
	default:
		panic("Query non supported by test:" + query)
	}
	return nil
}
