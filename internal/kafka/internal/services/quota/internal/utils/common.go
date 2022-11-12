package utils

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

func getMarketplaceBillingModelForCloudProvider(cloudProvider string) (amsv1.BillingModel, error) {
	switch cloudProvider {
	case CloudProviderAWS:
		return amsv1.BillingModelMarketplaceAWS, nil
	case CloudProviderRHM:
		return amsv1.BillingModelMarketplace, nil
	case CloudProviderAzure:
		return amsv1.BillingModelMarketplaceAzure, nil
	}

	return "", errors.InvalidBillingAccount("unsupported cloud provider")
}

func getCloudAccounts(quotaCosts []*amsv1.QuotaCost) []*amsv1.CloudAccount {
	var accounts []*amsv1.CloudAccount
	for _, quotaCost := range quotaCosts {
		accounts = append(accounts, quotaCost.CloudAccounts()...)
	}
	return accounts
}

// check if it there is a related resource that supports the marketplace billing model and has quota
func hasSufficientMarketplaceQuota(quotaCosts []*amsv1.QuotaCost, consumedSize int) bool {
	for _, quotaCost := range quotaCosts {
		for _, rr := range quotaCost.RelatedResources() {
			if rr.BillingModel() == string(amsv1.BillingModelMarketplace) &&
				(rr.Cost() == 0 || quotaCost.Consumed()+consumedSize <= quotaCost.Allowed()) {
				return true
			}
		}
	}
	return false
}
