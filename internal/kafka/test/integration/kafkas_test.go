package integration

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/acl"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/quota_management"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/common"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kasfleetshardsync"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm/clusterservicetest"

	mockkafka "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	coreTest "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"

	"github.com/bxcodec/faker/v3"
	"github.com/golang-jwt/jwt/v4"
	"github.com/onsi/gomega"
)

const (
	mockKafkaName      = "test-kafka1"
	kafkaReadyTimeout  = time.Minute * 10
	kafkaCheckInterval = time.Second * 1
	testMultiAZ        = true
	invalidKafkaName   = "Test_Cluster9"
	longKafkaName      = "thisisaninvalidkafkaclusternamethatexceedsthenamesizelimit"
)

type errorCheck struct {
	wantErr  bool
	code     errors.ServiceErrorCode
	httpCode int
}

type kafkaValidation struct {
	kafka             *dbapi.KafkaRequest
	eCheck            errorCheck
	capacityUsed      string
	assignedClusterID string
}

// TestKafkaCreate_Success validates the happy path of the kafka post endpoint:
// - kafka successfully registered with database
// - kafka worker picks up on creation job
// - cluster is found for kafka
// - kafka is assigned cluster
func TestKafkaCreate_Success(t *testing.T) {
	g := gomega.NewWithT(t)
	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, nil)
	defer teardown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	clusterID, getClusterErr := common.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}
	// setup pre-requisites to performing requests
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	// POST responses per openapi spec: 201, 409, 500
	k := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
		Plan:          fmt.Sprintf("%s.x1", types.STANDARD.String()),
	}

	kafka, resp, err := common.WaitForKafkaCreateToBeAccepted(ctx, test.TestServices.DBFactory, client, k)
	if resp != nil {
		resp.Body.Close()
	}
	// kafka successfully registered with database
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error posting object:  %v", err)
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusAccepted))
	g.Expect(kafka.Id).NotTo(gomega.BeEmpty(), "g.Expected ID assigned on creation")
	g.Expect(kafka.Kind).To(gomega.Equal(presenters.KindKafka))
	g.Expect(kafka.Href).To(gomega.Equal(fmt.Sprintf("/api/kafkas_mgmt/v1/kafkas/%s", kafka.Id)))
	g.Expect(kafka.InstanceType).To(gomega.Equal(types.STANDARD.String()))
	g.Expect(kafka.ReauthenticationEnabled).To(gomega.BeTrue())
	g.Expect(kafka.BrowserUrl).To(gomega.Equal(fmt.Sprintf("%s%s/dashboard", test.TestServices.KafkaConfig.BrowserUrl, kafka.Id)))
	g.Expect(kafka.InstanceTypeName).To(gomega.Equal("Standard"))
	g.Expect(kafka.ExpiresAt).To(gomega.BeNil())
	g.Expect(kafka.AdminApiServerUrl).To(gomega.BeEmpty())

	// wait until the kafka goes into a ready state
	// the timeout here assumes a backing cluster has already been provisioned
	foundKafka, err := common.WaitForKafkaToReachStatus(ctx, test.TestServices.DBFactory, client, kafka.Id, constants2.KafkaRequestStatusReady)
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error waiting for kafka request to become ready: %v", err)
	// check the owner is set correctly
	g.Expect(foundKafka.Owner).To(gomega.Equal(account.Username()))
	g.Expect(foundKafka.BootstrapServerHost).To(gomega.Not(gomega.BeEmpty()))
	g.Expect(foundKafka.Version).To(gomega.Equal(kasfleetshardsync.GetDefaultReportedKafkaVersion()))
	g.Expect(foundKafka.Owner).To(gomega.Equal(kafka.Owner))
	g.Expect(foundKafka.AdminApiServerUrl).To(gomega.Equal(kasfleetshardsync.AdminServerURI))

	// checking kafka_request bootstrap server port number being present
	kafka, resp, err = client.DefaultApi.GetKafkaById(ctx, foundKafka.Id)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error getting created kafka_request:  %v", err)
	g.Expect(strings.HasSuffix(kafka.BootstrapServerHost, ":443")).To(gomega.Equal(true))
	g.Expect(kafka.Version).To(gomega.Equal(kasfleetshardsync.GetDefaultReportedKafkaVersion()))
	g.Expect(kafka.AdminApiServerUrl).To(gomega.Equal(kasfleetshardsync.AdminServerURI))

	db := test.TestServices.DBFactory.New()
	var kafkaRequest dbapi.KafkaRequest
	if err := db.Unscoped().Where("id = ?", kafka.Id).First(&kafkaRequest).Error; err != nil {
		t.Error("failed to find kafka request")
	}

	g.Expect(kafkaRequest.QuotaType).To(gomega.Equal(KafkaConfig(h).Quota.Type))
	g.Expect(kafkaRequest.PlacementId).To(gomega.Not(gomega.BeEmpty()))
	g.Expect(kafkaRequest.Owner).To(gomega.Not(gomega.BeEmpty()))
	g.Expect(kafkaRequest.Namespace).To(gomega.Equal(fmt.Sprintf("kafka-%s", strings.ToLower(kafkaRequest.ID))))
	// this is set by the mockKasfFleetshardSync
	g.Expect(kafkaRequest.DesiredStrimziVersion).To(gomega.Equal("strimzi-cluster-operator.v0.23.0-0"))
	// default kafka_storage_size should be set on creation
	instanceType, err := test.TestServices.KafkaConfig.SupportedInstanceTypes.Configuration.GetKafkaInstanceTypeByID(kafka.InstanceType)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	instanceSize, err := instanceType.GetKafkaInstanceSizeByID(kafka.SizeId)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	g.Expect(kafka.DeprecatedKafkaStorageSize).To(gomega.Equal(instanceSize.MaxDataRetentionSize.String()))

	maxDataRetentionSizeBytes, err := instanceSize.MaxDataRetentionSize.ToInt64()
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(kafka.MaxDataRetentionSize.Bytes).To(gomega.Equal(maxDataRetentionSizeBytes))

	common.CheckMetricExposed(h, t, metrics.KafkaCreateRequestDuration)
	common.CheckMetricExposed(h, t, metrics.ClusterStatusCapacityUsed)
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount, constants2.KafkaOperationCreate.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount, constants2.KafkaOperationCreate.String()))

	// delete test kafka to free up resources on an OSD cluster
	deleteTestKafka(t, h, ctx, client, foundKafka.Id)
}

func TestKafkaCreate_ValidatePlanParam(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, nil)
	defer teardown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	clusterID, getClusterErr := common.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	k := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
		Plan:          fmt.Sprintf("%s.x1", types.STANDARD.String()),
	}

	kafka, resp, err := client.DefaultApi.CreateKafka(ctx, true, k)
	if resp != nil {
		resp.Body.Close()
	}
	// successful creation of kafka with a valid "standard" plan format
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error posting object:  %v", err)
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusAccepted))
	g.Expect(kafka.Id).NotTo(gomega.BeEmpty(), "g.Expected ID assigned on creation")
	g.Expect(kafka.InstanceType).To(gomega.Equal(types.STANDARD.String()))
	g.Expect(kafka.MultiAz).To(gomega.BeTrue())
	g.Expect(kafka.ExpiresAt).To(gomega.BeNil())

	// successful creation of kafka with valid "developer plan format
	k2 := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          "test-kafka-2",
		Plan:          fmt.Sprintf("%s.x1", types.DEVELOPER.String()),
	}
	accountWithoutStandardInstances := h.NewAccountWithNameAndOrg("test-nameacc-2", "123456")
	ctx2 := h.NewAuthenticatedContext(accountWithoutStandardInstances, nil)
	kafka, resp, err = client.DefaultApi.CreateKafka(ctx2, true, k2)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error posting object:  %v", err)
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusAccepted))
	g.Expect(kafka.Id).NotTo(gomega.BeEmpty(), "g.Expected ID assigned on creation")
	g.Expect(kafka.InstanceType).To(gomega.Equal(types.DEVELOPER.String()))
	g.Expect(kafka.MultiAz).To(gomega.BeFalse())
	// Verify that developer instances should have an expiration time set
	g.Expect(kafka.ExpiresAt).NotTo(gomega.BeNil())
	instanceTypeConfig, err := test.TestServices.KafkaConfig.SupportedInstanceTypes.Configuration.GetKafkaInstanceTypeByID(kafka.InstanceType)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	instanceTypeSizeConfig, err := instanceTypeConfig.GetKafkaInstanceSizeByID("x1")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(*kafka.ExpiresAt).To(gomega.Equal(kafka.CreatedAt.Add(time.Duration(*instanceTypeSizeConfig.LifespanSeconds) * time.Second)))

	// unsuccessful creation of kafka with invalid instance type provided in the "plan" parameter
	k.Plan = "invalid.x1"
	kafka, resp, err = client.DefaultApi.CreateKafka(ctx, true, k)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).To(gomega.HaveOccurred(), "Error should have occurred when attempting to create kafka with invalid instance type provided in the plan")

	// unsuccessful creation of kafka with invalid size_id provided in the "plan" parameter
	k.Plan = fmt.Sprintf("%s.x9999", types.STANDARD.String())
	kafka, resp, err = client.DefaultApi.CreateKafka(ctx, true, k)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).To(gomega.HaveOccurred(), "Error should have occurred when attempting to create kafka unsupported size_id")
}

func TestKafka_InstanceTypeCapacity(t *testing.T) {
	g := gomega.NewWithT(t)

	clusterDns := "app.example.com"
	clusterCriteria := services.FindClusterCriteria{
		Provider: "aws",
		Region:   "us-east-1",
		MultiAZ:  true,
	}
	// Start with no cluster config and manual scaling.
	configHook := func(clusterConfig *config.DataplaneClusterConfig, reconcilerConfig *workers.ReconcilerConfig) {
		// set the interval to 1 second to have sufficient time for the metric to be propagate
		// so that metrics checks does not timeout after 10s
		reconcilerConfig.ReconcilerRepeatInterval = 1 * time.Second
		clusterConfig.DataPlaneClusterScalingType = config.ManualScaling
		clusterConfig.ClusterConfig = config.NewClusterConfig(config.ClusterList{
			config.ManualCluster{ClusterId: "first", ClusterDNS: clusterDns, Status: api.ClusterReady, KafkaInstanceLimit: 2, Region: clusterCriteria.Region, MultiAZ: clusterCriteria.MultiAZ, CloudProvider: clusterCriteria.Provider, Schedulable: true, SupportedInstanceType: "standard,developer"},
			config.ManualCluster{ClusterId: "second", ClusterDNS: clusterDns, Status: api.ClusterReady, KafkaInstanceLimit: 2, Region: clusterCriteria.Region, MultiAZ: clusterCriteria.MultiAZ, CloudProvider: clusterCriteria.Provider, Schedulable: true, SupportedInstanceType: "standard"},
			config.ManualCluster{ClusterId: "third", ClusterDNS: clusterDns, Status: api.ClusterReady, KafkaInstanceLimit: 2, Region: clusterCriteria.Region, MultiAZ: clusterCriteria.MultiAZ, CloudProvider: clusterCriteria.Provider, Schedulable: true, SupportedInstanceType: "developer"},
			config.ManualCluster{ClusterId: "fourth", ClusterDNS: clusterDns, Status: api.ClusterReady, KafkaInstanceLimit: 2, Region: clusterCriteria.Region, MultiAZ: clusterCriteria.MultiAZ, CloudProvider: clusterCriteria.Provider, Schedulable: true, SupportedInstanceType: "standard"},
			config.ManualCluster{ClusterId: "fifth", ClusterDNS: clusterDns, Status: api.ClusterReady, KafkaInstanceLimit: 2, Region: clusterCriteria.Region, MultiAZ: clusterCriteria.MultiAZ, CloudProvider: clusterCriteria.Provider, Schedulable: true, SupportedInstanceType: "developer"},
			config.ManualCluster{ClusterId: "sixth", ClusterDNS: clusterDns, Status: api.ClusterReady, KafkaInstanceLimit: 0, Region: clusterCriteria.Region, MultiAZ: clusterCriteria.MultiAZ, CloudProvider: clusterCriteria.Provider, Schedulable: true, SupportedInstanceType: "standard,developer"},
		})
	}

	// setup ocm server
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// start servers
	h, _, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, configHook)
	defer teardown()

	ocmConfig := test.TestServices.OCMConfig

	if ocmConfig.MockMode != ocm.MockModeEmulateServer || h.Env.Name == environments.TestingEnv {
		t.SkipNow()
	}

	kasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	kasfFleetshardSync := kasFleetshardSyncBuilder.Build()
	kasfFleetshardSync.Start()

	unsupportedRegion := "eu-west-1"

	providerConfig := ProviderConfig(h)

	developerLimit := int(5)

	standardLimit := int(5)

	providerConfig.ProvidersConfig = config.ProviderConfiguration{
		SupportedProviders: config.ProviderList{
			{
				Name:    "aws",
				Default: true,
				Regions: config.RegionList{
					{
						Name:    "us-east-1",
						Default: true,
						SupportedInstanceTypes: config.InstanceTypeMap{
							"standard": config.InstanceTypeConfig{
								Limit: &standardLimit,
							},
							"developer": config.InstanceTypeConfig{
								Limit: &developerLimit,
							},
						},
					},
				},
			},
		},
	}

	kafkaConfig := KafkaConfig(h)

	kafkaConfig.SupportedInstanceTypes = &config.KafkaSupportedInstanceTypesConfig{
		Configuration: config.SupportedKafkaInstanceTypesConfig{
			SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
				{
					Id:          "standard",
					DisplayName: "Standard",
					Sizes: []config.KafkaInstanceSize{
						{
							Id:                          "x1",
							DisplayName:                 "1",
							IngressThroughputPerSec:     "30Mi",
							EgressThroughputPerSec:      "30Mi",
							TotalMaxConnections:         1000,
							MaxDataRetentionSize:        "100Gi",
							MaxPartitions:               1000,
							MaxDataRetentionPeriod:      "P14D",
							MaxConnectionAttemptsPerSec: 100,
							QuotaConsumed:               1,
							QuotaType:                   "rhosak",
							CapacityConsumed:            1,
							MaxMessageSize:              "1Mi",
							MinInSyncReplicas:           2,
							ReplicationFactor:           3,
							SupportedAZModes:            []string{"multi"},
							MaturityStatus:              config.MaturityStatusStable,
						},
						{
							Id:                          "x2",
							DisplayName:                 "2",
							IngressThroughputPerSec:     "60Mi",
							EgressThroughputPerSec:      "60Mi",
							TotalMaxConnections:         2000,
							MaxDataRetentionSize:        "200Gi",
							MaxPartitions:               2000,
							MaxDataRetentionPeriod:      "P14D",
							MaxConnectionAttemptsPerSec: 200,
							QuotaConsumed:               2,
							QuotaType:                   "rhosak",
							CapacityConsumed:            2,
							MaxMessageSize:              "1Mi",
							MinInSyncReplicas:           2,
							ReplicationFactor:           3,
							SupportedAZModes:            []string{"multi"},
							MaturityStatus:              config.MaturityStatusTechPreview,
						},
						{
							Id:                          "x3",
							DisplayName:                 "3",
							IngressThroughputPerSec:     "90Mi",
							EgressThroughputPerSec:      "90Mi",
							TotalMaxConnections:         2000,
							MaxDataRetentionSize:        "200Gi",
							MaxPartitions:               2000,
							MaxDataRetentionPeriod:      "P14D",
							MaxConnectionAttemptsPerSec: 200,
							QuotaConsumed:               3,
							QuotaType:                   "rhosak",
							CapacityConsumed:            3,
							MaxMessageSize:              "1Mi",
							MinInSyncReplicas:           2,
							ReplicationFactor:           3,
							SupportedAZModes:            []string{"multi"},
							MaturityStatus:              config.MaturityStatusTechPreview,
						},
					},
				},
				{
					Id:          "developer",
					DisplayName: "Trial",
					Sizes: []config.KafkaInstanceSize{
						{
							Id:                          "x1",
							DisplayName:                 "1",
							IngressThroughputPerSec:     "60Mi",
							EgressThroughputPerSec:      "60Mi",
							TotalMaxConnections:         2000,
							MaxDataRetentionSize:        "200Gi",
							MaxPartitions:               2000,
							MaxDataRetentionPeriod:      "P14D",
							MaxConnectionAttemptsPerSec: 200,
							QuotaConsumed:               1,
							QuotaType:                   "rhosak",
							CapacityConsumed:            1,
							MaxMessageSize:              "1Mi",
							MinInSyncReplicas:           1,
							ReplicationFactor:           1,
							SupportedAZModes:            []string{"single"},
							MaturityStatus:              config.MaturityStatusTechPreview,
						},
						{
							Id:                          "x2",
							DisplayName:                 "2",
							IngressThroughputPerSec:     "60Mi",
							EgressThroughputPerSec:      "60Mi",
							TotalMaxConnections:         2000,
							MaxDataRetentionSize:        "200Gi",
							MaxPartitions:               2000,
							MaxDataRetentionPeriod:      "P14D",
							MaxConnectionAttemptsPerSec: 200,
							QuotaConsumed:               2,
							QuotaType:                   "rhosak",
							CapacityConsumed:            2,
							MaxMessageSize:              "1Mi",
							MinInSyncReplicas:           1,
							ReplicationFactor:           1,
							SupportedAZModes:            []string{"single"},
							MaturityStatus:              config.MaturityStatusTechPreview,
						},
						{
							Id:                          "x3",
							DisplayName:                 "3",
							IngressThroughputPerSec:     "90Mi",
							EgressThroughputPerSec:      "90Mi",
							TotalMaxConnections:         2000,
							MaxDataRetentionSize:        "200Gi",
							MaxPartitions:               2000,
							MaxDataRetentionPeriod:      "P14D",
							MaxConnectionAttemptsPerSec: 200,
							QuotaConsumed:               3,
							QuotaType:                   "rhosak",
							CapacityConsumed:            3,
							MaxMessageSize:              "1Mi",
							MinInSyncReplicas:           1,
							ReplicationFactor:           1,
							SupportedAZModes:            []string{"single"},
							MaturityStatus:              config.MaturityStatusTechPreview,
						},
					},
				},
			},
		},
	}

	pollErr := common.WaitForClustersMatchCriteriaToBeGivenCount(test.TestServices.DBFactory, &test.TestServices.ClusterService, &clusterCriteria, 6)
	g.Expect(pollErr).NotTo(gomega.HaveOccurred())

	// add identity provider id for all of the clusters to avoid configuring the KAFKA_SRE IDP as it is not needed
	db := test.TestServices.DBFactory.New()
	err := db.Exec("UPDATE clusters set identity_provider_id = 'some-identity-provider-id'").Error
	g.Expect(err).NotTo(gomega.HaveOccurred())

	// Need to mark the clusters to ClusterWaitingForKasFleetShardOperator so that fleetshardsync and placement can actually happen
	updateErr := test.TestServices.ClusterService.UpdateMultiClusterStatus([]string{"first", "second", "third", "fourth"}, api.ClusterReady)
	g.Expect(updateErr).NotTo(gomega.HaveOccurred())

	tooLargeStandardKafka := buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
		kafkaRequest.Owner = "testuser1@example.com"
		kafkaRequest.Name = "dummy-kafka-1"
		kafkaRequest.InstanceType = types.STANDARD.String()
		kafkaRequest.SizeId = "x3"
	})

	x2StandardKafka := buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
		kafkaRequest.Owner = "testuser1@example.com"
		kafkaRequest.Name = "dummy-kafka-2"
		kafkaRequest.InstanceType = types.STANDARD.String()
		kafkaRequest.SizeId = "x2"
	})

	standardKafka1 := buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
		kafkaRequest.Owner = "testuser1@example.com"
		kafkaRequest.Name = "dummy-kafka-3"
		kafkaRequest.InstanceType = types.STANDARD.String()
		kafkaRequest.SizeId = "x1"
	})

	standardKafka2 := buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
		kafkaRequest.Owner = "testuser2@example.com"
		kafkaRequest.Name = "dummy-kafka-4"
		kafkaRequest.InstanceType = types.STANDARD.String()
		kafkaRequest.SizeId = "x1"
	})

	standardKafka3 := buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
		kafkaRequest.Owner = "testuser3@example.com"
		kafkaRequest.Name = "dummy-kafka-5"
		kafkaRequest.InstanceType = types.STANDARD.String()
		kafkaRequest.SizeId = "x1"
	})

	standardKafka4 := buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
		kafkaRequest.Owner = "testuser4@example.com"
		kafkaRequest.Name = "dummy-kafka-6"
		kafkaRequest.InstanceType = types.STANDARD.String()
		kafkaRequest.SizeId = "x1"
	})

	standardKafkaIncorrectRegion := buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
		kafkaRequest.Owner = "testuser5@example.com"
		kafkaRequest.Name = "dummy-kafka-7"
		kafkaRequest.InstanceType = types.STANDARD.String()
		kafkaRequest.Region = unsupportedRegion
		kafkaRequest.SizeId = "x1"
	})

	tooLargeDeveloperKafka := buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
		kafkaRequest.Owner = "dummyuser1"
		kafkaRequest.Name = "dummy-kafka-8"
		kafkaRequest.InstanceType = types.DEVELOPER.String()
		kafkaRequest.SizeId = "x3"
	})

	x2DeveloperKafka := buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
		kafkaRequest.Owner = "dummyuser2"
		kafkaRequest.Name = "dummy-kafka-9"
		kafkaRequest.InstanceType = types.DEVELOPER.String()
		kafkaRequest.SizeId = "x2"
	})

	developerKafka1 := buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
		kafkaRequest.Owner = "dummyuser3"
		kafkaRequest.Name = "dummy-kafka-10"
		kafkaRequest.InstanceType = types.DEVELOPER.String()
		kafkaRequest.SizeId = "x1"
	})

	developerKafka2 := buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
		kafkaRequest.Owner = "dummyuser4"
		kafkaRequest.Name = "dummy-kafka-11"
		kafkaRequest.InstanceType = types.DEVELOPER.String()
		kafkaRequest.SizeId = "x1"
	})

	developerKafka3 := buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
		kafkaRequest.Owner = "dummyuser5"
		kafkaRequest.Name = "dummy-kafka-12"
		kafkaRequest.InstanceType = types.DEVELOPER.String()
		kafkaRequest.SizeId = "x1"
	})

	developerKafka4 := buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
		kafkaRequest.Owner = "dummyuser6"
		kafkaRequest.Name = "dummy-kafka-13"
		kafkaRequest.InstanceType = types.DEVELOPER.String()
		kafkaRequest.SizeId = "x1"
	})

	developerKafkaIncorrectRegion := buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
		kafkaRequest.Owner = "testuser5@example.com"
		kafkaRequest.Name = "dummy-kafka-14"
		kafkaRequest.InstanceType = types.DEVELOPER.String()
		kafkaRequest.Region = unsupportedRegion
		kafkaRequest.SizeId = "x1"
	})

	errorCheckWithError := errorCheck{
		wantErr:  true,
		code:     errors.ErrorTooManyKafkaInstancesReached,
		httpCode: http.StatusForbidden,
	}

	errorCheckNoError := errorCheck{
		wantErr: false,
	}

	errorCheckUnsupportedRegion := errorCheck{
		wantErr:  true,
		code:     errors.ErrorRegionNotSupported,
		httpCode: http.StatusBadRequest,
	}

	kafkaValidations := []kafkaValidation{
		// first two kafkas are larger than cluster capacity (3 vs 2), hence can't be placed into
		// any cluster despite of the global capacity being 5 for developer and standard instance types
		{
			kafka:        tooLargeStandardKafka,
			eCheck:       errorCheckWithError,
			capacityUsed: "",
		},
		{
			kafka:        tooLargeDeveloperKafka,
			eCheck:       errorCheckWithError,
			capacityUsed: "",
		},
		{
			kafka:             x2StandardKafka,
			eCheck:            errorCheckNoError,
			capacityUsed:      "2", // the kafka will use two capacity of the "first" cluster as a standard instance
			assignedClusterID: "first",
		},
		{
			kafka:             x2DeveloperKafka,
			eCheck:            errorCheckNoError,
			capacityUsed:      "2", // the kafka will use two capacity of the "third" cluster as a developer instance
			assignedClusterID: "third",
		},
		{
			kafka:             standardKafka1,
			eCheck:            errorCheckNoError,
			capacityUsed:      "1", // the kafka will use one capacity of the "second" cluster as a standard instance
			assignedClusterID: "second",
		},
		{
			kafka:             standardKafka2,
			eCheck:            errorCheckNoError,
			capacityUsed:      "2", // the kafka will use the remaining one capacity of the "second" cluster as a standard instance
			assignedClusterID: "second",
		},
		{
			kafka:             standardKafka3,
			eCheck:            errorCheckNoError,
			capacityUsed:      "1", // the kafka will use one capacity of the "fourth" cluster as a standard instance
			assignedClusterID: "fourth",
		},
		{
			kafka:             developerKafka1,
			eCheck:            errorCheckNoError,
			capacityUsed:      "1", // the kafka will use one capacity of the "fifth" cluster as a developer instance
			assignedClusterID: "fifth",
		},
		{
			kafka:             developerKafka2,
			eCheck:            errorCheckNoError,
			capacityUsed:      "2", // the kafka will use the remaining capacity of the "fifth" cluster
			assignedClusterID: "fifth",
		},
		// the "standard,developer" cluster should be filled up by standard instances, hence despite of
		// global capacity being available for developer instances, there is no space on clusters supporting this type of instance
		{
			kafka:        developerKafka3,
			eCheck:       errorCheckWithError,
			capacityUsed: "",
		},
		{
			kafka:        developerKafka4,
			eCheck:       errorCheckWithError,
			capacityUsed: "",
		},
		{
			kafka:        standardKafka4,
			eCheck:       errorCheckWithError,
			capacityUsed: "",
		},
		{
			kafka:        standardKafkaIncorrectRegion,
			eCheck:       errorCheckUnsupportedRegion,
			capacityUsed: "",
		},
		{
			kafka:        developerKafkaIncorrectRegion,
			eCheck:       errorCheckUnsupportedRegion,
			capacityUsed: "",
		},
	}

	defer func() {
		kasfFleetshardSync.Stop()
		h.Env.Stop()

		list, _ := test.TestServices.ClusterService.FindAllClusters(clusterCriteria)
		for _, clusters := range list {
			if err := getAndDeleteServiceAccounts(clusters.ClientID, h.Env); err != nil {
				t.Fatalf("Failed to delete service account with client id: %v", clusters.ClientID)
			}
			if err := getAndDeleteServiceAccounts(clusters.ID, h.Env); err != nil {
				t.Fatalf("Failed to delete service account with cluster id: %v", clusters.ID)
			}
		}
	}()

	testValidations(h, t, kafkaValidations)
}

// build a test kafka request
func buildKafkaRequest(modifyFn func(kafkaRequest *dbapi.KafkaRequest)) *dbapi.KafkaRequest {
	kafkaRequest := &dbapi.KafkaRequest{
		Region:        "us-east-1",
		CloudProvider: "aws",
		Name:          "dummy-kafka-1",
		MultiAZ:       true,
		Owner:         "dummyuser2",
		Status:        constants2.KafkaRequestStatusAccepted.String(),
		InstanceType:  types.STANDARD.String(),
	}
	if modifyFn != nil {
		modifyFn(kafkaRequest)
	}
	return kafkaRequest
}

func testValidations(h *coreTest.Helper, t *testing.T, kafkaValidations []kafkaValidation) {
	g := gomega.NewWithT(t)

	for _, val := range kafkaValidations {
		errK := test.TestServices.KafkaService.RegisterKafkaJob(val.kafka)
		if (errK != nil) != val.eCheck.wantErr {
			t.Errorf("Ung.Expected error %v, wantErr = %v for kafka %s", errK, val.eCheck.wantErr, val.kafka.Name)
		}

		if val.eCheck.wantErr {
			if errK == nil {
				t.Errorf("RegisterKafkaJob() g.Expected err to be received but got nil")
			} else {
				if errK.Code != val.eCheck.code {
					t.Errorf("RegisterKafkaJob() received error code %v, g.Expected error %v for kafka %s", errK.Code, val.eCheck.code, val.kafka.Name)
				}
				if errK.HttpCode != val.eCheck.httpCode {
					t.Errorf("RegisterKafkaJob() received http code %v, g.Expected %v for kafka %s", errK.HttpCode, val.eCheck.httpCode, val.kafka.Name)
				}
			}
		} else {
			g.Expect(val.kafka.ClusterID).To(gomega.Equal(val.assignedClusterID))
			checkMetricsError := common.WaitForMetricToBePresent(h, t, metrics.ClusterStatusCapacityUsed, val.capacityUsed, val.kafka.InstanceType, val.assignedClusterID, val.kafka.Region, val.kafka.CloudProvider)
			g.Expect(checkMetricsError).NotTo(gomega.HaveOccurred())
		}
	}
}

func ProviderConfig(h *coreTest.Helper) (providerConfig *config.ProviderConfig) {
	h.Env.MustResolve(&providerConfig)
	return
}

func TestKafka_Update(t *testing.T) {
	g := gomega.NewWithT(t)

	owner := "test-user"
	anotherOwner := "test_kafka_service"
	emptyOwner := ""
	sampleKafkaID := api.NewID()
	reauthenticationEnabled := true
	reauthenticationEnabled2 := false
	emptyOwnerKafkaUpdateReq := public.KafkaUpdateRequest{Owner: &emptyOwner, ReauthenticationEnabled: &reauthenticationEnabled}
	sameOwnerKafkaUpdateReq := public.KafkaUpdateRequest{Owner: &owner}
	newOwnerKafkaUpdateReq := public.KafkaUpdateRequest{Owner: &anotherOwner}

	emptyKafkaUpdate := public.KafkaUpdateRequest{}

	reauthenticationUpdateToFalse := public.KafkaUpdateRequest{ReauthenticationEnabled: &reauthenticationEnabled2}
	reauthenticationUpdateToTrue := public.KafkaUpdateRequest{ReauthenticationEnabled: &reauthenticationEnabled}

	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedGetClusterResponse, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	if err != nil {
		t.Fatalf(err.Error())
	}
	ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	orgId := "13640203"
	ownerAccout := h.NewAccount(owner, owner, "some-email@kafka.com", orgId)
	nonOwnerAccount := h.NewAccount("other-user", "some-other-user", "some-other@kafka.com", orgId)

	ctx := h.NewAuthenticatedContext(ownerAccout, nil)
	nonOwnerCtx := h.NewAuthenticatedContext(nonOwnerAccount, nil)

	adminAccount := h.NewRandAccount()
	claims := jwt.MapClaims{
		"is_org_admin": true,
	}

	adminCtx := h.NewAuthenticatedContext(adminAccount, claims)

	type args struct {
		ctx                context.Context
		kafkaID            string
		kafkaUpdateRequest public.KafkaUpdateRequest
	}
	tests := []struct {
		name           string
		args           args
		verifyResponse func(result public.KafkaRequest, resp *http.Response, err error)
	}{
		{
			name: "update with an empty body should not fail",
			args: args{
				ctx:                ctx,
				kafkaID:            sampleKafkaID,
				kafkaUpdateRequest: emptyKafkaUpdate,
			},
			verifyResponse: func(result public.KafkaRequest, resp *http.Response, err error) {
				g.Expect(err).To(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
			},
		},
		{
			name: "should fail if owner is empty",
			args: args{
				ctx:                ctx,
				kafkaID:            sampleKafkaID,
				kafkaUpdateRequest: emptyOwnerKafkaUpdateReq,
			},
			verifyResponse: func(result public.KafkaRequest, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should fail if trying to update owner as not an owner/ org_admin",
			args: args{
				ctx:                nonOwnerCtx,
				kafkaID:            sampleKafkaID,
				kafkaUpdateRequest: newOwnerKafkaUpdateReq,
			},
			verifyResponse: func(result public.KafkaRequest, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should fail if trying to update with an empty body as a non owner or org admin",
			args: args{
				ctx:                nonOwnerCtx,
				kafkaID:            sampleKafkaID,
				kafkaUpdateRequest: emptyKafkaUpdate,
			},
			verifyResponse: func(result public.KafkaRequest, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should fail when trying to update non-existent kafka",
			args: args{
				ctx:                ctx,
				kafkaID:            "non-existent-id",
				kafkaUpdateRequest: sameOwnerKafkaUpdateReq,
			},
			verifyResponse: func(result public.KafkaRequest, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNotFound))
			},
		},
		{
			name: "should succeed changing reauthentication enabled value to true",
			args: args{
				ctx:                ctx,
				kafkaID:            sampleKafkaID,
				kafkaUpdateRequest: reauthenticationUpdateToTrue,
			},
			verifyResponse: func(result public.KafkaRequest, resp *http.Response, err error) {
				g.Expect(err).To(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
				g.Expect(result.ReauthenticationEnabled).To(gomega.BeTrue())
			},
		},
		{
			name: "should succeed changing reauthentication enabled value to false",
			args: args{
				ctx:                ctx,
				kafkaID:            sampleKafkaID,
				kafkaUpdateRequest: reauthenticationUpdateToFalse,
			},
			verifyResponse: func(result public.KafkaRequest, resp *http.Response, err error) {
				g.Expect(err).To(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
				g.Expect(result.ReauthenticationEnabled).To(gomega.BeFalse())
			},
		},
		{
			name: "should succeed when passing a valid new username as an owner",
			args: args{
				ctx:                adminCtx,
				kafkaID:            sampleKafkaID,
				kafkaUpdateRequest: newOwnerKafkaUpdateReq,
			},
			verifyResponse: func(result public.KafkaRequest, resp *http.Response, err error) {
				g.Expect(err).To(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
				g.Expect(result.Owner).To(gomega.Equal(*newOwnerKafkaUpdateReq.Owner))
			},
		},
	}

	db := test.TestServices.DBFactory.New()
	// create a dummy cluster and assign a kafka to it
	cluster := &api.Cluster{
		Meta: api.Meta{
			ID: api.NewID(),
		},
		ClusterID:          api.NewID(),
		MultiAZ:            true,
		Region:             "baremetal",
		CloudProvider:      "baremetal",
		Status:             api.ClusterReady,
		IdentityProviderID: "some-id",
		ClusterDNS:         "some-cluster-dns",
		ProviderType:       api.ClusterProviderStandalone,
	}

	if err := db.Create(cluster).Error; err != nil {
		t.Error("failed to create dummy cluster")
		return
	}

	// create a kafka that will be updated
	kafka := mockkafka.BuildKafkaRequest(
		mockkafka.WithPredefinedTestValues(),
		mockkafka.With(mockkafka.CLUSTER_ID, cluster.ClusterID),
		mockkafka.WithReauthenticationEnabled(false),
		mockkafka.With(mockkafka.ID, sampleKafkaID),
	)

	if err := db.Create(kafka).Error; err != nil {
		t.Errorf("failed to create Kafka db record due to error: %v", err)
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			client := test.NewApiClient(h)
			result, resp, err := client.DefaultApi.UpdateKafkaById(tt.args.ctx, tt.args.kafkaID, tt.args.kafkaUpdateRequest)
			if resp != nil {
				resp.Body.Close()
			}
			tt.verifyResponse(result, resp, err)
		})
	}
}

func TestKafkaCreate_TooManyKafkas(t *testing.T) {
	g := gomega.NewWithT(t)

	// Start with no cluster config and manual scaling.
	configHook := func(clusterConfig *config.DataplaneClusterConfig) {
		clusterConfig.DataPlaneClusterScalingType = config.ManualScaling
		clusterConfig.ClusterConfig = config.NewClusterConfig(config.ClusterList{
			config.ManualCluster{ClusterId: "test01", ClusterDNS: "app.example.com", Status: api.ClusterReady, KafkaInstanceLimit: 1, Region: mocks.MockCluster.Region().ID(), MultiAZ: testMultiAZ, CloudProvider: mocks.MockCluster.CloudProvider().ID(), Schedulable: true, SupportedInstanceType: "standard,developer"},
		})
	}

	// setup ocm server
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// start servers
	h, client, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, configHook)
	defer teardown()

	ocmConfig := test.TestServices.OCMConfig

	if ocmConfig.MockMode != ocm.MockModeEmulateServer || h.Env.Name == environments.TestingEnv {
		t.SkipNow()
	}

	kasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	kasfFleetshardSync := kasFleetshardSyncBuilder.Build()
	kasfFleetshardSync.Start()
	defer kasfFleetshardSync.Stop()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	k := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          "dummy-kafka",
	}

	_, resp, err := client.DefaultApi.CreateKafka(ctx, true, k)
	if resp != nil {
		resp.Body.Close()
	}

	g.Expect(err).ToNot(gomega.HaveOccurred(), "Ung.Expected error occurred when creating kafka:  %v", err)

	k.Name = mockKafkaName

	_, resp, err = client.DefaultApi.CreateKafka(ctx, true, k)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).To(gomega.HaveOccurred(), "g.Expecting error to be thrown when creating more kafkas than allowed by the cluster limit:  %v", err)
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusForbidden))

	clusterCriteria := services.FindClusterCriteria{
		Provider: "aws",
		Region:   "us-east-1",
		MultiAZ:  true,
	}

	defer func() {
		kasfFleetshardSync.Stop()
		h.Env.Stop()

		list, _ := test.TestServices.ClusterService.FindAllClusters(clusterCriteria)
		for _, clusters := range list {
			if err := getAndDeleteServiceAccounts(clusters.ClientID, h.Env); err != nil {
				t.Fatalf("Failed to delete service account with client id: %v", clusters.ClientID)
			}
			if err := getAndDeleteServiceAccounts(clusters.ID, h.Env); err != nil {
				t.Fatalf("Failed to delete service account with cluster id: %v", clusters.ID)
			}
		}
	}()
}

// TestKafkaPost_Validations tests the API validations performed by the kafka creation endpoint
//
// these could also be unit tests
func TestKafkaPost_Validations(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	tests := []struct {
		name     string
		body     public.KafkaRequestPayload
		wantCode int
	}{
		{
			name: "HTTP 400 when region not supported",
			body: public.KafkaRequestPayload{
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				Region:        "us-east-3",
				Name:          mockKafkaName,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when provider not supported",
			body: public.KafkaRequestPayload{
				CloudProvider: "azure",
				Region:        mocks.MockCluster.Region().ID(),
				Name:          mockKafkaName,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when name not provided",
			body: public.KafkaRequestPayload{
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				Region:        mocks.MockCluster.Region().ID(),
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when name is not valid",
			body: public.KafkaRequestPayload{
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				Region:        mocks.MockCluster.Region().ID(),
				Name:          invalidKafkaName,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when name is too long",
			body: public.KafkaRequestPayload{
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				Region:        mocks.MockCluster.Region().ID(),
				Name:          longKafkaName,
			},
			wantCode: http.StatusBadRequest,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)

			_, resp, _ := client.DefaultApi.CreateKafka(ctx, true, tt.body)
			if resp != nil {
				resp.Body.Close()
			}
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantCode))
		})
	}
}

// TestKafkaPost_NameUniquenessValidations tests kafka cluster name uniqueness verification during its creation by the API
func TestKafkaPost_NameUniquenessValidations(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, tearDown := test.NewKafkaHelperWithHooks(t, ocmServer, nil)
	defer tearDown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	clusterID, getClusterErr := common.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	// create two random accounts in same organisation
	account1 := h.NewRandAccount()
	ctx1 := h.NewAuthenticatedContext(account1, nil)

	account2 := h.NewRandAccount()
	ctx2 := h.NewAuthenticatedContext(account2, nil)

	// this value is taken from config/quota-management-list-configuration.yaml
	anotherOrgID := "12147054"

	// create a third account in a different organisation
	accountFromAnotherOrg := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), anotherOrgID)
	ctx3 := h.NewAuthenticatedContext(accountFromAnotherOrg, nil)

	k := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
	}

	// create the first kafka
	_, resp1, _ := client.DefaultApi.CreateKafka(ctx1, true, k)
	if resp1 != nil {
		resp1.Body.Close()
	}
	// attempt to create the second kafka with same name
	_, resp2, _ := client.DefaultApi.CreateKafka(ctx2, true, k)
	if resp2 != nil {
		resp2.Body.Close()
	}
	// create another kafka with same name for different org
	_, resp3, _ := client.DefaultApi.CreateKafka(ctx3, true, k)
	if resp3 != nil {
		resp3.Body.Close()
	}
	// verify that the first and third requests were accepted
	g.Expect(resp1.StatusCode).To(gomega.Equal(http.StatusAccepted))
	g.Expect(resp3.StatusCode).To(gomega.Equal(http.StatusAccepted))

	// verify that the second request resulted into an error accepted
	g.Expect(resp2.StatusCode).To(gomega.Equal(http.StatusConflict))

}

// TestKafkaDenyList_UnauthorizedValidation tests the deny list API access validations is performed when enabled
func TestKafkaDenyList_UnauthorizedValidation(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	// create an account with a random organisation id. This is different than the control plane team organisation id which is used by default
	// this value is taken from config/deny-list-configuration.yaml
	username := "denied-test-user1@example.com"
	account := h.NewAccount(username, username, username, "13640203")
	ctx := h.NewAuthenticatedContext(account, nil)

	tests := []struct {
		name      string
		operation func() *http.Response
	}{
		{
			name: "HTTP 403 when listing kafkas",
			operation: func() *http.Response {
				_, resp, _ := client.DefaultApi.GetKafkas(ctx, &public.GetKafkasOpts{})
				return resp
			},
		},
		{
			name: "HTTP 403 when creating a new kafka request",
			operation: func() *http.Response {
				body := public.KafkaRequestPayload{
					CloudProvider: mocks.MockCluster.CloudProvider().ID(),
					Region:        "us-east-3",
					Name:          mockKafkaName,
				}

				_, resp, _ := client.DefaultApi.CreateKafka(ctx, true, body)
				return resp
			},
		},
		{
			name: "HTTP 403 when deleting new kafka request",
			operation: func() *http.Response {
				_, resp, _ := client.DefaultApi.DeleteKafkaById(ctx, "kafka-id", true)
				return resp
			},
		},
		{
			name: "HTTP 403 when getting a new kafka request",
			operation: func() *http.Response {
				_, resp, _ := client.DefaultApi.GetKafkaById(ctx, "kafka-id")
				return resp
			},
		},
	}

	// verify that the user does not have access to the service
	for _, rangevar := range tests {
		tt := rangevar
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			resp := (&tt).operation()
			if resp != nil {
				resp.Body.Close()
			}
			g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusForbidden))
			g.Expect(resp.Header.Get("Content-Type")).To(gomega.Equal("application/json"))
		})
	}
}

func TestKafka_AccessList_UnauthorizedValidation(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	//h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	h, client, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, func(acl *acl.AccessControlListConfig) {
		acl.EnableAccessList = true
	})
	defer teardown()

	// orgId value does not match any from config/access-list-configuration.yaml
	orgId := "not-on-access-list"
	account := h.NewAccount("testUsername", "testName", "testEmail", orgId)
	ctx := h.NewAuthenticatedContext(account, nil)

	tests := []struct {
		name      string
		acl       *acl.AccessControlListConfig
		modifyFn  func(acl *acl.AccessControlListConfig)
		operation func() *http.Response
	}{
		{
			name: "HTTP 403 when listing kafkas",
			operation: func() *http.Response {
				_, resp, _ := client.DefaultApi.GetKafkas(ctx, &public.GetKafkasOpts{})
				return resp
			},
		},
		{
			name: "HTTP 403 when creating a new kafka request",
			operation: func() *http.Response {
				body := public.KafkaRequestPayload{
					CloudProvider: mocks.MockCluster.CloudProvider().ID(),
					Region:        "us-east-3",
					Name:          mockKafkaName,
				}

				_, resp, _ := client.DefaultApi.CreateKafka(ctx, true, body)
				return resp
			},
		},
		{
			name: "HTTP 403 when deleting new kafka request",
			operation: func() *http.Response {
				_, resp, _ := client.DefaultApi.DeleteKafkaById(ctx, "kafka-id", true)
				return resp
			},
		},
		{
			name: "HTTP 403 when getting a new kafka request",
			operation: func() *http.Response {
				_, resp, _ := client.DefaultApi.GetKafkaById(ctx, "kafka-id")
				return resp
			},
		},
	}

	// verify that the organisation does not have access to the service
	for _, rangevar := range tests {
		tt := rangevar
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			resp := (&tt).operation()
			if resp != nil {
				resp.Body.Close()
			}
			g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusForbidden))
			g.Expect(resp.Header.Get("Content-Type")).To(gomega.Equal("application/json"))
		})
	}
}

// TestKafkaDenyList_RemovingKafkaForDeniedOwners tests that all kafkas of denied owners are removed
func TestKafkaDenyList_RemovingKafkaForDeniedOwners(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	clusterID, err := common.GetRunningOsdClusterID(h, t)
	if err != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", err)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	// create an account with a random organisation id. This is different than the control plane team organisation id which is used by default
	// these values are taken from config/deny-list-configuration.yaml
	username1 := "denied-test-user1@example.com"
	username2 := "denied-test-user2@example.com"

	// create dummy kafkas and assign it to user, at the end we'll verify that the kafka has been deleted
	kafkas := []*dbapi.KafkaRequest{
		mockkafka.BuildKafkaRequest(
			mockkafka.WithPredefinedTestValues(),
			mockkafka.With(mockkafka.NAME, "dummy-kafka"),
			mockkafka.With(mockkafka.OWNER, username1),
			mockkafka.With(mockkafka.CLUSTER_ID, clusterID),
			mockkafka.With(mockkafka.STATUS, constants2.KafkaRequestStatusAccepted.String()),
			mockkafka.With(mockkafka.BOOTSTRAP_SERVER_HOST, ""),
		),
		mockkafka.BuildKafkaRequest(
			mockkafka.WithPredefinedTestValues(),
			mockkafka.With(mockkafka.NAME, "dummy-kafka-2"),
			mockkafka.With(mockkafka.OWNER, username2),
			mockkafka.With(mockkafka.CLUSTER_ID, clusterID),
			mockkafka.With(mockkafka.STATUS, constants2.KafkaRequestStatusAccepted.String()),
			mockkafka.With(mockkafka.BOOTSTRAP_SERVER_HOST, ""),
			mockkafka.WithMultiAZ(false),
			mockkafka.With(mockkafka.INSTANCE_TYPE, types.DEVELOPER.String()),
		),
		mockkafka.BuildKafkaRequest(
			mockkafka.WithPredefinedTestValues(),
			mockkafka.With(mockkafka.NAME, "dummy-kafka-3"),
			mockkafka.With(mockkafka.OWNER, username2),
			mockkafka.With(mockkafka.CLUSTER_ID, clusterID),
			mockkafka.With(mockkafka.STATUS, constants2.KafkaRequestStatusPreparing.String()),
			mockkafka.With(mockkafka.BOOTSTRAP_SERVER_HOST, ""),
			mockkafka.WithMultiAZ(false),
			mockkafka.With(mockkafka.INSTANCE_TYPE, types.DEVELOPER.String()),
		),
		mockkafka.BuildKafkaRequest(
			mockkafka.WithPredefinedTestValues(),
			mockkafka.With(mockkafka.NAME, "dummy-kafka-to-deprovision"),
			mockkafka.With(mockkafka.OWNER, username2),
			mockkafka.With(mockkafka.CLUSTER_ID, clusterID),
			mockkafka.With(mockkafka.STATUS, constants2.KafkaRequestStatusProvisioning.String()),
		),
		mockkafka.BuildKafkaRequest(
			mockkafka.WithPredefinedTestValues(),
			mockkafka.With(mockkafka.NAME, "this-kafka-will-remain"),
			mockkafka.With(mockkafka.OWNER, "some-other-user"),
			mockkafka.With(mockkafka.CLUSTER_ID, clusterID),
			mockkafka.With(mockkafka.STATUS, constants2.KafkaRequestStatusAccepted.String()),
		),
	}

	db := test.TestServices.DBFactory.New()
	if err := db.Create(&kafkas).Error; err != nil {
		g.Expect(err).NotTo(gomega.HaveOccurred())
		return
	}

	// Verify all Kafkas that needs to be deleted are removed
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	kafkaDeletionErr := common.WaitForNumberOfKafkaToBeGivenCount(ctx, test.TestServices.DBFactory, client, 1)
	g.Expect(kafkaDeletionErr).NotTo(gomega.HaveOccurred(), "Error waiting for first kafka deletion: %v", kafkaDeletionErr)
}

// TestKafkaQuotaManagementList_MaxAllowedInstances tests the quota management list max allowed instances creation validations is performed when enabled
// The max allowed instances limit is set to "1" set for one organisation is not depassed.
// At the same time, users of a different organisation should be able to create instances.
func TestKafkaQuotaManagementList_MaxAllowedInstances(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, func(acl *quota_management.QuotaManagementListConfig, c *config.DataplaneClusterConfig) {
		acl.EnableInstanceLimitControl = true
	})
	defer teardown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	clusterID, getClusterErr := common.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	// this value is taken from config/quota-management-list-configuration.yaml
	orgIdWithLimitOfOne := "12147054"
	internalUserAccount := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), orgIdWithLimitOfOne)
	internalUserCtx := h.NewAuthenticatedContext(internalUserAccount, nil)

	kafka1 := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
	}

	kafka2 := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          "test-kafka-2",
	}
	//MultiAZ: false for developer/trial instances
	kafka3 := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          "test-kafka-3",
	}
	//MultiAZ: false for developer/trial instances
	kafka4 := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          "test-kafka-4",
	}
	//MultiAZ: false for developer/trial instances
	kafka5 := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          "test-kafka-5",
	}

	// create the first kafka
	resp1Body, resp1, _ := client.DefaultApi.CreateKafka(internalUserCtx, true, kafka1)
	if resp1 != nil {
		resp1.Body.Close()
	}
	// create the second kafka
	_, resp2, _ := client.DefaultApi.CreateKafka(internalUserCtx, true, kafka2)
	if resp2 != nil {
		resp2.Body.Close()
	}
	// verify that the request errored with 403 forbidden for the account in same organisation
	g.Expect(resp2.StatusCode).To(gomega.Equal(http.StatusForbidden))
	g.Expect(resp2.Header.Get("Content-Type")).To(gomega.Equal("application/json"))

	// verify that the first request was accepted as standard type
	g.Expect(resp1.StatusCode).To(gomega.Equal(http.StatusAccepted))
	g.Expect(resp1Body.InstanceType).To(gomega.Equal("standard"))

	// verify that user of the same org cannot create a new kafka since limit has been reached
	accountInSameOrg := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), orgIdWithLimitOfOne)
	internalUserCtx = h.NewAuthenticatedContext(accountInSameOrg, nil)

	_, sameOrgUserResp, _ := client.DefaultApi.CreateKafka(internalUserCtx, true, kafka2)
	if sameOrgUserResp != nil {
		sameOrgUserResp.Body.Close()
	}
	g.Expect(sameOrgUserResp.StatusCode).To(gomega.Equal(http.StatusForbidden))
	g.Expect(sameOrgUserResp.Header.Get("Content-Type")).To(gomega.Equal("application/json"))

	// verify that user of a different organisation can still create kafka instances
	accountInDifferentOrg := h.NewRandAccount()
	internalUserCtx = h.NewAuthenticatedContext(accountInDifferentOrg, nil)

	// attempt to create kafka for this user account
	_, resp4, _ := client.DefaultApi.CreateKafka(internalUserCtx, true, kafka1)
	if resp4 != nil {
		resp4.Body.Close()
	}
	// verify that the request was accepted
	g.Expect(resp4.StatusCode).To(gomega.Equal(http.StatusAccepted))

	// verify that an external user not listed in the quota list can create the default maximum allowed kafka instances
	externalUserAccount := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), "ext-org-id")
	externalUserCtx := h.NewAuthenticatedContext(externalUserAccount, nil)

	resp5Body, resp5, _ := client.DefaultApi.CreateKafka(externalUserCtx, true, kafka3)
	if resp5 != nil {
		resp5.Body.Close()
	}
	_, resp6, _ := client.DefaultApi.CreateKafka(externalUserCtx, true, kafka4)
	if resp6 != nil {
		resp6.Body.Close()
	}
	// verify that the first request was accepted for an developer type
	g.Expect(resp5.StatusCode).To(gomega.Equal(http.StatusAccepted))
	g.Expect(resp5Body.InstanceType).To(gomega.Equal("developer"))

	// verify that the second request for the external user errored with 403 Forbidden
	g.Expect(resp6.StatusCode).To(gomega.Equal(http.StatusForbidden))
	g.Expect(resp6.Header.Get("Content-Type")).To(gomega.Equal("application/json"))

	// verify that another external user in the same org can also create the default maximum allowed kafka instances
	externalUserSameOrgAccount := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), "ext-org-id")
	externalUserSameOrgCtx := h.NewAuthenticatedContext(externalUserSameOrgAccount, nil)

	_, resp7, _ := client.DefaultApi.CreateKafka(externalUserSameOrgCtx, true, kafka5)
	if resp7 != nil {
		resp7.Body.Close()
	}
	_, resp8, _ := client.DefaultApi.CreateKafka(externalUserSameOrgCtx, true, kafka4)
	if resp8 != nil {
		resp8.Body.Close()
	}
	// verify that the first request was accepted
	g.Expect(resp7.StatusCode).To(gomega.Equal(http.StatusAccepted))

	// verify that the second request for the external user errored with 403 Forbidden
	g.Expect(resp8.StatusCode).To(gomega.Equal(http.StatusForbidden))
	g.Expect(resp8.Header.Get("Content-Type")).To(gomega.Equal("application/json"))
}

// TestKafkaGet tests getting kafkas via the API endpoint
func TestKafkaGet(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, nil)
	defer teardown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	clusterID, getClusterErr := common.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	reauthenticationEnabled := false
	k := public.KafkaRequestPayload{
		Region:                  mocks.MockCluster.Region().ID(),
		CloudProvider:           mocks.MockCluster.CloudProvider().ID(),
		Name:                    mockKafkaName,
		ReauthenticationEnabled: &reauthenticationEnabled,
	}

	seedKafka, resp, err := client.DefaultApi.CreateKafka(ctx, true, k)
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("failed to create seeded kafka request: %s", err.Error())
	}

	// 200 OK
	kafka, resp, err := client.DefaultApi.GetKafkaById(ctx, seedKafka.Id)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error occurred when attempting to get kafka request:  %v", err)
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(kafka.Id).NotTo(gomega.BeEmpty(), "g.Expected ID assigned on creation")
	g.Expect(kafka.Kind).To(gomega.Equal(presenters.KindKafka))
	g.Expect(kafka.Href).To(gomega.Equal(fmt.Sprintf("/api/kafkas_mgmt/v1/kafkas/%s", kafka.Id)))
	g.Expect(kafka.Region).To(gomega.Equal(mocks.MockCluster.Region().ID()))
	g.Expect(kafka.CloudProvider).To(gomega.Equal(mocks.MockCluster.CloudProvider().ID()))
	g.Expect(kafka.Name).To(gomega.Equal(mockKafkaName))
	g.Expect(kafka.Status).To(gomega.Equal(constants2.KafkaRequestStatusAccepted.String()))
	g.Expect(kafka.ReauthenticationEnabled).To(gomega.BeFalse())

	instanceType, err := test.TestServices.KafkaConfig.SupportedInstanceTypes.Configuration.GetKafkaInstanceTypeByID(kafka.InstanceType)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	instanceSize, err := instanceType.GetKafkaInstanceSizeByID(kafka.SizeId)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	g.Expect(kafka.DeprecatedKafkaStorageSize).To(gomega.Equal(instanceSize.MaxDataRetentionSize.String()))

	maxDataRetentionSizeBytes, err := instanceSize.MaxDataRetentionSize.ToInt64()
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(kafka.MaxDataRetentionSize.Bytes).To(gomega.Equal(maxDataRetentionSizeBytes))

	// When kafka is in 'Accepted' state it means that it still has not been
	// allocated to a cluster, which means that kas fleetshard-sync has not reported
	// yet any status, so the version attribute (actual version) at this point
	// should still be empty
	g.Expect(kafka.Version).To(gomega.Equal(""))
	g.Expect(kafka.BrowserUrl).To(gomega.Equal(fmt.Sprintf("%s%s/dashboard", test.TestServices.KafkaConfig.BrowserUrl, kafka.Id)))
	// 404 Not Found
	kafka, resp, _ = client.DefaultApi.GetKafkaById(ctx, fmt.Sprintf("not-%s", seedKafka.Id))
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNotFound))

	// A different account than the one used to create the Kafka instance, within
	// the same organization than the used account used to create the Kafka instance,
	// should be able to read the Kafka cluster
	account = h.NewRandAccount()
	ctx = h.NewAuthenticatedContext(account, nil)
	kafka, resp, _ = client.DefaultApi.GetKafkaById(ctx, seedKafka.Id)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(kafka.Id).NotTo(gomega.BeEmpty())

	// An account in a different organization than the one used to create the
	// Kafka instance should get a 404 Not Found.
	// this value is taken from config/quota-management-list-configuration.yaml
	anotherOrgID := "12147054"
	account = h.NewAccountWithNameAndOrg(faker.Name(), anotherOrgID)
	ctx = h.NewAuthenticatedContext(account, nil)
	_, resp, _ = client.DefaultApi.GetKafkaById(ctx, seedKafka.Id)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNotFound))

	// An allowed serviceaccount in config/quota-management-list-configuration.yaml
	// without an organization ID should get a 401 Unauthorized
	account = h.NewAllowedServiceAccount()
	ctx = h.NewAuthenticatedContext(account, nil)
	_, resp, _ = client.DefaultApi.GetKafkaById(ctx, seedKafka.Id)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusUnauthorized))
}

func TestKafka_Delete(t *testing.T) {
	g := gomega.NewWithT(t)

	owner := "test-user"

	orgId := "13640203"

	sampleKafkaID := api.NewID()

	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedGetClusterResponse, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	if err != nil {
		t.Fatalf(err.Error())
	}
	ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	userAccount := h.NewAccount(owner, "test-user", "test@gmail.com", orgId)

	userCtx := h.NewAuthenticatedContext(userAccount, nil)

	type args struct {
		ctx     context.Context
		kafkaID string
		async   bool
	}
	tests := []struct {
		name           string
		args           args
		verifyResponse func(resp *http.Response, err error)
	}{
		{
			name: "should fail when deleting kafka without async set to true",
			args: args{
				ctx:     userCtx,
				kafkaID: sampleKafkaID,
				async:   false,
			},
			verifyResponse: func(resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should fail when deleting kafka with empty id",
			args: args{
				ctx:     userCtx,
				kafkaID: "",
				async:   true,
			},
			verifyResponse: func(resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should succeed when deleting kafka with valid id and context",
			args: args{
				ctx:     userCtx,
				kafkaID: sampleKafkaID,
				async:   true,
			},
			verifyResponse: func(resp *http.Response, err error) {
				g.Expect(err).To(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusAccepted))
			},
		},
	}

	db := test.TestServices.DBFactory.New()
	// create a dummy cluster and assign a kafka to it
	cluster := &api.Cluster{
		Meta: api.Meta{
			ID: api.NewID(),
		},
		ClusterID:          api.NewID(),
		MultiAZ:            true,
		Region:             "baremetal",
		CloudProvider:      "baremetal",
		Status:             api.ClusterReady,
		IdentityProviderID: "some-id",
		ClusterDNS:         "some-cluster-dns",
		ProviderType:       api.ClusterProviderStandalone,
	}

	if err := db.Create(cluster).Error; err != nil {
		t.Error("failed to create dummy cluster")
		return
	}

	// create a kafka that will be updated
	kafka := &dbapi.KafkaRequest{
		Meta: api.Meta{
			ID: sampleKafkaID,
		},
		MultiAZ:               true,
		Owner:                 owner,
		Region:                "test",
		CloudProvider:         "test",
		Name:                  "test-kafka",
		OrganisationId:        orgId,
		Status:                constants.KafkaRequestStatusReady.String(),
		ClusterID:             cluster.ClusterID,
		ActualKafkaVersion:    "2.6.0",
		DesiredKafkaVersion:   "2.6.0",
		ActualStrimziVersion:  "v.23.0",
		DesiredStrimziVersion: "v0.23.0",
		InstanceType:          types.DEVELOPER.String(),
	}

	if err := db.Create(kafka).Error; err != nil {
		t.Errorf("failed to create Kafka db record due to error: %v", err)
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			client := test.NewApiClient(h)
			_, resp, err := client.DefaultApi.DeleteKafkaById(tt.args.ctx, tt.args.kafkaID, tt.args.async)
			if resp != nil {
				resp.Body.Close()
			}
			tt.verifyResponse(resp, err)
		})
	}
}

// TestKafkaDelete_DeleteDuringCreation - test deleting kafka instance during creation
func TestKafkaDelete_DeleteDuringCreation(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, func(ocmConfig *ocm.OCMConfig, c *config.DataplaneClusterConfig, reconcilerConfig *workers.ReconcilerConfig) {
		if ocmConfig.MockMode == ocm.MockModeEmulateServer {
			// increase repeat interval to allow time to delete the kafka instance before moving onto the next state
			// no need to reset this on teardown as it is always set at the start of each test within the registerIntegrationWithHooks setup for emulated servers.
			reconcilerConfig.ReconcilerRepeatInterval = 10 * time.Second
		}
		c.ClusterConfig = config.NewClusterConfig([]config.ManualCluster{test.NewMockDataplaneCluster(mockKafkaClusterName, 10)})
	})
	defer teardown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	// custom update kafka status implementation to ensure that kafkas created within this test never gets updated to a 'ready' state.
	mockKasFleetshardSyncBuilder.SetUpdateKafkaStatusFunc(func(helper *coreTest.Helper, privateClient *private.APIClient) error {
		dataplaneCluster, findDataplaneClusterErr := test.TestServices.ClusterService.FindCluster(services.FindClusterCriteria{
			Status: api.ClusterReady,
		})
		if findDataplaneClusterErr != nil {
			return fmt.Errorf("Unable to retrieve a ready data plane cluster: %v", findDataplaneClusterErr)
		}

		// only update kafka status if a data plane cluster is available
		if dataplaneCluster != nil {
			ctx, err := kasfleetshardsync.NewAuthenticatedContextForDataPlaneCluster(helper, dataplaneCluster.ClusterID)
			if err != nil {
				return err
			}

			kafkaList, resp, err := privateClient.AgentClustersApi.GetKafkas(ctx, dataplaneCluster.ClusterID)
			if resp != nil {
				resp.Body.Close()
			}
			if err != nil {
				return err
			}
			kafkaStatusList := make(map[string]private.DataPlaneKafkaStatus)
			for _, kafka := range kafkaList.Items {
				id := kafka.Metadata.Annotations.Bf2OrgId
				if kafka.Spec.Deleted {
					kafkaStatusList[id] = kasfleetshardsync.GetDeletedKafkaStatusResponse()
				}
			}

			if resp, err = privateClient.AgentClustersApi.UpdateKafkaClusterStatus(ctx, dataplaneCluster.ClusterID, kafkaStatusList); err != nil {
				return err
			} else {
				if resp != nil {
					resp.Body.Close()
				}
			}
		}
		return nil
	})
	mockKasFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasFleetshardSync.Start()
	defer mockKasFleetshardSync.Stop()

	clusterID, getClusterErr := common.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	k := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
	}

	// Test deletion of a kafka in an 'accepted' state
	kafka, resp, err := common.WaitForKafkaCreateToBeAccepted(ctx, test.TestServices.DBFactory, client, k)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error waiting for accepted kafka:  %v", err)
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusAccepted))
	g.Expect(kafka.Id).NotTo(gomega.BeEmpty(), "g.Expected ID assigned on creation")

	_, resp, err = client.DefaultApi.DeleteKafkaById(ctx, kafka.Id, true)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to delete kafka request: %v", err)

	_ = common.WaitForKafkaToBeDeleted(ctx, test.TestServices.DBFactory, client, kafka.Id)
	kafkaList, resp, err := client.DefaultApi.GetKafkas(ctx, &public.GetKafkasOpts{})
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to list kafka request: %v", err)
	g.Expect(kafkaList.Total).Should(gomega.BeZero(), " Kafka list response should be empty")

	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount, constants2.KafkaOperationDeprovision.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount, constants2.KafkaOperationDeprovision.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount, constants2.KafkaOperationDelete.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount, constants2.KafkaOperationDelete.String()))

	// Test deletion of a kafka in a 'preparing' state
	kafka, resp, err = common.WaitForKafkaCreateToBeAccepted(ctx, test.TestServices.DBFactory, client, k)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error waiting for accepted kafka:  %v", err)
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusAccepted))
	g.Expect(kafka.Id).NotTo(gomega.BeEmpty(), "g.Expected ID assigned on creation")

	kafka, err = common.WaitForKafkaToReachStatus(ctx, test.TestServices.DBFactory, client, kafka.Id, constants2.KafkaRequestStatusPreparing)
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error waiting for kafka request to be preparing: %v", err)

	_, resp, err = client.DefaultApi.DeleteKafkaById(ctx, kafka.Id, true)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to delete kafka request: %v", err)

	_ = common.WaitForKafkaToBeDeleted(ctx, test.TestServices.DBFactory, client, kafka.Id)
	kafkaList, resp, err = client.DefaultApi.GetKafkas(ctx, &public.GetKafkasOpts{})
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to list kafka request: %v", err)
	g.Expect(kafkaList.Total).Should(gomega.BeZero(), " Kafka list response should be empty")

	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 2", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount, constants2.KafkaOperationDeprovision.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 2", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount, constants2.KafkaOperationDeprovision.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 2", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount, constants2.KafkaOperationDelete.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 2", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount, constants2.KafkaOperationDelete.String()))

	// Test deletion of a kafka in a 'provisioning' state
	kafka, resp, err = common.WaitForKafkaCreateToBeAccepted(ctx, test.TestServices.DBFactory, client, k)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error waiting for accepted kafka:  %v", err)
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusAccepted))
	g.Expect(kafka.Id).NotTo(gomega.BeEmpty(), "g.Expected ID assigned on creation")

	kafka, err = common.WaitForKafkaToReachStatus(ctx, test.TestServices.DBFactory, client, kafka.Id, constants2.KafkaRequestStatusProvisioning)
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error waiting for kafka request to be provisioning: %v", err)

	_, resp, err = client.DefaultApi.DeleteKafkaById(ctx, kafka.Id, true)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to delete kafka request: %v", err)

	_ = common.WaitForKafkaToBeDeleted(ctx, test.TestServices.DBFactory, client, kafka.Id)
	kafkaList, resp, err = client.DefaultApi.GetKafkas(ctx, &public.GetKafkasOpts{})
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to list kafka request: %v", err)
	g.Expect(kafkaList.Total).Should(gomega.BeZero(), " Kafka list response should be empty")

	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 3", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount, constants2.KafkaOperationDeprovision.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 3", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount, constants2.KafkaOperationDeprovision.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 3", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount, constants2.KafkaOperationDelete.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 3", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount, constants2.KafkaOperationDelete.String()))
}

// TestKafkaDelete - tests fail kafka delete
func TestKafkaDelete_Fail(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	kafka := public.KafkaRequest{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
		MultiAz:       testMultiAZ,
		Id:            "invalid-8a41f783-b5e4-4692-a7cd-c0b9c8eeede9",
	}

	_, resp, err := client.DefaultApi.DeleteKafkaById(ctx, kafka.Id, true)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).To(gomega.HaveOccurred())
	// The id is invalid, so the metric is not g.Expected to exist
	common.CheckMetric(h, t, fmt.Sprintf("%s_%s", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount), false)
	common.CheckMetric(h, t, fmt.Sprintf("%s_%s", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount), false)
}

func TestKafka_DeleteAdminNonOwner(t *testing.T) {
	g := gomega.NewWithT(t)

	owner := "test-user"

	sampleKafkaID := api.NewID()

	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedGetClusterResponse, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	if err != nil {
		t.Fatalf(err.Error())
	}
	ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	otherUserAccount := h.NewAccountWithNameAndOrg("non-owner", "13640203")

	adminAccount := h.NewAccountWithNameAndOrg("admin", "13640203")

	otherUserCtx := h.NewAuthenticatedContext(otherUserAccount, nil)

	claims := jwt.MapClaims{
		"is_org_admin": true,
	}

	adminCtx := h.NewAuthenticatedContext(adminAccount, claims)

	type args struct {
		ctx     context.Context
		kafkaID string
	}
	tests := []struct {
		name           string
		args           args
		verifyResponse func(resp *http.Response, err error)
	}{
		{
			name: "should fail when deleting kafka by other user within the same org (not an owner)",
			args: args{
				ctx:     otherUserCtx,
				kafkaID: sampleKafkaID,
			},
			verifyResponse: func(resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should not fail when deleting kafka by other user within the same org (not an owner but org admin)",
			args: args{
				ctx:     adminCtx,
				kafkaID: sampleKafkaID,
			},
			verifyResponse: func(resp *http.Response, err error) {
				g.Expect(err).To(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusAccepted))
			},
		},
	}

	db := test.TestServices.DBFactory.New()
	// create a dummy cluster and assign a kafka to it
	cluster := &api.Cluster{
		Meta: api.Meta{
			ID: api.NewID(),
		},
		ClusterID:          api.NewID(),
		MultiAZ:            true,
		Region:             "baremetal",
		CloudProvider:      "baremetal",
		Status:             api.ClusterReady,
		IdentityProviderID: "some-id",
		ClusterDNS:         "some-cluster-dns",
		ProviderType:       api.ClusterProviderStandalone,
	}

	if err := db.Create(cluster).Error; err != nil {
		t.Error("failed to create dummy cluster")
		return
	}

	// create a kafka that will be updated
	kafka := &dbapi.KafkaRequest{
		Meta: api.Meta{
			ID: sampleKafkaID,
		},
		MultiAZ:               false,
		Owner:                 owner,
		Region:                "test",
		CloudProvider:         "test",
		Name:                  "test-kafka",
		OrganisationId:        "13640203",
		Status:                constants.KafkaRequestStatusReady.String(),
		ClusterID:             cluster.ClusterID,
		ActualKafkaVersion:    "2.6.0",
		DesiredKafkaVersion:   "2.6.0",
		ActualStrimziVersion:  "v.23.0",
		DesiredStrimziVersion: "v0.23.0",
		InstanceType:          types.STANDARD.String(),
	}

	if err := db.Create(kafka).Error; err != nil {
		t.Errorf("failed to create Kafka db record due to error: %v", err)
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			client := test.NewApiClient(h)
			_, resp, err := client.DefaultApi.DeleteKafkaById(tt.args.ctx, tt.args.kafkaID, true)
			if resp != nil {
				resp.Body.Close()
			}
			tt.verifyResponse(resp, err)
		})
	}
}

// TestKafkaList_Success tests getting kafka requests list
func TestKafkaList_Success(t *testing.T) {
	g := gomega.NewWithT(t)

	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, func(acl *quota_management.QuotaManagementListConfig, c *config.DataplaneClusterConfig) {
		c.ClusterConfig = config.NewClusterConfig([]config.ManualCluster{test.NewMockDataplaneCluster(mockKafkaClusterName, 1)})
	})
	defer teardown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	// setup pre-requisites to performing requests
	account := h.NewRandAccount()
	initCtx := h.NewAuthenticatedContext(account, nil)

	// get initial list (should be empty)
	initList, resp, err := client.DefaultApi.GetKafkas(initCtx, nil)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error occurred when attempting to list kafka requests:  %v", err)
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(initList.Items).To(gomega.BeEmpty(), "g.Expected empty kafka requests list")
	g.Expect(initList.Size).To(gomega.Equal(int32(0)), "g.Expected Size == 0")
	g.Expect(initList.Total).To(gomega.Equal(int32(0)), "g.Expected Total == 0")

	clusterID, getClusterErr := common.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}
	reauthenticationEnabled := true
	k := public.KafkaRequestPayload{
		Region:                  mocks.MockCluster.Region().ID(),
		CloudProvider:           mocks.MockCluster.CloudProvider().ID(),
		Name:                    mockKafkaName,
		ReauthenticationEnabled: &reauthenticationEnabled,
	}

	// POST kafka request to populate the list
	seedKafka, resp, err := client.DefaultApi.CreateKafka(initCtx, true, k)
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("failed to create seeded KafkaRequest: %s", err.Error())
	}

	// get populated list of kafka requests
	afterPostList, resp, err := client.DefaultApi.GetKafkas(initCtx, nil)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error occurred when attempting to list kafka requests:  %v", err)
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(len(afterPostList.Items)).To(gomega.Equal(1), "g.Expected kafka requests list length to be 1")
	g.Expect(afterPostList.Size).To(gomega.Equal(int32(1)), "g.Expected Size == 1")
	g.Expect(afterPostList.Total).To(gomega.Equal(int32(1)), "g.Expected Total == 1")

	// get kafka request item from the list
	listItem := afterPostList.Items[0]

	// check whether the seedKafka properties are the same as those from the kafka request list item
	g.Expect(seedKafka.Id).To(gomega.Equal(listItem.Id))
	g.Expect(seedKafka.Kind).To(gomega.Equal(listItem.Kind))
	g.Expect(listItem.Kind).To(gomega.Equal(presenters.KindKafka))
	g.Expect(seedKafka.Href).To(gomega.Equal(listItem.Href))
	g.Expect(seedKafka.Region).To(gomega.Equal(listItem.Region))
	g.Expect(listItem.Region).To(gomega.Equal(clusterservicetest.MockClusterRegion))
	g.Expect(seedKafka.CloudProvider).To(gomega.Equal(listItem.CloudProvider))
	g.Expect(listItem.CloudProvider).To(gomega.Equal(clusterservicetest.MockClusterCloudProvider))
	g.Expect(seedKafka.Name).To(gomega.Equal(listItem.Name))
	g.Expect(listItem.Name).To(gomega.Equal(mockKafkaName))
	g.Expect(listItem.Status).To(gomega.Equal(constants2.KafkaRequestStatusAccepted.String()))
	g.Expect(listItem.ReauthenticationEnabled).To(gomega.BeTrue())
	g.Expect(listItem.BrowserUrl).To(gomega.Equal(fmt.Sprintf("%s%s/dashboard", test.TestServices.KafkaConfig.BrowserUrl, listItem.Id)))

	// new account setup to prove that users can list kafkas instances created by a member of their org
	account = h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	// get populated list of kafka requests

	afterPostList, resp, err = client.DefaultApi.GetKafkas(ctx, nil)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error occurred when attempting to list kafka requests:  %v", err)
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(len(afterPostList.Items)).To(gomega.Equal(1), "g.Expected kafka requests list length to be 1")
	g.Expect(afterPostList.Size).To(gomega.Equal(int32(1)), "g.Expected Size == 1")
	g.Expect(afterPostList.Total).To(gomega.Equal(int32(1)), "g.Expected Total == 1")

	// get kafka request item from the list
	listItem = afterPostList.Items[0]

	// check whether the seedKafka properties are the same as those from the kafka request list item
	g.Expect(seedKafka.Id).To(gomega.Equal(listItem.Id))
	g.Expect(seedKafka.Kind).To(gomega.Equal(listItem.Kind))
	g.Expect(listItem.Kind).To(gomega.Equal(presenters.KindKafka))
	g.Expect(seedKafka.Href).To(gomega.Equal(listItem.Href))
	g.Expect(seedKafka.Region).To(gomega.Equal(listItem.Region))
	g.Expect(listItem.Region).To(gomega.Equal(clusterservicetest.MockClusterRegion))
	g.Expect(seedKafka.CloudProvider).To(gomega.Equal(listItem.CloudProvider))
	g.Expect(listItem.CloudProvider).To(gomega.Equal(clusterservicetest.MockClusterCloudProvider))
	g.Expect(seedKafka.Name).To(gomega.Equal(listItem.Name))
	g.Expect(listItem.Name).To(gomega.Equal(mockKafkaName))
	g.Expect(listItem.Status).To(gomega.Equal(constants2.KafkaRequestStatusAccepted.String()))

	// new account setup to prove that users can only list their own (the one they created and the one created by a member of their org) kafka instances
	// this value is taken from config/quota-management-list-configuration.yaml
	anotherOrgID := "13639843"
	account = h.NewAccount(h.NewID(), faker.Name(), faker.Email(), anotherOrgID)
	ctx = h.NewAuthenticatedContext(account, nil)

	// g.Expecting empty list for user that hasn't created any kafkas yet
	newUserList, resp, err := client.DefaultApi.GetKafkas(ctx, nil)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error occurred when attempting to list kafka requests:  %v", err)
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(len(newUserList.Items)).To(gomega.Equal(0), "g.Expected kafka requests list length to be 0")
	g.Expect(newUserList.Size).To(gomega.Equal(int32(0)), "g.Expected Size == 0")
	g.Expect(newUserList.Total).To(gomega.Equal(int32(0)), "g.Expected Total == 0")
}

// TestKafkaList_InvalidToken - tests listing kafkas with invalid token
func TestKafkaList_UnauthUser(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	_, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	// create empty context
	ctx := context.Background()

	kafkaRequests, resp, err := client.DefaultApi.GetKafkas(ctx, nil)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).To(gomega.HaveOccurred()) // g.Expecting an error here due unauthenticated user
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusUnauthorized))
	g.Expect(kafkaRequests.Items).To(gomega.BeNil())
	g.Expect(kafkaRequests.Size).To(gomega.Equal(int32(0)), "g.Expected Size == 0")
	g.Expect(kafkaRequests.Total).To(gomega.Equal(int32(0)), "g.Expected Total == 0")
}

func deleteTestKafka(t *testing.T, h *coreTest.Helper, ctx context.Context, client *public.APIClient, kafkaID string) {
	g := gomega.NewWithT(t)

	_, resp, err := client.DefaultApi.DeleteKafkaById(ctx, kafkaID, true)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to delete kafka request: %v", err)

	// wait for kafka to be deleted
	err = common.NewPollerBuilder(test.TestServices.DBFactory).
		OutputFunction(t.Logf).
		IntervalAndTimeout(kafkaCheckInterval, kafkaReadyTimeout).
		RetryLogMessagef("Waiting for kafka '%s' to be deleted", kafkaID).
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			db := test.TestServices.DBFactory.New()
			var kafkaRequest dbapi.KafkaRequest
			if err := db.Unscoped().Where("id = ?", kafkaID).First(&kafkaRequest).Error; err != nil {
				return false, err
			}
			return kafkaRequest.DeletedAt.Valid, nil
		}).Build().Poll()
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to delete kafka request: %v", err)
}

func TestKafkaList_IncorrectOCMIssuer_AuthzFailure(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	claims := jwt.MapClaims{
		"iss":      "invalidiss",
		"org_id":   account.Organization().ExternalID(),
		"username": account.Username(),
	}

	ctx := h.NewAuthenticatedContext(account, claims)

	_, resp, err := client.DefaultApi.GetKafkas(ctx, &public.GetKafkasOpts{})
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).Should(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusUnauthorized))
}

func TestKafkaList_CorrectTokenIssuer_AuthzSuccess(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	claims := jwt.MapClaims{
		"iss":      test.TestServices.ServerConfig.TokenIssuerURL,
		"org_id":   account.Organization().ExternalID(),
		"username": account.Username(),
	}

	ctx := h.NewAuthenticatedContext(account, claims)

	_, resp, err := client.DefaultApi.GetKafkas(ctx, &public.GetKafkasOpts{})
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
}

// TestKafka_RemovingExpiredKafkas_NoStandardInstances tests that all developer kafkas are removed after their allocated life span has expired.
// No standard instances are present
func TestKafka_RemovingExpiredKafkas_NoStandardInstances(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	clusterID, getClusterErr := common.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	// create dummy kafkas and assign it to user, at the end we'll verify that the kafka has been deleted
	kafkas := []*dbapi.KafkaRequest{
		mockkafka.BuildKafkaRequest(
			mockkafka.WithPredefinedTestValues(),
			mockkafka.WithMultiAZ(false),
			mockkafka.With(mockkafka.NAME, "dummy-kafka-to-remain"),
			mockkafka.With(mockkafka.STATUS, constants2.KafkaRequestStatusAccepted.String()),
			mockkafka.With(mockkafka.BOOTSTRAP_SERVER_HOST, ""),
			mockkafka.With(mockkafka.INSTANCE_TYPE, types.DEVELOPER.String()),
			mockkafka.With(mockkafka.CLUSTER_ID, clusterID),
		),
		mockkafka.BuildKafkaRequest(
			mockkafka.WithPredefinedTestValues(),
			mockkafka.WithCreatedAt(time.Now().Add(time.Duration(-49*time.Hour))),
			mockkafka.WithMultiAZ(false),
			mockkafka.With(mockkafka.CLUSTER_ID, clusterID),
			mockkafka.With(mockkafka.NAME, "dummy-kafka-2"),
			mockkafka.With(mockkafka.STATUS, constants2.KafkaRequestStatusProvisioning.String()),
			mockkafka.With(mockkafka.INSTANCE_TYPE, types.DEVELOPER.String()),
		),
		mockkafka.BuildKafkaRequest(
			mockkafka.WithPredefinedTestValues(),
			mockkafka.WithCreatedAt(time.Now().Add(time.Duration(-48*time.Hour))),
			mockkafka.WithMultiAZ(false),
			mockkafka.With(mockkafka.CLUSTER_ID, clusterID),
			mockkafka.With(mockkafka.NAME, "dummy-kafka-3"),
			mockkafka.With(mockkafka.INSTANCE_TYPE, types.DEVELOPER.String()),
		),
	}

	db := test.TestServices.DBFactory.New()
	if err := db.Create(&kafkas).Error; err != nil {
		g.Expect(err).NotTo(gomega.HaveOccurred())
		return
	}

	// also verify that any kafkas whose life has expired has been deleted.
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	kafkaDeletionErr := common.WaitForNumberOfKafkaToBeGivenCount(ctx, test.TestServices.DBFactory, client, 1)
	g.Expect(kafkaDeletionErr).NotTo(gomega.HaveOccurred(), "Error waiting for kafka deletion: %v", kafkaDeletionErr)
}

// TestKafka_RemovingExpiredKafkas_EmptyList tests that all developer kafkas are removed after their allocated life span has expired
// while standard instances are left behind
func TestKafka_RemovingExpiredKafkas_WithStandardInstances(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	clusterID, getClusterErr := common.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	// create dummy kafkas and assign it to user, at the end we'll verify that the kafka has been deleted
	kafkas := []*dbapi.KafkaRequest{
		mockkafka.BuildKafkaRequest(
			mockkafka.WithPredefinedTestValues(),
			mockkafka.WithMultiAZ(false),
			mockkafka.With(mockkafka.NAME, "dummy-kafka-not-yet-expired"),
			mockkafka.With(mockkafka.STATUS, constants2.KafkaRequestStatusAccepted.String()),
			mockkafka.With(mockkafka.BOOTSTRAP_SERVER_HOST, ""),
			mockkafka.With(mockkafka.INSTANCE_TYPE, types.DEVELOPER.String()),
			mockkafka.With(mockkafka.CLUSTER_ID, clusterID),
		),
		mockkafka.BuildKafkaRequest(
			mockkafka.WithPredefinedTestValues(),
			mockkafka.WithCreatedAt(time.Now().Add(time.Duration(-49*time.Hour))),
			mockkafka.WithMultiAZ(false),
			mockkafka.With(mockkafka.CLUSTER_ID, clusterID),
			mockkafka.With(mockkafka.NAME, "dummy-kafka-to-remove"),
			mockkafka.With(mockkafka.INSTANCE_TYPE, types.DEVELOPER.String()),
		),
		mockkafka.BuildKafkaRequest(
			mockkafka.WithPredefinedTestValues(),
			mockkafka.WithCreatedAt(time.Now().Add(time.Duration(-48*time.Hour))),
			mockkafka.WithMultiAZ(false),
			mockkafka.With(mockkafka.CLUSTER_ID, clusterID),
			mockkafka.With(mockkafka.NAME, "dummy-kafka-long-lived"),
			mockkafka.With(mockkafka.INSTANCE_TYPE, types.STANDARD.String()),
		),
	}

	db := test.TestServices.DBFactory.New()
	if err := db.Create(&kafkas).Error; err != nil {
		g.Expect(err).NotTo(gomega.HaveOccurred())
		return
	}

	// also verify that any kafkas whose life has expired has been deleted.
	account := h.NewRandAccount()
	account.Organization().ID()
	ctx := h.NewAuthenticatedContext(account, nil)

	kafkaDeletionErr := common.WaitForNumberOfKafkaToBeGivenCount(ctx, test.TestServices.DBFactory, client, 2)
	g.Expect(kafkaDeletionErr).NotTo(gomega.HaveOccurred(), "Error waiting for kafka deletion: %v", kafkaDeletionErr)
}

func getAndDeleteServiceAccounts(clientID string, env *environments.Env) error {
	var keycloakConfig *keycloak.KeycloakConfig
	env.MustResolve(&keycloakConfig)
	defer env.Cleanup()

	kcClientKafkaSre := keycloak.NewClient(keycloakConfig, keycloakConfig.OSDClusterIDPRealm)
	accessTokenKafkaSre, _ := kcClientKafkaSre.GetToken()

	client, err := kcClientKafkaSre.GetClient(clientID, accessTokenKafkaSre)
	if err != nil {
		return err
	}
	if client != nil {
		err = kcClientKafkaSre.DeleteClient(*client.ID, accessTokenKafkaSre)
	} else {
		kcClient := keycloak.NewClient(keycloakConfig, keycloakConfig.KafkaRealm)
		accessToken, _ := kcClient.GetToken()
		if client, _ := kcClient.GetClient(clientID, accessToken); client != nil {
			err = kcClient.DeleteClient(*client.ID, accessToken)
		}
	}
	return err
}
