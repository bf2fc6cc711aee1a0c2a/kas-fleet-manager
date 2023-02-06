package integration

import (
	"database/sql"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/common"
	clusterMocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/clusters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kasfleetshardsync"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"testing"
	"time"

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

func TestKafkaPromote_Reconciler(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, func(reconcilerConfig *workers.ReconcilerConfig) {
		// set the interval to 1 second
		reconcilerConfig.ReconcilerRepeatInterval = 1 * time.Second
	})
	defer teardown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	cluster := clusterMocks.BuildCluster(func(cluster *api.Cluster) {
		cluster.Meta = api.Meta{
			ID: api.NewID(),
		}
		cluster.ProviderType = api.ClusterProviderStandalone
		cluster.SupportedInstanceType = types.STANDARD.String()
		cluster.ClientID = "some-client-id"
		cluster.ClientSecret = "some-client-secret"
		cluster.ClusterID = "some-cluster-id"
		cluster.CloudProvider = mocks.MockCluster.CloudProvider().ID()
		cluster.Region = mocks.MockCluster.Region().ID()
		cluster.ProviderSpec = api.JSON{}
		cluster.ClusterSpec = api.JSON{}
		cluster.ClusterType = api.ManagedDataPlaneClusterType.String()
		cluster.Status = api.ClusterProvisioning
	})

	g := gomega.NewWithT(t)
	db := test.TestServices.DBFactory.New()
	err := db.Create(cluster).Error
	g.Expect(err).NotTo(gomega.HaveOccurred())

	tests := []struct {
		name                string
		actualBillingModel  string
		desiredBillingModel string
		timeout             time.Duration
		wantErr             bool
	}{
		{
			name:                "Test promote eval to standard",
			actualBillingModel:  "eval",
			desiredBillingModel: "standard",
		},
		{
			name:                "Test promote eval to marketplace",
			actualBillingModel:  "eval",
			desiredBillingModel: "marketplace",
		},
		{
			name:                "Test promote eval to badbm",
			actualBillingModel:  "eval",
			desiredBillingModel: "badbm",
			timeout:             15,
			wantErr:             true,
		},
	}

	for _, t1 := range tests {
		tt := t1
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			timeout := tt.timeout
			if timeout == 0 {
				timeout = 60
			}
			kafkas := []*dbapi.KafkaRequest{
				kafkaMocks.BuildKafkaRequest(
					kafkaMocks.WithPredefinedTestValues(),
					kafkaMocks.With(kafkaMocks.NAME, "dummy-kafka"),
					kafkaMocks.With(kafkaMocks.OWNER, "username"),
					kafkaMocks.With(kafkaMocks.CLUSTER_ID, cluster.ClusterID),
					kafkaMocks.With(kafkaMocks.STATUS, constants.KafkaRequestStatusAccepted.String()),
					kafkaMocks.With(kafkaMocks.BOOTSTRAP_SERVER_HOST, ""),
					kafkaMocks.With(kafkaMocks.DESIRED_KAFKA_BILLING_MODEL, tt.desiredBillingModel),
					kafkaMocks.With(kafkaMocks.ACTUAL_KAFKA_BILLING_MODEL, tt.actualBillingModel),
					kafkaMocks.With(kafkaMocks.PROMOTION_STATUS, "promoting"),
					kafkaMocks.WithExpiresAt(sql.NullTime{
						Time:  time.Now().AddDate(0, 0, 5),
						Valid: true,
					}),
				),
			}

			db := test.TestServices.DBFactory.New()
			dbErr := db.Create(&kafkas).Error
			g.Expect(dbErr).NotTo(gomega.HaveOccurred())

			var promotedKafka dbapi.KafkaRequest

			err1 := common.NewPollerBuilder(test.TestServices.DBFactory).
				IntervalAndTimeout(1*time.Second, timeout*time.Second).
				RetryLogMessage("Waiting for kafka to be promoted to standard").
				RetryLogFunction(func(retry int, maxRetry int) string {
					if promotedKafka.ID == "" {
						return fmt.Sprintf("Waiting for kafka to be promoted to %s", tt.desiredBillingModel)
					} else {
						return fmt.Sprintf("Waiting for kafka (%s) to be promoted to %s. Current status (actual, desired): %s,%s", promotedKafka.ID, tt.desiredBillingModel, promotedKafka.ActualKafkaBillingModel, promotedKafka.DesiredKafkaBillingModel)
					}
				}).
				OnRetry(func(attempt int, maxRetries int) (bool, error) {
					err := db.Where("id = ?", kafkas[0].ID).Find(&promotedKafka).Error
					if err != nil {
						return true, err
					}
					return promotedKafka.ActualKafkaBillingModel == tt.desiredBillingModel && promotedKafka.DesiredKafkaBillingModel == promotedKafka.ActualKafkaBillingModel, nil
				}).Build().Poll()
			g.Expect(err1 != nil).To(gomega.Equal(tt.wantErr), "Unexpected error occurred %v", err1)
			if !tt.wantErr {
				g.Expect(promotedKafka.ActualKafkaBillingModel).To(gomega.Equal(tt.desiredBillingModel))
				g.Expect(promotedKafka.DesiredKafkaBillingModel).To(gomega.Equal(tt.desiredBillingModel))
				g.Expect(promotedKafka.PromotionStatus).To(gomega.BeEmpty())
				g.Expect(promotedKafka.PromotionDetails).To(gomega.BeEmpty())
				g.Expect(promotedKafka.ExpiresAt).To(gomega.Equal(sql.NullTime{
					Time:  time.Time{},
					Valid: false,
				}))
			} else {
				g.Expect(promotedKafka.PromotionStatus).To(gomega.Equal(dbapi.KafkaPromotionStatusFailed))
				g.Expect(promotedKafka.PromotionDetails).ToNot(gomega.BeEmpty())
			}
		})
	}
}
