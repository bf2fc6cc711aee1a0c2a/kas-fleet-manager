package integration

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/onsi/gomega"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

func TestOCMClientGetQuotaCosts(t *testing.T) {
	g := gomega.NewWithT(t)

	// create a mock ocm api server with custom response for the GET /{orgId}/quota_cost endpoint
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()

	mockRelatedResourceBuilderItem1 := amsv1.NewRelatedResource().ResourceName("related-resource-1").Product("product-1").ResourceType("type-1")
	mockQuotaCostBuilderItem1 := amsv1.NewQuotaCost().
		QuotaID("quota-1").
		Allowed(mocks.MockQuotaMaxAllowed).
		Consumed(mocks.MockQuotaConsumed).
		RelatedResources(mockRelatedResourceBuilderItem1)

	mockRelatedResourceBuilderItem2 := amsv1.NewRelatedResource().ResourceName("related-resource-2").Product("product-2").ResourceType("type-2")
	mockQuotaCostBuilderItem2 := amsv1.NewQuotaCost().
		QuotaID("quota-2").
		Allowed(mocks.MockQuotaMaxAllowed).
		Consumed(mocks.MockQuotaConsumed).
		RelatedResources(mockRelatedResourceBuilderItem2)

	quotaCostList, err := amsv1.NewQuotaCostList().Items(mockQuotaCostBuilderItem1, mockQuotaCostBuilderItem2).Build()
	g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to build mock quota list")
	ocmServerBuilder.SetGetOrganizationQuotaCost(quotaCostList, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	_, _, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	// skip the test if not running on ocm mock mode. We cannot test using an actual ocm env as the values for the quota costs
	// cannot be pre-determined for the assertions as they may change overtime.
	if test.TestServices.OCMConfig.MockMode != ocm.MockModeEmulateServer {
		t.SkipNow()
	}

	testOrgId := "test-org-id"

	// All quota cost from response of mock ocm server is returned if fetchRelatedResource is false and no filter is specified
	quotaCosts, err := test.TestServices.OCMClient.GetQuotaCosts(testOrgId, false, false)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(quotaCosts).To(gomega.HaveLen(2))

	// All quota cost from response of mock ocm server is returned if fetchRelatedResource is true and no filter is specified
	quotaCosts, err = test.TestServices.OCMClient.GetQuotaCosts(testOrgId, true, false)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(quotaCosts).To(gomega.HaveLen(2))

	// All quota cost from response of mock ocm server is returned if fetchRelatedResource is false even if a filter was specified
	quotaCosts, err = test.TestServices.OCMClient.GetQuotaCosts(testOrgId, false, false, ocm.QuotaCostRelatedResourceFilter{
		ResourceName: &[]string{"related-resource-1"}[0],
	})
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(quotaCosts).To(gomega.HaveLen(2))

	// Only the quota cost that matches the filter is returned if fetchRelatedResource is true and filter is specified
	quotaCosts, err = test.TestServices.OCMClient.GetQuotaCosts(testOrgId, true, false, ocm.QuotaCostRelatedResourceFilter{
		ResourceName: &[]string{"related-resource-1"}[0],
	})
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(quotaCosts).To(gomega.HaveLen(1))

	relatedResource, ok := quotaCosts[0].GetRelatedResources()
	g.Expect(ok).To(gomega.Equal(true))
	g.Expect(relatedResource).To(gomega.HaveLen(1))
	g.Expect(relatedResource[0].ResourceName()).To(gomega.Equal("related-resource-1"))
}
