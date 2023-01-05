package ams

import (
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

func newReservedResource(resourceName string, billingModel string) *v1.ReservedResource {
	res, _ := v1.NewReservedResource().
		ResourceName(resourceName).
		ResourceType(resourceName).
		BillingModel(v1.BillingModel(billingModel)).
		Build()
	return res
}

var reservedResources = map[string][]*v1.ReservedResource{
	"subscription-1":  {newReservedResource("rhosak", "marketplace")},
	"subscription-2":  {newReservedResource("rhosak", "standard")},
	"subscription-3":  {newReservedResource("rhosak", "standard")},
	"subscription-4":  {newReservedResource("rhosak", "standard")},
	"subscription-5":  {newReservedResource("rhosak", "marketplace-aws")},
	"subscription-6":  {newReservedResource("rhosak", "standard")},
	"subscription-7":  {newReservedResource("rhosak", "standard")},
	"subscription-8":  {newReservedResource("rhosak", "standard")},
	"subscription-9":  {newReservedResource("rhosak", "standard")},
	"subscription-10": {newReservedResource("rhosak", "standard")},
}

func GetReservedResourcesBySubscriptionID(subscriptionID string) []*v1.ReservedResource {
	res, ok := reservedResources[subscriptionID]
	if ok {
		return res
	}
	return nil
}
