package integration

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/common"
	mockclusters "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/clusters"
	mockkafka "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kasfleetshardsync"
	mocksupportedinstancetypes "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/supported_instance_types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/quota_management"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"

	"github.com/onsi/gomega"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

func TestKafka_QuotaManagementListExpirationBasedOnQuota(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	bmNoExpirationID := "bm-no-expiration"
	bmNotExpiredID := "bm-not-expired"
	bmExpiredWithGracePeriodID := "bm-expired-with-grace-period"
	bmExpiredWithoutGracePeriodID := "bm-expired-without-grace-period"
	bmNoLongerEntitledID := "bm-no-longer-entitled"

	dummyOrgID := "dummy-org-1"
	dummyOrgUser := "org-1-user"
	dummyUser1 := "dummy-user-2"

	defaultInstanceSize := mocksupportedinstancetypes.BuildKafkaInstanceSize(
		mocksupportedinstancetypes.WithLifespanSeconds(nil),
	)

	h, client, tearDown := test.NewKafkaHelperWithHooks(t, ocmServer, func(kafkaConfig *config.KafkaConfig, quotaManagementListConfig *quota_management.QuotaManagementListConfig) {
		// set quota management type to 'quota-mangement-list'
		kafkaConfig.Quota.Type = api.QuotaManagementListQuotaType.String()

		// configure supported instance types with dummy kafka billing models
		kafkaConfig.SupportedInstanceTypes = &config.KafkaSupportedInstanceTypesConfig{
			Configuration: config.SupportedKafkaInstanceTypesConfig{
				SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
					{
						Id:          types.STANDARD.String(),
						DisplayName: types.STANDARD.String(),
						Sizes: []config.KafkaInstanceSize{
							*defaultInstanceSize,
						},
						SupportedBillingModels: []config.KafkaBillingModel{
							{
								ID:               bmNoExpirationID,
								AMSResource:      ocm.RHOSAKResourceName,
								AMSProduct:       string(ocm.RHOSAKProduct),
								AMSBillingModels: []string{string(amsv1.BillingModelStandard)},
							},
							{
								ID:               bmNotExpiredID,
								AMSResource:      ocm.RHOSAKResourceName,
								AMSProduct:       string(ocm.RHOSAKProduct),
								AMSBillingModels: []string{string(amsv1.BillingModelStandard)},
							},
							{
								ID:               bmExpiredWithoutGracePeriodID,
								AMSResource:      ocm.RHOSAKResourceName,
								AMSProduct:       string(ocm.RHOSAKProduct),
								AMSBillingModels: []string{string(amsv1.BillingModelStandard)},
							},
							{
								ID:               bmExpiredWithGracePeriodID,
								AMSResource:      ocm.RHOSAKResourceName,
								AMSProduct:       string(ocm.RHOSAKProduct),
								AMSBillingModels: []string{string(amsv1.BillingModelStandard)},
								GracePeriodDays:  10,
							},
							{
								ID:               bmNoLongerEntitledID,
								AMSResource:      ocm.RHOSAKResourceName,
								AMSProduct:       string(ocm.RHOSAKProduct),
								AMSBillingModels: []string{string(amsv1.BillingModelStandard)},
							},
						},
					},
					{
						Id:          types.DEVELOPER.String(),
						DisplayName: types.DEVELOPER.String(),
						Sizes: []config.KafkaInstanceSize{
							*mocksupportedinstancetypes.BuildKafkaInstanceSize(),
						},
						SupportedBillingModels: []config.KafkaBillingModel{
							{
								ID:               types.STANDARD.String(),
								AMSResource:      ocm.RHOSAKResourceName,
								AMSProduct:       string(ocm.RHOSAKTrialProduct),
								AMSBillingModels: []string{string(amsv1.BillingModelStandard)},
							},
						},
					},
				},
			},
		}

		// ensures quota entitlement status is applied correctly, if set to false quota will always be active
		quotaManagementListConfig.EnableInstanceLimitControl = true

		// register dummy organisations and users to the quota management list configuration
		quotaManagementListConfig.QuotaList = quota_management.RegisteredUsersListConfiguration{
			Organisations: quota_management.OrganisationList{
				{
					Id:      dummyOrgID,
					AnyUser: true,
					GrantedQuota: quota_management.QuotaList{
						{
							InstanceTypeID: types.STANDARD.String(),
							KafkaBillingModels: quota_management.BillingModelList{
								{
									Id: bmNoExpirationID,
								},
								{
									Id:             bmNotExpiredID,
									ExpirationDate: &[]quota_management.ExpirationDate{quota_management.ExpirationDate(time.Now().AddDate(0, 0, 10))}[0],
								},
								{
									Id:             bmExpiredWithoutGracePeriodID,
									ExpirationDate: &[]quota_management.ExpirationDate{quota_management.ExpirationDate(time.Now().AddDate(0, 0, -1))}[0],
								},
								{
									Id:             bmExpiredWithGracePeriodID,
									ExpirationDate: &[]quota_management.ExpirationDate{quota_management.ExpirationDate(time.Now().AddDate(0, 0, -1))}[0],
								},
							},
						},
					},
				},
			},
			ServiceAccounts: quota_management.AccountList{
				{
					Username: dummyUser1,
					GrantedQuota: quota_management.QuotaList{
						{
							InstanceTypeID: types.STANDARD.String(),
							KafkaBillingModels: quota_management.BillingModelList{
								{
									Id: bmNoExpirationID,
								},
								{
									Id:             bmNotExpiredID,
									ExpirationDate: &[]quota_management.ExpirationDate{quota_management.ExpirationDate(time.Now().AddDate(0, 0, 10))}[0],
								},
								{
									Id:             bmExpiredWithoutGracePeriodID,
									ExpirationDate: &[]quota_management.ExpirationDate{quota_management.ExpirationDate(time.Now().AddDate(0, 0, -1))}[0],
								},
								{
									Id:             bmExpiredWithGracePeriodID,
									ExpirationDate: &[]quota_management.ExpirationDate{quota_management.ExpirationDate(time.Now().AddDate(0, 0, -1))}[0],
								},
							},
						},
					},
				},
			},
		}
	})
	defer tearDown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	// set up dummy data plane cluster that accepts both standard and developer clusters
	testCluster := mockclusters.BuildCluster(func(cluster *api.Cluster) {
		cluster.Meta = api.Meta{
			ID: api.NewID(),
		}
		cluster.ProviderType = api.ClusterProviderStandalone // ensures no errors will occur due to it not being available on ocm
		cluster.SupportedInstanceType = api.AllInstanceTypeSupport.String()
		cluster.ClientID = "some-client-id"
		cluster.ClientSecret = "some-client-secret"
		cluster.ClusterID = "test-cluster"
		cluster.CloudProvider = mocks.MockCloudProviderID
		cluster.Region = mocks.MockCloudRegionID
		cluster.Status = api.ClusterReady
		cluster.ProviderSpec = api.JSON{}
		cluster.ClusterSpec = api.JSON{}
		cluster.AvailableStrimziVersions = api.JSON(mockclusters.AvailableStrimziVersions)
	})
	db := h.DBFactory().New()
	err := db.Create(testCluster).Error
	g.Expect(err).NotTo(gomega.HaveOccurred(), "failed to create dummy data plane cluster")

	kafkasToRemain := []*dbapi.KafkaRequest{
		// kafkas to remain as billing model is not expired
		mockkafka.BuildKafkaRequest(
			mockkafka.WithPredefinedTestValues(),
			mockkafka.With(mockkafka.NAME, "kafka-to-remain-1"),
			mockkafka.With(mockkafka.ORGANISATION_ID, dummyOrgID),
			mockkafka.With(mockkafka.OWNER, dummyOrgUser),
			mockkafka.With(mockkafka.STATUS, constants.KafkaRequestStatusReady.String()),
			mockkafka.With(mockkafka.CANARY_SERVICE_ACCOUNT_CLIENT_ID, "dummy-sa-client-id"),
			mockkafka.With(mockkafka.CANARY_SERVICE_ACCOUNT_CLIENT_SECRET, "dummy-sa-client-secret"),
			mockkafka.With(mockkafka.INSTANCE_TYPE, types.STANDARD.String()),
			mockkafka.With(mockkafka.ACTUAL_KAFKA_BILLING_MODEL, bmNoExpirationID),
			mockkafka.With(mockkafka.DESIRED_KAFKA_BILLING_MODEL, bmNoExpirationID),
			mockkafka.With(mockkafka.CLUSTER_ID, testCluster.ClusterID),
			// simulate kafka with a null expires_at value
			mockkafka.WithExpiresAt(sql.NullTime{}),
		),
		mockkafka.BuildKafkaRequest(
			mockkafka.WithPredefinedTestValues(),
			mockkafka.With(mockkafka.NAME, "kafka-to-remain-2"),
			mockkafka.With(mockkafka.ORGANISATION_ID, dummyOrgID),
			mockkafka.With(mockkafka.OWNER, dummyOrgUser),
			mockkafka.With(mockkafka.STATUS, constants.KafkaRequestStatusReady.String()),
			mockkafka.With(mockkafka.CANARY_SERVICE_ACCOUNT_CLIENT_ID, "dummy-sa-client-id"),
			mockkafka.With(mockkafka.CANARY_SERVICE_ACCOUNT_CLIENT_SECRET, "dummy-sa-client-secret"),
			mockkafka.With(mockkafka.INSTANCE_TYPE, types.STANDARD.String()),
			mockkafka.With(mockkafka.ACTUAL_KAFKA_BILLING_MODEL, bmNoExpirationID),
			mockkafka.With(mockkafka.DESIRED_KAFKA_BILLING_MODEL, bmNoExpirationID),
			mockkafka.With(mockkafka.CLUSTER_ID, testCluster.ClusterID),
			// after reconcile, expires_at should be set to null as quota is not expired, no expiration date set in granted quota
			mockkafka.WithExpiresAt(sql.NullTime{Time: time.Now().AddDate(0, 0, 1), Valid: true}),
		),
		mockkafka.BuildKafkaRequest(
			mockkafka.WithPredefinedTestValues(),
			mockkafka.With(mockkafka.NAME, "kafka-to-remain-3"),
			mockkafka.With(mockkafka.ORGANISATION_ID, dummyOrgID),
			mockkafka.With(mockkafka.OWNER, dummyOrgUser),
			mockkafka.With(mockkafka.STATUS, constants.KafkaRequestStatusReady.String()),
			mockkafka.With(mockkafka.CANARY_SERVICE_ACCOUNT_CLIENT_ID, "dummy-sa-client-id"),
			mockkafka.With(mockkafka.CANARY_SERVICE_ACCOUNT_CLIENT_SECRET, "dummy-sa-client-secret"),
			mockkafka.With(mockkafka.INSTANCE_TYPE, types.STANDARD.String()),
			mockkafka.With(mockkafka.ACTUAL_KAFKA_BILLING_MODEL, bmNotExpiredID),
			mockkafka.With(mockkafka.DESIRED_KAFKA_BILLING_MODEL, bmNotExpiredID),
			mockkafka.With(mockkafka.CLUSTER_ID, testCluster.ClusterID),
			// after reconcile, expires_at should be set to null as quota is not expired based on the expiration date defined in granted quota
			mockkafka.WithExpiresAt(sql.NullTime{Time: time.Now().AddDate(0, 0, 1), Valid: true}),
		),
		// kafka should be suspended but not deleted as billing model is expired but has a grace period allowance
		mockkafka.BuildKafkaRequest(
			mockkafka.WithPredefinedTestValues(),
			// mockkafka.With(mockkafka.ID, "kafka-to-suspend-for-org-1"),
			mockkafka.With(mockkafka.NAME, "kafka-to-suspend-1"),
			mockkafka.With(mockkafka.ORGANISATION_ID, dummyOrgID),
			mockkafka.With(mockkafka.OWNER, dummyOrgUser),
			mockkafka.With(mockkafka.STATUS, constants.KafkaRequestStatusReady.String()),
			mockkafka.With(mockkafka.CANARY_SERVICE_ACCOUNT_CLIENT_ID, "dummy-sa-client-id"),
			mockkafka.With(mockkafka.CANARY_SERVICE_ACCOUNT_CLIENT_SECRET, "dummy-sa-client-secret"),
			mockkafka.With(mockkafka.INSTANCE_TYPE, types.STANDARD.String()),
			mockkafka.With(mockkafka.ACTUAL_KAFKA_BILLING_MODEL, bmExpiredWithGracePeriodID),
			mockkafka.With(mockkafka.DESIRED_KAFKA_BILLING_MODEL, bmExpiredWithGracePeriodID),
			mockkafka.With(mockkafka.CLUSTER_ID, testCluster.ClusterID),
			mockkafka.WithExpiresAt(sql.NullTime{}),
		),
	}

	kafkasToExpire := []*dbapi.KafkaRequest{
		// should expire and deleted straight away as the given quota is expired and has no grace period allowance
		mockkafka.BuildKafkaRequest(
			mockkafka.WithPredefinedTestValues(),
			mockkafka.With(mockkafka.NAME, "kafka-to-expire-1"),
			mockkafka.With(mockkafka.ORGANISATION_ID, dummyOrgID),
			mockkafka.With(mockkafka.OWNER, dummyOrgUser),
			mockkafka.With(mockkafka.STATUS, constants.KafkaRequestStatusReady.String()),
			mockkafka.With(mockkafka.CANARY_SERVICE_ACCOUNT_CLIENT_ID, "dummy-sa-client-id"),
			mockkafka.With(mockkafka.CANARY_SERVICE_ACCOUNT_CLIENT_SECRET, "dummy-sa-client-secret"),
			mockkafka.With(mockkafka.INSTANCE_TYPE, types.STANDARD.String()),
			mockkafka.With(mockkafka.ACTUAL_KAFKA_BILLING_MODEL, bmExpiredWithoutGracePeriodID),
			mockkafka.With(mockkafka.DESIRED_KAFKA_BILLING_MODEL, bmExpiredWithoutGracePeriodID),
			mockkafka.With(mockkafka.CLUSTER_ID, testCluster.ClusterID),
			// simulate kafka with a null expires_at value
			mockkafka.WithExpiresAt(sql.NullTime{}),
		),
		// should expire and deleted straight away as the given quota is expired and has no grace period allowance
		mockkafka.BuildKafkaRequest(
			mockkafka.WithPredefinedTestValues(),
			mockkafka.With(mockkafka.NAME, "kafka-to-expire-2"),
			mockkafka.With(mockkafka.ORGANISATION_ID, dummyOrgID),
			mockkafka.With(mockkafka.OWNER, dummyOrgUser),
			mockkafka.With(mockkafka.STATUS, constants.KafkaRequestStatusReady.String()),
			mockkafka.With(mockkafka.CANARY_SERVICE_ACCOUNT_CLIENT_ID, "dummy-sa-client-id"),
			mockkafka.With(mockkafka.CANARY_SERVICE_ACCOUNT_CLIENT_SECRET, "dummy-sa-client-secret"),
			mockkafka.With(mockkafka.INSTANCE_TYPE, types.STANDARD.String()),
			mockkafka.With(mockkafka.ACTUAL_KAFKA_BILLING_MODEL, bmExpiredWithoutGracePeriodID),
			mockkafka.With(mockkafka.DESIRED_KAFKA_BILLING_MODEL, bmExpiredWithoutGracePeriodID),
			mockkafka.With(mockkafka.CLUSTER_ID, testCluster.ClusterID),
			// simulate kafka with a zero time.Time expires_at value
			mockkafka.WithExpiresAt(sql.NullTime{Time: time.Time{}, Valid: true}),
		),
		// should expire as quota for given billing model is no longer entitled to the user
		mockkafka.BuildKafkaRequest(
			mockkafka.WithPredefinedTestValues(),
			mockkafka.With(mockkafka.NAME, "kafka-to-expire-3"),
			mockkafka.With(mockkafka.ORGANISATION_ID, dummyOrgID),
			mockkafka.With(mockkafka.OWNER, dummyOrgUser),
			mockkafka.With(mockkafka.STATUS, constants.KafkaRequestStatusReady.String()),
			mockkafka.With(mockkafka.CANARY_SERVICE_ACCOUNT_CLIENT_ID, "dummy-sa-client-id"),
			mockkafka.With(mockkafka.CANARY_SERVICE_ACCOUNT_CLIENT_SECRET, "dummy-sa-client-secret"),
			mockkafka.With(mockkafka.INSTANCE_TYPE, types.STANDARD.String()),
			mockkafka.With(mockkafka.ACTUAL_KAFKA_BILLING_MODEL, bmNoLongerEntitledID),
			mockkafka.With(mockkafka.DESIRED_KAFKA_BILLING_MODEL, bmNoLongerEntitledID),
			mockkafka.With(mockkafka.CLUSTER_ID, testCluster.ClusterID),
			mockkafka.WithExpiresAt(sql.NullTime{}),
		),
	}

	// create dummy kafkas
	var kafkas []*dbapi.KafkaRequest
	kafkas = append(kafkas, kafkasToRemain...)
	kafkas = append(kafkas, kafkasToExpire...)
	err = db.Create(&kafkas).Error
	g.Expect(err).NotTo(gomega.HaveOccurred(), "failed to create dummy kafkas")

	// test cases for verifying expiration logic for users registered by organisation
	orgAccount := h.NewAccount(dummyOrgUser, "", "", dummyOrgID)
	orgCtx := h.NewAuthenticatedContext(orgAccount, nil)

	kafkaDeletionErr := common.WaitForNumberOfKafkaToBeGivenCount(orgCtx, test.TestServices.DBFactory, client, int32(len(kafkasToRemain)), func(builder common.PollerBuilder) common.PollerBuilder {
		return builder.DumpDB("kafka_requests", "", "id", "name", "status", "actual_kafka_billing_model", "expires_at")
	})
	g.Expect(kafkaDeletionErr).NotTo(gomega.HaveOccurred(), "Error waiting for kafka deletion: %v", kafkaDeletionErr)

	kafkaRequestList, _, err := client.DefaultApi.GetKafkas(orgCtx, &public.GetKafkasOpts{})
	g.Expect(err).ToNot(gomega.HaveOccurred())

	// verify that kafkas with an expired quota and grace period is suspended
	// other kafkas should not have an expires_at value set
	for _, k := range kafkaRequestList.Items {
		if strings.HasPrefix(k.Name, "kafka-to-suspend-") {
			_, err := common.WaitForKafkaToReachStatus(orgCtx, test.TestServices.DBFactory, client, k.Id, constants.KafkaRequestStatusSuspended)
			g.Expect(err).NotTo(gomega.HaveOccurred(), "error waiting for kafka %q to be suspended", k.Id)
			g.Expect(k.ExpiresAt).NotTo(gomega.BeNil())
			continue
		}
		g.Expect(k.ExpiresAt).To(gomega.BeNil())
	}

	// test cases for verifying expiration logic for users registered as service account
	// create dummy kafkas for user registered as a service account in the quota management list
	for i, k := range kafkas {
		k.ID = fmt.Sprintf("dummy-kafka-for-user-%d", i)
		k.OrganisationId = "unregistered-org"
		k.Owner = dummyUser1
	}

	err = db.Create(&kafkas).Error
	g.Expect(err).NotTo(gomega.HaveOccurred(), "failed to create dummy kafkas")

	saAccount := h.NewAccount(dummyUser1, "", "", "unregistered-org")
	saCtx := h.NewAuthenticatedContext(saAccount, nil)

	kafkaDeletionErr = common.WaitForNumberOfKafkaToBeGivenCount(saCtx, test.TestServices.DBFactory, client, int32(len(kafkasToRemain)), func(builder common.PollerBuilder) common.PollerBuilder {
		return builder.DumpDB("kafka_requests", "", "id", "name", "status", "actual_kafka_billing_model", "expires_at")
	})
	g.Expect(kafkaDeletionErr).NotTo(gomega.HaveOccurred(), "Error waiting for kafka deletion: %v", kafkaDeletionErr)

	kafkaRequestList, _, err = client.DefaultApi.GetKafkas(saCtx, &public.GetKafkasOpts{})
	g.Expect(err).ToNot(gomega.HaveOccurred())

	// verify that kafkas with an expired quota and grace period is suspended
	// other kafkas should not have an expires_at value set
	for _, k := range kafkaRequestList.Items {
		if strings.HasPrefix(k.Name, "kafka-to-suspend-") {
			_, err := common.WaitForKafkaToReachStatus(saCtx, test.TestServices.DBFactory, client, k.Id, constants.KafkaRequestStatusSuspended)
			g.Expect(err).NotTo(gomega.HaveOccurred(), "error waiting for kafka %q to be suspended", k.Id)
			g.Expect(k.ExpiresAt).NotTo(gomega.BeNil())
			continue
		}
		g.Expect(k.ExpiresAt).To(gomega.BeNil())
	}
}

func TestKafka_ExpirationBasedOnLifespanSeconds(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	longLifespanSeconds := 178000
	shortLifespanSeconds := 2

	h, client, tearDown := test.NewKafkaHelperWithHooks(t, ocmServer, func(kafkaConfig *config.KafkaConfig, quotaManagementListConfig *quota_management.QuotaManagementListConfig) {
		// configure supported instance types with dummy kafka billing models
		kafkaConfig.SupportedInstanceTypes = &config.KafkaSupportedInstanceTypesConfig{
			Configuration: config.SupportedKafkaInstanceTypesConfig{
				SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
					{
						Id:          types.STANDARD.String(),
						DisplayName: types.STANDARD.String(),
						Sizes: []config.KafkaInstanceSize{
							*mocksupportedinstancetypes.BuildKafkaInstanceSize(
								mocksupportedinstancetypes.WithLifespanSeconds(&longLifespanSeconds),
							),
						},
						SupportedBillingModels: []config.KafkaBillingModel{
							{
								ID:               types.STANDARD.String(),
								AMSResource:      ocm.RHOSAKResourceName,
								AMSProduct:       string(ocm.RHOSAKProduct),
								AMSBillingModels: []string{string(amsv1.BillingModelStandard)},
							},
						},
					},
					{
						Id:          types.DEVELOPER.String(),
						DisplayName: types.DEVELOPER.String(),
						Sizes: []config.KafkaInstanceSize{
							*mocksupportedinstancetypes.BuildKafkaInstanceSize(
								mocksupportedinstancetypes.WithLifespanSeconds(&shortLifespanSeconds),
							),
						},
						SupportedBillingModels: []config.KafkaBillingModel{
							{
								ID:               types.STANDARD.String(),
								AMSResource:      ocm.RHOSAKResourceName,
								AMSProduct:       string(ocm.RHOSAKTrialProduct),
								AMSBillingModels: []string{string(amsv1.BillingModelStandard)},
							},
						},
					},
				},
			},
		}
	})
	defer tearDown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	// set up dummy data plane cluster that accepts both standard and developer clusters
	testCluster := mockclusters.BuildCluster(func(cluster *api.Cluster) {
		cluster.Meta = api.Meta{
			ID: api.NewID(),
		}
		cluster.ProviderType = api.ClusterProviderStandalone // ensures no errors will occur due to it not being available on ocm
		cluster.SupportedInstanceType = api.AllInstanceTypeSupport.String()
		cluster.ClientID = "some-client-id"
		cluster.ClientSecret = "some-client-secret"
		cluster.ClusterID = "test-cluster"
		cluster.CloudProvider = mocks.MockCloudProviderID
		cluster.Region = mocks.MockCloudRegionID
		cluster.Status = api.ClusterReady
		cluster.ProviderSpec = api.JSON{}
		cluster.ClusterSpec = api.JSON{}
		cluster.AvailableStrimziVersions = api.JSON(mockclusters.AvailableStrimziVersions)
	})
	db := h.DBFactory().New()
	err := db.Create(testCluster).Error
	g.Expect(err).NotTo(gomega.HaveOccurred(), "failed to create dummy data plane cluster")

	// ensure kafka request with long lifespan does not get deleted
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	result, _, err := client.DefaultApi.CreateKafka(ctx, true, public.KafkaRequestPayload{
		Name: "kafka-with-long-lifespan",
		Plan: fmt.Sprintf("%s.x1", types.STANDARD), // has a long lifespan of 2d
	})
	g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to create kafka with long lifespan")
	expectedExpirationDate := result.CreatedAt.Add(time.Duration(longLifespanSeconds) * time.Second).Format(time.RFC1123Z)
	g.Expect(result.ExpiresAt.Format(time.RFC1123Z)).To(gomega.Equal(expectedExpirationDate), "expected expires_at for kafka with long lifespan does not match actual")

	// wait for kafka to be ready and ensure the expiration date does not get updated
	kafka, err := common.WaitForKafkaToReachStatus(ctx, test.TestServices.DBFactory, client, result.Id, constants.KafkaRequestStatusReady)
	g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to wait for kafka with long lifespan to reach status %q", constants.KafkaRequestStatusReady)
	g.Expect(kafka.ExpiresAt.Format(time.RFC1123Z)).To(gomega.Equal(expectedExpirationDate), "expected expires_at for kafka with long lifespan does not match actual")

	// ensure kafka request with short lifespan gets deleted
	account = h.NewAccount("dummy-user-1", "", "", "dummy-org")
	ctx = h.NewAuthenticatedContext(account, nil)
	result, _, err = client.DefaultApi.CreateKafka(ctx, true, public.KafkaRequestPayload{
		Name: "kafka-with-short-lifespan",
		Plan: fmt.Sprintf("%s.x1", types.DEVELOPER), // has a short lifespan of 10s
	})
	g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to create kafka with short lifespan")
	expectedExpirationDate = result.CreatedAt.Add(time.Duration(shortLifespanSeconds) * time.Second).Format(time.RFC1123Z)
	g.Expect(result.ExpiresAt.Format(time.RFC1123Z)).To(gomega.Equal(expectedExpirationDate), "expected expires_at for kafka with short lifespan does not match actual")

	err = common.WaitForKafkaToBeDeleted(ctx, test.TestServices.DBFactory, client, result.Id)
	g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to wait for kafka with short lifespan to be deleted")
}
