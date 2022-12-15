package utils

import amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"

type AMSQuotaConfigProvider interface {
	GetQuotaCostsForProduct(organizationID, resourceName, product string) ([]*amsv1.QuotaCost, error)
}

// It can happen that multiple resolvers will ask for the same quota costs. To avoid multiple
// calls to AMS, we cache the result here
type cachingQuotaConfigProvider struct {
	cache         map[string][]*amsv1.QuotaCost
	quotaProvider AMSQuotaConfigProvider
}

func (qr *cachingQuotaConfigProvider) GetQuotaCostsForProduct(organizationID, resourceName, product string) ([]*amsv1.QuotaCost, error) {
	key := organizationID + resourceName + product
	if quotaList, ok := qr.cache[key]; ok {
		return quotaList, nil
	}
	quotaList, err := qr.quotaProvider.GetQuotaCostsForProduct(organizationID, resourceName, product)
	if err != nil {
		return nil, err
	}
	qr.cache[key] = quotaList
	return quotaList, nil
}
