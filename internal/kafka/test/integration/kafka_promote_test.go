package integration

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	kafkaMocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/onsi/gomega"
)

func TestKafkaPromote_Promote(t *testing.T) {
	// This test tests side-effects of the promotion. The several conditions
	// are tested on the corresponding unit tests.

	g := gomega.NewWithT(t)

	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, apiClient, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	var testKafka *dbapi.KafkaRequest
	var updatedKafka public.KafkaRequest

	db := h.DBFactory().New()
	testKafka = kafkaMocks.BuildKafkaRequest(kafkaMocks.WithPredefinedTestValues())
	testKafka.DesiredKafkaBillingModel = "eval"
	testKafka.ActualKafkaBillingModel = "eval"
	testKafka.Owner = account.Username()
	testKafka.OrganisationId = account.Organization().ExternalID()

	err := db.Create(testKafka).Error
	g.Expect(err).NotTo(gomega.HaveOccurred(), "failed to insert a ready kafka in the database")

	kafkaPromoteRequest := public.KafkaPromoteRequest{
		DesiredKafkaBillingModel:     "standard",
		DesiredMarketplace:           "",
		DesiredBillingCloudAccountId: "",
	}
	// Test a standard instance with kafka billing model can be promoted
	// to a standard instance with kafka billing model standard
	resp, err := apiClient.DefaultApi.PromoteKafka(ctx, testKafka.ID, true, kafkaPromoteRequest)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	updatedKafka, resp, err = apiClient.DefaultApi.GetKafkaById(ctx, testKafka.ID)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(updatedKafka.PromotionStatus).To(gomega.Equal(dbapi.KafkaPromotionStatusPromoting.String()))
	g.Expect(err).NotTo(gomega.HaveOccurred())

	// Test a standard instance with kafka billing model can be promoted
	// to a standard instance with kafka billing model standard even when
	// there is a previously failed promotion
	testKafka.ID = api.NewID()
	testKafka.Marketplace = "aws"
	testKafka.BillingCloudAccountId = "123456"
	testKafka.PromotionStatus = dbapi.KafkaPromotionStatusFailed
	testKafka.PromotionDetails = "test promotion failure details"
	err = db.Create(testKafka).Error
	g.Expect(err).NotTo(gomega.HaveOccurred(), "failed to insert a ready kafka in the database")

	resp, err = apiClient.DefaultApi.PromoteKafka(ctx, testKafka.ID, true, kafkaPromoteRequest)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	updatedKafka, resp, err = apiClient.DefaultApi.GetKafkaById(ctx, testKafka.ID)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(updatedKafka.PromotionStatus).To(gomega.Equal(dbapi.KafkaPromotionStatusPromoting.String()))
	g.Expect(updatedKafka.PromotionDetails).To(gomega.BeEmpty())
	// we also check that when promoting to standard we empty any potential
	// previous marketplace-related attributes
	g.Expect(updatedKafka.Marketplace).To(gomega.BeEmpty())
	g.Expect(updatedKafka.BillingCloudAccountId).To(gomega.BeEmpty())

	// Test a standard instance with kafka billing model can be promoted
	// to a standard instance with kafka billing model marketplace with its
	// corresponding marketplace and billing cloud account id
	testKafka.ID = api.NewID()
	testKafka.Marketplace = ""
	testKafka.BillingCloudAccountId = ""
	testKafka.PromotionStatus = dbapi.KafkaPromotionStatusNoPromotion
	testKafka.PromotionDetails = ""
	err = db.Create(testKafka).Error
	g.Expect(err).NotTo(gomega.HaveOccurred(), "failed to insert a ready kafka in the database")

	kafkaPromoteRequestToMarketplace := public.KafkaPromoteRequest{
		DesiredKafkaBillingModel:     "marketplace",
		DesiredMarketplace:           "aws",
		DesiredBillingCloudAccountId: "123456",
	}
	resp, err = apiClient.DefaultApi.PromoteKafka(ctx, testKafka.ID, true, kafkaPromoteRequestToMarketplace)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	updatedKafka, resp, err = apiClient.DefaultApi.GetKafkaById(ctx, testKafka.ID)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(updatedKafka.PromotionStatus).To(gomega.Equal(dbapi.KafkaPromotionStatusPromoting.String()))
	g.Expect(updatedKafka.PromotionDetails).To(gomega.BeEmpty())
	g.Expect(updatedKafka.Marketplace).To(gomega.Equal("aws"))
	g.Expect(updatedKafka.BillingCloudAccountId).To(gomega.Equal("123456"))
}
