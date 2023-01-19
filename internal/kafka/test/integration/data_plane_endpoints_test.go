package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	adminprivate "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services/kafka_tls_certificate_management"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/common"
	kafkamocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kasfleetshardsync"
	mocksupportedinstancetypes "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/supported_instance_types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"

	coreTest "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/golang-jwt/jwt/v4"
	"github.com/onsi/gomega"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/pkg/errors"
	"gopkg.in/resty.v1"
)

type TestServer struct {
	OcmServer     *httptest.Server
	TearDown      func()
	ClusterID     string
	Token         string
	Client        *public.APIClient
	PrivateClient *private.APIClient
	Helper        *coreTest.Helper
	Ctx           context.Context
}

type claimsFunc func(account *v1.Account, clusterId string, h *coreTest.Helper) jwt.MapClaims

var clusterId = api.NewID()
var clusterDNS = "some-cluster.dns.org"

func setup(t *testing.T, claims claimsFunc, startupHook interface{}) TestServer {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	h, client, tearDown := test.NewKafkaHelperWithHooks(t, ocmServer, startupHook)
	db := test.TestServices.DBFactory.New()
	// create a dummy cluster that will be used throughout the test
	cluster := &api.Cluster{
		Meta: api.Meta{
			ID: clusterId,
		},
		ClusterID:             clusterId,
		MultiAZ:               true,
		Region:                "baremetal",
		CloudProvider:         "baremetal",
		Status:                api.ClusterReady,
		IdentityProviderID:    "some-id",
		ClusterDNS:            clusterDNS,
		ProviderType:          api.ClusterProviderStandalone,
		SupportedInstanceType: api.AllInstanceTypeSupport.String(),
		ClientID:              fmt.Sprintf("kas-fleetshard-agent-%s", clusterId),
		ClientSecret:          "some-cluster-secret",
	}

	err := cluster.SetAvailableStrimziVersions(getTestStrimziVersionsMatrix())

	if err != nil {
		t.Error("failed to set available strimzi versions")
	}

	if err := db.Create(cluster).Error; err != nil {
		t.Fatalf("failed to create dummy cluster")
	}

	account := h.NewAllowedServiceAccount()
	ctx := h.NewAuthenticatedContext(account, claims(account, clusterId, h))
	token := h.CreateJWTStringWithClaim(account, claims(account, clusterId, h))

	config := private.NewConfiguration()
	config.BasePath = fmt.Sprintf("http://%s", test.TestServices.ServerConfig.BindAddress)
	config.DefaultHeader = map[string]string{
		"Authorization": "Bearer " + token,
	}
	privateClient := private.NewAPIClient(config)

	return TestServer{
		OcmServer:     ocmServer,
		Client:        client,
		PrivateClient: privateClient,
		TearDown: func() {
			ocmServer.Close()
			tearDown()
		},
		ClusterID: clusterId,
		Token:     token,
		Helper:    h,
		Ctx:       ctx,
	}
}

func TestDataPlaneEndpoints_AuthzSuccess(t *testing.T) {
	g := gomega.NewWithT(t)

	clusterId := "test-cluster-id"
	testServer := setup(t, func(account *v1.Account, cid string, h *coreTest.Helper) jwt.MapClaims {
		return jwt.MapClaims{
			"iss": test.TestServices.KeycloakConfig.SSOProviderRealm().ValidIssuerURI,
			"realm_access": map[string][]string{
				"roles": {"kas_fleetshard_operator"},
			},
			"kas-fleetshard-operator-cluster-id": clusterId,
		}
	}, nil)

	defer testServer.TearDown()

	body := map[string]private.DataPlaneKafkaStatus{
		testServer.ClusterID: {},
	}
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(testServer.Token).
		SetBody(body).
		Put(testServer.Helper.RestURL("/agent-clusters/" + clusterId + "/kafkas/status"))

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(restyResp.StatusCode()).To(gomega.Equal(http.StatusNotFound)) //the clusterId is not valid

	clusterStatusUpdateRequest := private.DataPlaneClusterUpdateStatusRequest{}
	restyResp, err = resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(testServer.Token).
		SetBody(clusterStatusUpdateRequest).
		Put(testServer.Helper.RestURL("/agent-clusters/" + clusterId + "/status"))

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(restyResp.StatusCode()).To(gomega.Equal(http.StatusNotFound)) //the clusterId is not valid
}

func TestDataPlaneEndpoints_AuthzFailWhenNoRealmRole(t *testing.T) {
	g := gomega.NewWithT(t)

	testServer := setup(t, func(account *v1.Account, cid string, h *coreTest.Helper) jwt.MapClaims {
		return jwt.MapClaims{
			"iss":                                test.TestServices.KeycloakConfig.SSOProviderRealm().ValidIssuerURI,
			"kas-fleetshard-operator-cluster-id": "test-cluster-id",
		}
	}, nil)

	defer testServer.TearDown()

	body := map[string]private.DataPlaneKafkaStatus{
		"test-cluster-id": {},
	}
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(testServer.Token).
		SetBody(body).
		Put(testServer.Helper.RestURL("/agent-clusters/" + testServer.ClusterID + "/kafkas/status"))

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(restyResp.StatusCode()).To(gomega.Equal(http.StatusNotFound))

	clusterStatusUpdateRequest := private.DataPlaneClusterUpdateStatusRequest{}
	restyResp, err = resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(testServer.Token).
		SetBody(clusterStatusUpdateRequest).
		Put(testServer.Helper.RestURL("/agent-clusters/" + testServer.ClusterID + "/status"))

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(restyResp.StatusCode()).To(gomega.Equal(http.StatusNotFound))
}

func TestDataPlaneEndpoints_AuthzFailWhenClusterIdNotMatch(t *testing.T) {
	g := gomega.NewWithT(t)

	testServer := setup(t, func(account *v1.Account, cid string, h *coreTest.Helper) jwt.MapClaims {
		return jwt.MapClaims{
			"iss": test.TestServices.KeycloakConfig.SSOProviderRealm().ValidIssuerURI,
			"realm_access": map[string][]string{
				"roles": {"kas_fleetshard_operator"},
			},
			"clientId": fmt.Sprintf("kas-fleetshard-agent-%s", "different-cluster-id"),
		}
	}, nil)
	defer testServer.TearDown()

	body := map[string]private.DataPlaneKafkaStatus{
		"test-cluster-id": {},
	}
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(testServer.Token).
		SetBody(body).
		Put(testServer.Helper.RestURL("/agent-clusters/" + testServer.ClusterID + "/kafkas/status"))

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(restyResp.StatusCode()).To(gomega.Equal(http.StatusNotFound))

	clusterStatusUpdateRequest := private.DataPlaneClusterUpdateStatusRequest{}
	restyResp, err = resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(testServer.Token).
		SetBody(clusterStatusUpdateRequest).
		Put(testServer.Helper.RestURL("/agent-clusters/" + testServer.ClusterID + "/status"))

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(restyResp.StatusCode()).To(gomega.Equal(http.StatusNotFound))
}

func TestDataPlaneEndpoints_GetManagedKafkas(t *testing.T) {
	g := gomega.NewWithT(t)

	testServer := setup(t, func(account *v1.Account, cid string, h *coreTest.Helper) jwt.MapClaims {
		username, _ := account.GetUsername()
		return jwt.MapClaims{
			"username": username,
			"iss":      test.TestServices.KeycloakConfig.SSOProviderRealm().ValidIssuerURI,
			"realm_access": map[string][]string{
				"roles": {"kas_fleetshard_operator"},
			},
			"clientId": fmt.Sprintf("kas-fleetshard-agent-%s", cid),
		}
	}, func(kafkaConfig *config.KafkaConfig, reconcilerConfig *workers.ReconcilerConfig) {
		// increase repeat interval so that the dummy kafkas created with a 'preparing' status does not get updated to the next state
		reconcilerConfig.ReconcilerRepeatInterval = 1 * time.Minute
	})
	defer testServer.TearDown()

	// the following kafkas are Expected to be returned by the endpoint
	validKafkas := []*dbapi.KafkaRequest{
		kafkamocks.BuildKafkaRequest(
			kafkamocks.WithPredefinedTestValues(),
			kafkamocks.With(kafkamocks.CLUSTER_ID, testServer.ClusterID),
			kafkamocks.With(kafkamocks.NAME, "test-kafka-1"),
			kafkamocks.With(kafkamocks.STATUS, constants.KafkaRequestStatusProvisioning.String()),
			kafkamocks.With(kafkamocks.INSTANCE_TYPE, types.DEVELOPER.String()),
		),

		kafkamocks.BuildKafkaRequest(
			kafkamocks.WithPredefinedTestValues(),
			kafkamocks.With(kafkamocks.CLUSTER_ID, testServer.ClusterID),
			kafkamocks.With(kafkamocks.NAME, "test-kafka-2"),
			kafkamocks.With(kafkamocks.STATUS, constants.KafkaRequestStatusReady.String()),
		),
		kafkamocks.BuildKafkaRequest(
			kafkamocks.WithPredefinedTestValues(),
			kafkamocks.With(kafkamocks.CLUSTER_ID, testServer.ClusterID),
			kafkamocks.With(kafkamocks.NAME, "test-kafka-3"),
			kafkamocks.With(kafkamocks.STATUS, constants.KafkaRequestStatusFailed.String()),
		),
		kafkamocks.BuildKafkaRequest(
			kafkamocks.WithPredefinedTestValues(),
			kafkamocks.With(kafkamocks.CLUSTER_ID, testServer.ClusterID),
			kafkamocks.With(kafkamocks.NAME, "test-kafka-4"),
			kafkamocks.With(kafkamocks.STATUS, constants.KafkaRequestStatusDeprovision.String()),
		),

		// suspending related kafkas
		kafkamocks.BuildKafkaRequest(
			kafkamocks.WithPredefinedTestValues(),
			kafkamocks.With(kafkamocks.CLUSTER_ID, testServer.ClusterID),
			kafkamocks.With(kafkamocks.NAME, "test-kafka-5"),
			kafkamocks.With(kafkamocks.STATUS, constants.KafkaRequestStatusSuspended.String()),
		),
		kafkamocks.BuildKafkaRequest(
			kafkamocks.WithPredefinedTestValues(),
			kafkamocks.With(kafkamocks.CLUSTER_ID, testServer.ClusterID),
			kafkamocks.With(kafkamocks.NAME, "test-kafka-6"),
			kafkamocks.With(kafkamocks.STATUS, constants.KafkaRequestStatusSuspending.String()),
		),
		kafkamocks.BuildKafkaRequest(
			kafkamocks.WithPredefinedTestValues(),
			kafkamocks.With(kafkamocks.CLUSTER_ID, testServer.ClusterID),
			kafkamocks.With(kafkamocks.NAME, "test-kafka-7"),
			kafkamocks.With(kafkamocks.STATUS, constants.KafkaRequestStatusResuming.String()),
		),
	}

	// the following kafkas should not be returned by the endpoint
	invalidKafkas := []*dbapi.KafkaRequest{
		// kafka that is already been deleted, 'deleted_at' is not empty
		kafkamocks.BuildKafkaRequest(
			kafkamocks.WithPredefinedTestValues(),
			kafkamocks.WithDeleted(true),
			kafkamocks.With(kafkamocks.CLUSTER_ID, testServer.ClusterID),
			kafkamocks.With(kafkamocks.NAME, "test-kafka-5"),
			kafkamocks.With(kafkamocks.STATUS, constants.KafkaRequestStatusDeprovision.String()),
		),
		// kafka that is in a preparing state
		kafkamocks.BuildKafkaRequest(
			kafkamocks.WithPredefinedTestValues(),
			kafkamocks.With(kafkamocks.CLUSTER_ID, testServer.ClusterID),
			kafkamocks.With(kafkamocks.NAME, "test-kafka-6"),
			kafkamocks.With(kafkamocks.STATUS, constants.KafkaRequestStatusPreparing.String()),
		),
		// kafka that failed during preparing
		kafkamocks.BuildKafkaRequest(
			kafkamocks.WithPredefinedTestValues(),
			kafkamocks.With(kafkamocks.CLUSTER_ID, testServer.ClusterID),
			kafkamocks.With(kafkamocks.NAME, "test-kafka-7"),
			kafkamocks.With(kafkamocks.STATUS, constants.KafkaRequestStatusFailed.String()),
			kafkamocks.With(kafkamocks.BOOTSTRAP_SERVER_HOST, ""),
		),
	}

	dummyKafkas := append(validKafkas, invalidKafkas...)

	// create dummy kafkas
	db := test.TestServices.DBFactory.New()
	if err := db.Create(&dummyKafkas).Error; err != nil {
		g.Expect(err).NotTo(gomega.HaveOccurred())
		return
	}

	list, resp, err := testServer.PrivateClient.AgentClustersApi.GetKafkas(testServer.Ctx, testServer.ClusterID)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(list.Items).To(gomega.HaveLen(7)) // only count valid Managed Kafka CR

	var kafkaConfig *config.KafkaConfig
	testServer.Helper.Env.MustResolve(&kafkaConfig)

	for _, k := range validKafkas {
		if mk := findManagedKafkaByID(list.Items, k.ID); mk != nil {
			instanceSize, err := kafkaConfig.GetKafkaInstanceSize(k.InstanceType, k.SizeId)
			if err != nil {
				t.Errorf("failed to retrieve instance size for kafka '%s': %v", mk.Metadata.Name, err.Error())
				break
			}
			g.Expect(mk.Metadata.Name).To(gomega.Equal(k.Name))
			g.Expect(mk.Metadata.Annotations.Bf2OrgPlacementId).To(gomega.Equal(k.PlacementId))
			g.Expect(mk.Metadata.Annotations.Bf2OrgId).To(gomega.Equal(k.ID))
			g.Expect(mk.Metadata.Labels.Bf2OrgKafkaInstanceProfileType).To(gomega.Equal(k.InstanceType))
			g.Expect(mk.Metadata.Labels.Bf2OrgKafkaInstanceProfileQuotaConsumed).To(gomega.Equal(strconv.Itoa(instanceSize.QuotaConsumed)))
			g.Expect(mk.Metadata.Labels.Bf2OrgSuspended).To(gomega.Equal(fmt.Sprintf("%t", arrays.Contains(constants.GetSuspendedStatuses(), k.Status))))
			g.Expect(mk.Metadata.Namespace).NotTo(gomega.BeEmpty())
			g.Expect(mk.Spec.Deleted).To(gomega.Equal(k.Status == constants.KafkaRequestStatusDeprovision.String()))
			g.Expect(mk.Spec.Versions.Kafka).To(gomega.Equal(k.DesiredKafkaVersion))
			g.Expect(mk.Spec.Versions.KafkaIbp).To(gomega.Equal(k.DesiredKafkaIBPVersion))
			g.Expect(mk.Spec.Endpoint.Tls).To(gomega.BeNil())
			g.Expect(mk.Spec.Capacity.IngressPerSec).To(gomega.Equal(instanceSize.IngressThroughputPerSec.String()))
			g.Expect(mk.Spec.Capacity.EgressPerSec).To(gomega.Equal(instanceSize.EgressThroughputPerSec.String()))
			g.Expect(mk.Spec.Capacity.TotalMaxConnections).To(gomega.Equal(int32(instanceSize.TotalMaxConnections)))
			g.Expect(mk.Spec.Capacity.MaxConnectionAttemptsPerSec).To(gomega.Equal(int32(instanceSize.MaxConnectionAttemptsPerSec)))
			g.Expect(mk.Spec.Capacity.MaxDataRetentionPeriod).To(gomega.Equal(instanceSize.MaxDataRetentionPeriod))
			g.Expect(mk.Spec.Capacity.MaxPartitions).To(gomega.Equal(int32(instanceSize.MaxPartitions)))
		} else {
			t.Error("failed matching managedkafka id with kafkarequest id")
			break
		}
	}
}

func TestDataPlaneEndpoints_UpdateManagedKafkas(t *testing.T) {
	g := gomega.NewWithT(t)

	testServer := setup(t, func(account *v1.Account, cid string, h *coreTest.Helper) jwt.MapClaims {
		username, _ := account.GetUsername()
		return jwt.MapClaims{
			"username": username,
			"iss":      test.TestServices.KeycloakConfig.SSOProviderRealm().ValidIssuerURI,
			"realm_access": map[string][]string{
				"roles": {"kas_fleetshard_operator"},
			},
			"clientId": fmt.Sprintf("kas-fleetshard-agent-%s", cid),
		}
	}, nil)
	defer testServer.TearDown()

	biggerStorageUpdateRequest := adminprivate.KafkaUpdateRequest{
		DeprecatedKafkaStorageSize: "70Gi",
	}

	var testKafkas = []*dbapi.KafkaRequest{
		kafkamocks.BuildKafkaRequest(
			kafkamocks.WithPredefinedTestValues(),
			kafkamocks.With(kafkamocks.CLUSTER_ID, testServer.ClusterID),
			kafkamocks.With(kafkamocks.NAME, "test-kafka-1"),
			kafkamocks.With(kafkamocks.STATUS, constants.KafkaRequestStatusProvisioning.String()),
			kafkamocks.With(kafkamocks.STORAGE_SIZE, mocksupportedinstancetypes.DefaultMaxDataRetentionSize),
			kafkamocks.With(kafkamocks.DESIRED_STRIMZI_VERSION, "strimzi-cluster-operator.v0.24.0-0"),
			kafkamocks.With(kafkamocks.DESIRED_KAFKA_VERSION, "2.8.1"),
			kafkamocks.With(kafkamocks.DESIRED_KAFKA_IBP_VERSION, "2.7.0"),
		),
		kafkamocks.BuildKafkaRequest(
			kafkamocks.WithPredefinedTestValues(),
			kafkamocks.With(kafkamocks.CLUSTER_ID, testServer.ClusterID),
			kafkamocks.With(kafkamocks.NAME, "test-kafka-2"),
			kafkamocks.With(kafkamocks.STATUS, constants.KafkaRequestStatusReady.String()),
			kafkamocks.With(kafkamocks.STORAGE_SIZE, mocksupportedinstancetypes.DefaultMaxDataRetentionSize),
			kafkamocks.With(kafkamocks.DESIRED_STRIMZI_VERSION, "strimzi-cluster-operator.v0.24.0-0"),
			kafkamocks.With(kafkamocks.ACTUAL_STRIMZI_VERSION, "strimzi-cluster-operator.v0.24.0-0"),
			kafkamocks.With(kafkamocks.DESIRED_KAFKA_VERSION, "2.8.1"),
			kafkamocks.With(kafkamocks.ACTUAL_KAFKA_VERSION, "2.8.1"),
			kafkamocks.With(kafkamocks.DESIRED_KAFKA_IBP_VERSION, "2.7.0"),
			kafkamocks.With(kafkamocks.ACTUAL_KAFKA_IBP_VERSION, "2.7.0"),
		),
		kafkamocks.BuildKafkaRequest(
			kafkamocks.WithPredefinedTestValues(),
			kafkamocks.With(kafkamocks.CLUSTER_ID, testServer.ClusterID),
			kafkamocks.With(kafkamocks.NAME, "test-kafka-3"),
			kafkamocks.With(kafkamocks.STATUS, constants.KafkaRequestStatusFailed.String()),
			kafkamocks.With(kafkamocks.STORAGE_SIZE, mocksupportedinstancetypes.DefaultMaxDataRetentionSize),
			kafkamocks.With(kafkamocks.DESIRED_STRIMZI_VERSION, "strimzi-cluster-operator.v0.24.0-0"),
			kafkamocks.With(kafkamocks.ACTUAL_STRIMZI_VERSION, "strimzi-cluster-operator.v0.24.0-0"),
			kafkamocks.With(kafkamocks.DESIRED_KAFKA_VERSION, "2.8.1"),
			kafkamocks.With(kafkamocks.ACTUAL_KAFKA_VERSION, "2.8.1"),
			kafkamocks.With(kafkamocks.DESIRED_KAFKA_IBP_VERSION, "2.7.0"),
			kafkamocks.With(kafkamocks.ACTUAL_KAFKA_IBP_VERSION, "2.7.0"),
		),
		kafkamocks.BuildKafkaRequest(
			kafkamocks.WithPredefinedTestValues(),
			kafkamocks.With(kafkamocks.CLUSTER_ID, testServer.ClusterID),
			kafkamocks.With(kafkamocks.NAME, "test-kafka-4"),
			kafkamocks.With(kafkamocks.STATUS, constants.KafkaRequestStatusDeprovision.String()),
			kafkamocks.With(kafkamocks.STORAGE_SIZE, mocksupportedinstancetypes.DefaultMaxDataRetentionSize),
			kafkamocks.With(kafkamocks.DESIRED_STRIMZI_VERSION, "strimzi-cluster-operator.v0.24.0-0"),
			kafkamocks.With(kafkamocks.ACTUAL_STRIMZI_VERSION, "strimzi-cluster-operator.v0.24.0-0"),
			kafkamocks.With(kafkamocks.DESIRED_KAFKA_VERSION, "2.8.1"),
			kafkamocks.With(kafkamocks.ACTUAL_KAFKA_VERSION, "2.8.1"),
			kafkamocks.With(kafkamocks.DESIRED_KAFKA_IBP_VERSION, "2.7.0"),
			kafkamocks.With(kafkamocks.ACTUAL_KAFKA_IBP_VERSION, "2.7.0"),
		),
	}

	// create dummy kafkas
	db := test.TestServices.DBFactory.New()
	if err := db.Create(&testKafkas).Error; err != nil {
		g.Expect(err).NotTo(gomega.HaveOccurred())
		return
	}

	// updating KafkaStorageSize, so that later it can be validated against "PrivateClient.AgentClustersApi.GetKafkas()"
	adminCtx := NewAuthenticatedContextForAdminEndpoints(testServer.Helper, []string{testFullRole})
	client := test.NewAdminPrivateAPIClient(testServer.Helper)
	for _, kafka := range testKafkas {
		result, resp, err := client.DefaultApi.UpdateKafkaById(adminCtx, kafka.ID, biggerStorageUpdateRequest)
		if resp != nil {
			resp.Body.Close()
		}
		g.Expect(err).To(gomega.BeNil())
		g.Expect(result.DeprecatedKafkaStorageSize).To(gomega.Equal(biggerStorageUpdateRequest.DeprecatedKafkaStorageSize))

		dataRetentionSizeQuantity := config.Quantity(biggerStorageUpdateRequest.DeprecatedKafkaStorageSize)
		dataRetentionSizeBytes, convErr := dataRetentionSizeQuantity.ToInt64()
		g.Expect(convErr).ToNot(gomega.HaveOccurred())
		g.Expect(result.MaxDataRetentionSize.Bytes).To(gomega.Equal(dataRetentionSizeBytes))
	}

	list, resp, err := testServer.PrivateClient.AgentClustersApi.GetKafkas(testServer.Ctx, testServer.ClusterID)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))

	var readyClusters, deletedClusters []string
	updates := map[string]private.DataPlaneKafkaStatus{}
	conditionsReasons := []string{
		"StrimziUpdating",
		"KafkaUpdating",
		"KafkaIbpUpdating",
	}
	lengthConditionsReasons := len(conditionsReasons)
	for idx, item := range list.Items {
		if !item.Spec.Deleted {
			updates[item.Metadata.Annotations.Bf2OrgId] = private.DataPlaneKafkaStatus{
				Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{{
					Type:   "Ready",
					Status: "True",
					Reason: conditionsReasons[idx%lengthConditionsReasons],
				}},
				Versions: private.DataPlaneKafkaStatusVersions{
					Kafka:    fmt.Sprintf("kafka-new-version-%s", item.Metadata.Annotations.Bf2OrgId),
					Strimzi:  fmt.Sprintf("strimzi-new-version-%s", item.Metadata.Annotations.Bf2OrgId),
					KafkaIbp: fmt.Sprintf("strimzi-ibp-new-version-%s", item.Metadata.Annotations.Bf2OrgId),
				},
				Routes: &[]private.DataPlaneKafkaStatusRoutes{
					{
						Name:   "test-route",
						Prefix: "",
						Router: clusterDNS,
					},
				},
			}
			readyClusters = append(readyClusters, item.Metadata.Annotations.Bf2OrgId)
		} else {
			updates[item.Metadata.Annotations.Bf2OrgId] = private.DataPlaneKafkaStatus{
				Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{{
					Type:   "Ready",
					Status: "False",
					Reason: "Deleted",
				}},
			}
			deletedClusters = append(deletedClusters, item.Metadata.Annotations.Bf2OrgId)
		}
	}

	// routes will be stored the first time status are updated
	resp, err = testServer.PrivateClient.AgentClustersApi.UpdateKafkaClusterStatus(testServer.Ctx, testServer.ClusterID, updates)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())

	// wait for the CNAMEs for routes to be created
	waitErr := common.NewPollerBuilder(test.TestServices.DBFactory).
		IntervalAndTimeout(1*time.Second, 1*time.Minute).
		RetryLogMessage("waiting for Kafka routes to be created").
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			c := &dbapi.KafkaRequest{}
			if err := db.First(c, "routes IS NOT NULL").Error; err != nil {
				return false, err
			}
			// if one route is created, it is safe to assume all routes are created
			return c.RoutesCreated, nil
		}).Build().Poll()
	g.Expect(waitErr).To(gomega.BeNil())

	// Send the requests again, this time the instances should be ready because routes are created
	resp, err = testServer.PrivateClient.AgentClustersApi.UpdateKafkaClusterStatus(testServer.Ctx, testServer.ClusterID, updates)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())

	for _, cid := range readyClusters {
		c := &dbapi.KafkaRequest{}
		if err := db.First(c, "id = ?", cid).Error; err != nil {
			t.Errorf("failed to find kafka cluster with id %s due to error: %v", cid, err)
		}

		sentUpdate, ok := updates[cid]
		if !ok {
			t.Errorf("failed to find sent kafka status update related to cluster with id %s", cid)
		}

		var sentReadyCondition string
		for _, cond := range sentUpdate.Conditions {
			if cond.Type == "Ready" {
				sentReadyCondition = cond.Reason
			}
		}
		g.Expect(sentReadyCondition).NotTo(gomega.BeEmpty())

		// Test version related reported fields
		g.Expect(c.Status).To(gomega.Equal(constants.KafkaRequestStatusReady.String()))
		g.Expect(c.ActualKafkaVersion).To(gomega.Equal(sentUpdate.Versions.Kafka))
		g.Expect(c.ActualKafkaIBPVersion).To(gomega.Equal(sentUpdate.Versions.KafkaIbp))
		g.Expect(c.ActualStrimziVersion).To(gomega.Equal(sentUpdate.Versions.Strimzi))
		g.Expect(c.StrimziUpgrading).To(gomega.Equal(sentReadyCondition == "StrimziUpdating"))
		g.Expect(c.KafkaUpgrading).To(gomega.Equal(sentReadyCondition == "KafkaUpdating"))
		g.Expect(c.KafkaIBPUpgrading).To(gomega.Equal(sentReadyCondition == "KafkaIbpUpdating"))

		// TODO test when kafka is being upgraded when kas fleet shard operator side
		// appropriately reports it
	}

	for _, cid := range deletedClusters {
		c := &dbapi.KafkaRequest{}
		// need to use Unscoped here as there is a chance the entry is soft deleted already
		if err := db.Unscoped().Where("id = ?", cid).First(c).Error; err != nil {
			t.Errorf("failed to find kafka cluster with id %s due to error: %v", cid, err)
		}
		g.Expect(c.Status).To(gomega.Equal(constants.KafkaRequestStatusDeleting.String()))
	}

	for _, cid := range readyClusters {
		// update the status to ready again and remove reason field to simulate the end of upgrade process as reported by kas-fleet-shard
		resp, err = testServer.PrivateClient.AgentClustersApi.UpdateKafkaClusterStatus(testServer.Ctx, testServer.ClusterID, map[string]private.DataPlaneKafkaStatus{
			cid: {
				Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{{
					Type:   "Ready",
					Status: "True",
				}},
				Versions: private.DataPlaneKafkaStatusVersions{
					Kafka:    fmt.Sprintf("kafka-new-version-%s", cid),
					KafkaIbp: fmt.Sprintf("kafka-ibp-new-version-%s", cid),
					Strimzi:  fmt.Sprintf("strimzi-new-version-%s", cid),
				},
			},
		})
		if resp != nil {
			resp.Body.Close()
		}
		g.Expect(err).NotTo(gomega.HaveOccurred())

		c := &dbapi.KafkaRequest{}
		if err := db.First(c, "id = ?", cid).Error; err != nil {
			t.Errorf("failed to find kafka cluster with id %s due to error: %v", cid, err)
		}

		// Make sure that the kafka stays in ready state and status of strimzi upgrade is false.
		g.Expect(c.Status).To(gomega.Equal(constants.KafkaRequestStatusReady.String()))
		g.Expect(c.StrimziUpgrading).To(gomega.BeFalse())
	}
}

func TestDataPlaneEndpoints_GetAndUpdateManagedKafkasWithTlsCerts(t *testing.T) {
	g := gomega.NewWithT(t)

	cert := "some-fake-cert"
	key := "some-fake-key"
	startHook := func(c *config.KafkaConfig, tlsConfig *kafka_tls_certificate_management.KafkaTLSCertificateManagementConfig) {
		tlsConfig.EnableKafkaExternalCertificate = true
		c.EnableKafkaCNAMERegistration = true
		tlsConfig.ManualCertificateManagementConfig.KafkaTLSCert = cert
		tlsConfig.ManualCertificateManagementConfig.KafkaTLSKey = key
	}
	testServer := setup(t, func(account *v1.Account, cid string, h *coreTest.Helper) jwt.MapClaims {
		username, _ := account.GetUsername()
		return jwt.MapClaims{
			"username": username,
			"iss":      test.TestServices.KeycloakConfig.SSOProviderRealm().ValidIssuerURI,
			"realm_access": map[string][]string{
				"roles": {"kas_fleetshard_operator"},
			},
			"clientId": fmt.Sprintf("kas-fleetshard-agent-%s", cid),
		}
	}, startHook)
	defer testServer.TearDown()
	bootstrapServerHost := "some-bootstrap⁻host"
	canaryServiceAccountClientId := "canary-servie-account-client-id"
	canaryServiceAccountClientSecret := "canary-service-account-client-secret"

	testKafka := &dbapi.KafkaRequest{
		ClusterID:                        testServer.ClusterID,
		MultiAZ:                          false,
		Name:                             mockKafkaName1,
		Status:                           constants.KafkaRequestStatusReady.String(),
		BootstrapServerHost:              bootstrapServerHost,
		CanaryServiceAccountClientID:     canaryServiceAccountClientId,
		CanaryServiceAccountClientSecret: canaryServiceAccountClientSecret,
		PlacementId:                      "some-placement-id",
		DesiredKafkaVersion:              "2.7.0",
		DesiredKafkaIBPVersion:           "2.7",
		InstanceType:                     types.DEVELOPER.String(),
		SizeId:                           "x1",
	}

	db := test.TestServices.DBFactory.New()

	// create dummy kafka
	if err := db.Save(testKafka).Error; err != nil {
		g.Expect(err).NotTo(gomega.HaveOccurred())
		return
	}

	list, resp, err := testServer.PrivateClient.AgentClustersApi.GetKafkas(testServer.Ctx, testServer.ClusterID)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(len(list.Items)).To(gomega.Equal(1)) // we should have one managed kafka cr

	if mk := findManagedKafkaByID(list.Items, testKafka.ID); mk != nil {
		g.Expect(mk.Spec.Endpoint.Tls.Cert).To(gomega.Equal(cert))
		g.Expect(mk.Spec.Endpoint.Tls.Key).To(gomega.Equal(key))
	} else {
		t.Error("failed matching managedkafka id with kafkarequest id")
	}
}

func TestDataPlaneEndpoints_GetAndUpdateManagedKafkasWithServiceAccounts(t *testing.T) {
	g := gomega.NewWithT(t)

	startHook := func(keycloakConfig *keycloak.KeycloakConfig) {
		keycloakConfig.EnableAuthenticationOnKafka = true
	}
	testServer := setup(t, func(account *v1.Account, cid string, h *coreTest.Helper) jwt.MapClaims {
		username, _ := account.GetUsername()
		return jwt.MapClaims{
			"username": username,
			"iss":      test.TestServices.KeycloakConfig.SSOProviderRealm().ValidIssuerURI,
			"realm_access": map[string][]string{
				"roles": {"kas_fleetshard_operator"},
			},
			"clientId": fmt.Sprintf("kas-fleetshard-agent-%s", cid),
		}
	}, startHook)
	defer testServer.TearDown()
	bootstrapServerHost := "some-bootstrap⁻host"
	canaryServiceAccountClientId := "canary-servie-account-client-id"
	canaryServiceAccountClientSecret := "canary-service-account-client-secret"

	testKafka := &dbapi.KafkaRequest{
		ClusterID:                        testServer.ClusterID,
		MultiAZ:                          false,
		Name:                             mockKafkaName1,
		Status:                           constants.KafkaRequestStatusReady.String(),
		BootstrapServerHost:              bootstrapServerHost,
		CanaryServiceAccountClientID:     canaryServiceAccountClientId,
		CanaryServiceAccountClientSecret: canaryServiceAccountClientSecret,
		PlacementId:                      "some-placement-id",
		DesiredKafkaVersion:              "2.7.0",
		DesiredKafkaIBPVersion:           "2.7",
		InstanceType:                     types.STANDARD.String(),
		SizeId:                           "x1",
	}

	db := test.TestServices.DBFactory.New()

	// create dummy kafka
	if err := db.Save(testKafka).Error; err != nil {
		g.Expect(err).NotTo(gomega.HaveOccurred())
		return
	}

	list, resp, err := testServer.PrivateClient.AgentClustersApi.GetKafkas(testServer.Ctx, testServer.ClusterID)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(len(list.Items)).To(gomega.Equal(1)) // we should have one managed kafka cr

	if mk := findManagedKafkaByID(list.Items, testKafka.ID); mk != nil {
		// check canary service account
		g.Expect(mk.Spec.ServiceAccounts).To(gomega.HaveLen(1))
		canaryServiceAccount := mk.Spec.ServiceAccounts[0]
		g.Expect(canaryServiceAccount.Name).To(gomega.Equal("canary"))
		g.Expect(canaryServiceAccount.Principal).To(gomega.Equal(canaryServiceAccountClientId))
		g.Expect(canaryServiceAccount.Password).To(gomega.Equal(canaryServiceAccountClientSecret))
	} else {
		t.Error("failed matching managedkafka id with kafkarequest id")
	}
}

func TestDataPlaneEndpoints_GetManagedKafkasWithoutOAuthTLSCert(t *testing.T) {
	g := gomega.NewWithT(t)

	startHook := func(c *keycloak.KeycloakConfig) {
		c.TLSTrustedCertificatesValue = ""
	}
	testServer := setup(t, func(account *v1.Account, cid string, h *coreTest.Helper) jwt.MapClaims {
		username, _ := account.GetUsername()
		return jwt.MapClaims{
			"username": username,
			"iss":      test.TestServices.KeycloakConfig.SSOProviderRealm().ValidIssuerURI,
			"realm_access": map[string][]string{
				"roles": {"kas_fleetshard_operator"},
			},
			"clientId": fmt.Sprintf("kas-fleetshard-agent-%s", cid),
		}
	}, startHook)
	defer testServer.TearDown()
	bootstrapServerHost := "some-bootstrap⁻host"

	testKafka := &dbapi.KafkaRequest{
		ClusterID:                        testServer.ClusterID,
		MultiAZ:                          false,
		Name:                             mockKafkaName1,
		Status:                           constants.KafkaRequestStatusReady.String(),
		BootstrapServerHost:              bootstrapServerHost,
		PlacementId:                      "some-placement-id",
		DesiredKafkaVersion:              "2.7.0",
		DesiredKafkaIBPVersion:           "2.7",
		InstanceType:                     types.STANDARD.String(),
		SizeId:                           "x1",
		CanaryServiceAccountClientID:     "some-fake-service-account-secret",
		CanaryServiceAccountClientSecret: "some-fake-canary-service-account-secret",
	}

	KeycloakConfig(testServer.Helper).EnableAuthenticationOnKafka = true

	db := test.TestServices.DBFactory.New()

	// create dummy kafka
	if err := db.Save(testKafka).Error; err != nil {
		g.Expect(err).NotTo(gomega.HaveOccurred())
		return
	}

	list, resp, err := testServer.PrivateClient.AgentClustersApi.GetKafkas(testServer.Ctx, testServer.ClusterID)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(len(list.Items)).To(gomega.Equal(1)) // we should have one managed kafka cr

	if mk := findManagedKafkaByID(list.Items, testKafka.ID); mk != nil {
		g.Expect(mk.Spec.Oauth.TlsTrustedCertificate).To(gomega.BeNil())
	} else {
		t.Error("failed matching managedkafka id with kafkarequest id")
	}
}

func TestDataPlaneEndpoints_GetManagedKafkasWithOauthMaximumSessionLifetime(t *testing.T) {
	g := gomega.NewWithT(t)

	startHook := func(c *keycloak.KeycloakConfig) {
		c.TLSTrustedCertificatesValue = ""
	}
	testServer := setup(t, func(account *v1.Account, cid string, h *coreTest.Helper) jwt.MapClaims {
		username, _ := account.GetUsername()
		return jwt.MapClaims{
			"username": username,
			"iss":      test.TestServices.KeycloakConfig.SSOProviderRealm().ValidIssuerURI,
			"realm_access": map[string][]string{
				"roles": {"kas_fleetshard_operator"},
			},
			"clientId": fmt.Sprintf("kas-fleetshard-agent-%s", cid),
		}
	}, startHook)
	defer testServer.TearDown()
	bootstrapServerHost := "some-bootstrap⁻host"

	testKafka := &dbapi.KafkaRequest{
		ClusterID:                        testServer.ClusterID,
		MultiAZ:                          false,
		Name:                             mockKafkaName1,
		Status:                           constants.KafkaRequestStatusReady.String(),
		BootstrapServerHost:              bootstrapServerHost,
		PlacementId:                      "some-placement-id",
		DesiredKafkaVersion:              "2.7.0",
		DesiredKafkaIBPVersion:           "2.7",
		InstanceType:                     types.STANDARD.String(),
		ReauthenticationEnabled:          true, // enable session reauthentication
		SizeId:                           "x1",
		CanaryServiceAccountClientID:     "some-fake-service-account-secret",
		CanaryServiceAccountClientSecret: "some-fake-canary-service-account-secret",
	}

	KeycloakConfig(testServer.Helper).EnableAuthenticationOnKafka = true

	db := test.TestServices.DBFactory.New()

	// create dummy kafka
	if err := db.Save(testKafka).Error; err != nil {
		g.Expect(err).NotTo(gomega.HaveOccurred())
		return
	}

	list, resp, err := testServer.PrivateClient.AgentClustersApi.GetKafkas(testServer.Ctx, testServer.ClusterID)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(len(list.Items)).To(gomega.Equal(1)) // we should have one managed kafka cr

	// check session lifetime value when reauthentication is enabled
	if mk := findManagedKafkaByID(list.Items, testKafka.ID); mk != nil {
		g.Expect(mk.Spec.Oauth.MaximumSessionLifetime).ToNot(gomega.BeNil())
		g.Expect(mk.Spec.Oauth.MaximumSessionLifetime).To(gomega.Equal(int64(299000)))
	} else {
		t.Error("failed matching managedkafka id with kafkarequest id")
	}

	// now disable and check that session lifetime is set to false for first kafka
	if err := db.Model(testKafka).UpdateColumn("reauthentication_enabled", false).Error; err != nil {
		g.Expect(err).NotTo(gomega.HaveOccurred())
		return
	}

	// create another dummy kafka
	anotherTestKafka := &dbapi.KafkaRequest{
		ClusterID:                        testServer.ClusterID,
		MultiAZ:                          false,
		Name:                             "another-kafka",
		Status:                           constants.KafkaRequestStatusReady.String(),
		BootstrapServerHost:              bootstrapServerHost,
		PlacementId:                      "some-placement-id",
		DesiredKafkaVersion:              "2.7.0",
		DesiredKafkaIBPVersion:           "2.7",
		InstanceType:                     types.STANDARD.String(),
		ReauthenticationEnabled:          true, // enable session reauthentication
		SizeId:                           "x1",
		CanaryServiceAccountClientID:     "some-fake-service-account-secret",
		CanaryServiceAccountClientSecret: "some-fake-canary-service-account-secret",
	}

	if err := db.Create(anotherTestKafka).Error; err != nil {
		g.Expect(err).NotTo(gomega.HaveOccurred())
		return
	}

	list, resp, err = testServer.PrivateClient.AgentClustersApi.GetKafkas(testServer.Ctx, testServer.ClusterID)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))

	// check session lifetime value when reauthentication is disabled
	if mk := findManagedKafkaByID(list.Items, testKafka.ID); mk != nil {
		g.Expect(mk.Spec.Oauth.MaximumSessionLifetime).ToNot(gomega.BeNil())
		g.Expect(mk.Spec.Oauth.MaximumSessionLifetime).To(gomega.Equal(int64(0)))
	} else {
		t.Fatalf("failed matching managedkafka id with kafkarequest id")
	}

	// check that session lifetime value is set
	if mk := findManagedKafkaByID(list.Items, anotherTestKafka.ID); mk != nil {
		g.Expect(mk.Spec.Oauth.MaximumSessionLifetime).ToNot(gomega.BeNil())
		g.Expect(mk.Spec.Oauth.MaximumSessionLifetime).To(gomega.Equal(int64(299000)))
	} else {
		t.Fatalf("failed matching managedkafka id with kafkarequest id")
	}

}

func TestDataPlaneEndpoints_UpdateManagedKafkasWithRoutesAndAdminApiServerUrl(t *testing.T) {
	g := gomega.NewWithT(t)

	testServer := setup(t, func(account *v1.Account, cid string, h *coreTest.Helper) jwt.MapClaims {
		username, _ := account.GetUsername()
		return jwt.MapClaims{
			"username": username,
			"iss":      test.TestServices.KeycloakConfig.SSOProviderRealm().ValidIssuerURI,
			"realm_access": map[string][]string{
				"roles": {"kas_fleetshard_operator"},
			},
			"clientId": fmt.Sprintf("kas-fleetshard-agent-%s", cid),
		}
	}, nil)
	defer testServer.TearDown()
	db := test.TestServices.DBFactory.New()
	var cluster api.Cluster
	if err := db.Where("cluster_id = ?", testServer.ClusterID).First(&cluster).Error; err != nil {
		g.Expect(err).NotTo(gomega.HaveOccurred())
		return
	}

	bootstrapServerHost := "prefix.some-bootstrap⁻host"
	adminApiServerUrl := "https://test-url.com"

	var testKafkas = []*dbapi.KafkaRequest{
		{
			ClusterID:                        testServer.ClusterID,
			MultiAZ:                          false,
			Name:                             mockKafkaName2,
			Status:                           constants.KafkaRequestStatusProvisioning.String(),
			BootstrapServerHost:              bootstrapServerHost,
			DesiredKafkaVersion:              "2.6.0",
			DesiredKafkaIBPVersion:           "2.6",
			InstanceType:                     types.DEVELOPER.String(),
			SizeId:                           "x1",
			CanaryServiceAccountClientID:     "some-fake-service-account-secret",
			CanaryServiceAccountClientSecret: "some-fake-canary-service-account-secret",
		},
	}

	// create dummy kafkas
	if err := db.Create(&testKafkas).Error; err != nil {
		g.Expect(err).NotTo(gomega.HaveOccurred())
		return
	}

	list, resp, err := testServer.PrivateClient.AgentClustersApi.GetKafkas(testServer.Ctx, testServer.ClusterID)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(len(list.Items)).To(gomega.Equal(1)) // only count valid Managed Kafka CR

	var readyClusters []string
	updates := map[string]private.DataPlaneKafkaStatus{}
	for _, item := range list.Items {
		updates[item.Metadata.Annotations.Bf2OrgId] = private.DataPlaneKafkaStatus{
			AdminServerURI: adminApiServerUrl,
			Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{{
				Type:   "Ready",
				Status: "True",
			}},
			Routes: &[]private.DataPlaneKafkaStatusRoutes{
				{
					Name:   "admin-api",
					Prefix: "admin-api",
					Router: fmt.Sprintf("router.%s", cluster.ClusterDNS),
				},
				{
					Name:   "bootstrap",
					Prefix: "",
					Router: fmt.Sprintf("router.%s", cluster.ClusterDNS),
				},
			},
		}
		readyClusters = append(readyClusters, item.Metadata.Annotations.Bf2OrgId)
	}

	// routes will be stored the first time status are updated
	resp, err = testServer.PrivateClient.AgentClustersApi.UpdateKafkaClusterStatus(testServer.Ctx, testServer.ClusterID, updates)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())

	// wait for the CNAMEs for routes to be created
	waitErr := common.NewPollerBuilder(test.TestServices.DBFactory).
		IntervalAndTimeout(1*time.Second, 1*time.Minute).
		RetryLogMessage("waiting for Kafka routes to be created").
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			c := &dbapi.KafkaRequest{}
			if err := db.First(c, "routes IS NOT NULL").Error; err != nil {
				return false, err
			}
			routes, err := c.GetRoutes()
			if err != nil {
				return false, err
			}
			if len(routes) != 2 {
				return false, errors.Errorf("Expected length of routes array to be 1")
			}
			wantDomain1 := "admin-api-prefix.some-bootstrap⁻host"
			if routes[0].Domain != wantDomain1 {
				return false, errors.Errorf("route domain value is %s but want %s", routes[0].Domain, wantDomain1)
			}
			wantDomain2 := "prefix.some-bootstrap⁻host"
			if routes[1].Domain != wantDomain2 {
				return false, errors.Errorf("route domain value is %s but want %s", routes[1].Domain, wantDomain2)
			}

			// if one route is created, it is safe to assume all routes are created
			return c.RoutesCreated, nil
		}).Build().Poll()

	g.Expect(waitErr).NotTo(gomega.HaveOccurred())

	// Send the requests again, this time the instances should be ready because routes are created
	resp, err = testServer.PrivateClient.AgentClustersApi.UpdateKafkaClusterStatus(testServer.Ctx, testServer.ClusterID, updates)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())

	for _, cid := range readyClusters {
		c := &dbapi.KafkaRequest{}
		if err := db.First(c, "id = ?", cid).Error; err != nil {
			t.Errorf("failed to find kafka cluster with id %s due to error: %v", cid, err)
		}
		g.Expect(c.Status).To(gomega.Equal(constants.KafkaRequestStatusReady.String()))
		g.Expect(c.AdminApiServerURL).To(gomega.Equal(adminApiServerUrl))
	}
}

func TestDataPlaneEndpoints_GetManagedKafkasWithOAuthTLSCert(t *testing.T) {
	g := gomega.NewWithT(t)

	cert := "some-fake-cert"
	startHook := func(c *keycloak.KeycloakConfig) {
		c.TLSTrustedCertificatesValue = cert
		c.EnableAuthenticationOnKafka = true
	}
	testServer := setup(t, func(account *v1.Account, cid string, h *coreTest.Helper) jwt.MapClaims {
		username, _ := account.GetUsername()
		return jwt.MapClaims{
			"username": username,
			"iss":      test.TestServices.KeycloakConfig.SSOProviderRealm().ValidIssuerURI,
			"realm_access": map[string][]string{
				"roles": {"kas_fleetshard_operator"},
			},
			"clientId": fmt.Sprintf("kas-fleetshard-agent-%s", cid),
		}
	}, startHook)
	defer testServer.TearDown()
	bootstrapServerHost := "some-bootstrap⁻host"

	testKafka := &dbapi.KafkaRequest{
		ClusterID:              testServer.ClusterID,
		MultiAZ:                false,
		Name:                   mockKafkaName1,
		Status:                 constants.KafkaRequestStatusReady.String(),
		BootstrapServerHost:    bootstrapServerHost,
		PlacementId:            "some-placement-id",
		DesiredKafkaVersion:    "2.7.0",
		DesiredKafkaIBPVersion: "2.7",
		InstanceType:           types.STANDARD.String(),
		SizeId:                 "x1",

		CanaryServiceAccountClientID:     "some-fake-service-account-secret",
		CanaryServiceAccountClientSecret: "some-fake-canary-service-account-secret",
	}

	db := test.TestServices.DBFactory.New()

	// create dummy kafka
	if err := db.Save(testKafka).Error; err != nil {
		g.Expect(err).NotTo(gomega.HaveOccurred())
		return
	}

	list, resp, err := testServer.PrivateClient.AgentClustersApi.GetKafkas(testServer.Ctx, testServer.ClusterID)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(len(list.Items)).To(gomega.Equal(1)) // we should have one managed kafka cr

	if mk := findManagedKafkaByID(list.Items, testKafka.ID); mk != nil {
		g.Expect(mk.Spec.Oauth.TlsTrustedCertificate).ToNot(gomega.BeNil())
	} else {
		t.Error("failed matching managedkafka id with kafkarequest id")
	}

}

func KeycloakConfig(helper *coreTest.Helper) (c *keycloak.KeycloakConfig) {
	helper.Env.MustResolveAll(&c)
	return
}

func TestDataPlaneEndpoints_UpdateManagedKafkaWithErrorStatus(t *testing.T) {
	g := gomega.NewWithT(t)

	testServer := setup(t, func(account *v1.Account, cid string, h *coreTest.Helper) jwt.MapClaims {
		username, _ := account.GetUsername()
		return jwt.MapClaims{
			"username": username,
			"iss":      test.TestServices.KeycloakConfig.SSOProviderRealm().ValidIssuerURI,
			"realm_access": map[string][]string{
				"roles": {"kas_fleetshard_operator"},
			},
			"clientId": fmt.Sprintf("kas-fleetshard-agent-%s", cid),
		}
	}, nil)
	defer testServer.TearDown()
	bootstrapServerHost := "some-bootstrap⁻host"

	db := test.TestServices.DBFactory.New()

	testKafka := dbapi.KafkaRequest{
		ClusterID:              testServer.ClusterID,
		MultiAZ:                false,
		Name:                   mockKafkaName1,
		Status:                 constants.KafkaRequestStatusReady.String(),
		BootstrapServerHost:    bootstrapServerHost,
		DesiredKafkaVersion:    "2.7.0",
		DesiredKafkaIBPVersion: "2.7",
		InstanceType:           types.STANDARD.String(),
		SizeId:                 "x1",
	}

	// create dummy kafkas
	if err := db.Create(&testKafka).Error; err != nil {
		g.Expect(err).NotTo(gomega.HaveOccurred())
		return
	}

	list, resp, err := testServer.PrivateClient.AgentClustersApi.GetKafkas(testServer.Ctx, testServer.ClusterID)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(len(list.Items)).To(gomega.Equal(1)) // we should have one managed kafka cr
	kafkaReqID := list.Items[0].Metadata.Annotations.Bf2OrgId

	errMessage := "test-err-message"
	updateReq := map[string]private.DataPlaneKafkaStatus{
		kafkaReqID: kasfleetshardsync.GetErrorWithCustomMessageKafkaStatusResponse(errMessage),
	}
	resp, err = testServer.PrivateClient.AgentClustersApi.UpdateKafkaClusterStatus(testServer.Ctx, testServer.ClusterID, updateReq)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())

	c := &dbapi.KafkaRequest{}
	if err := db.First(c, "id = ?", kafkaReqID).Error; err != nil {
		t.Errorf("failed to find kafka cluster with id %s due to error: %v", kafkaReqID, err)
	}
	g.Expect(c.Status).To(gomega.Equal(constants.KafkaRequestStatusFailed.String()))
	g.Expect(c.FailedReason).To(gomega.Equal("Kafka reported as failed from the data plane"))
}

func TestDataPlaneEndpoints_UpdateManagedKafka_RemoveFailedReason(t *testing.T) {
	g := gomega.NewWithT(t)

	testServer := setup(t, func(account *v1.Account, cid string, h *coreTest.Helper) jwt.MapClaims {
		username, _ := account.GetUsername()
		return jwt.MapClaims{
			"username": username,
			"iss":      test.TestServices.KeycloakConfig.SSOProviderRealm().ValidIssuerURI,
			"realm_access": map[string][]string{
				"roles": {"kas_fleetshard_operator"},
			},
			"clientId": fmt.Sprintf("kas-fleetshard-agent-%s", cid),
		}
	}, nil)
	defer testServer.TearDown()
	bootstrapServerHost := "some-bootstrap⁻host"

	db := test.TestServices.DBFactory.New()

	testKafka := dbapi.KafkaRequest{
		ClusterID:              testServer.ClusterID,
		MultiAZ:                false,
		Name:                   mockKafkaName1,
		Status:                 constants.KafkaRequestStatusFailed.String(),
		BootstrapServerHost:    bootstrapServerHost,
		DesiredKafkaVersion:    "2.7.0",
		DesiredKafkaIBPVersion: "2.7",
		FailedReason:           "test failed reason",
		RoutesCreated:          true,
		InstanceType:           types.STANDARD.String(),
		SizeId:                 "x1",
	}

	// create dummy kafkas
	if err := db.Create(&testKafka).Error; err != nil {
		g.Expect(err).NotTo(gomega.HaveOccurred())
		return
	}

	list, resp, err := testServer.PrivateClient.AgentClustersApi.GetKafkas(testServer.Ctx, testServer.ClusterID)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(len(list.Items)).To(gomega.Equal(1)) // we should have one managed kafka cr
	kafkaReqID := list.Items[0].Metadata.Annotations.Bf2OrgId

	updateReq := map[string]private.DataPlaneKafkaStatus{
		kafkaReqID: kasfleetshardsync.GetReadyKafkaStatusResponse(clusterDNS),
	}
	resp, err = testServer.PrivateClient.AgentClustersApi.UpdateKafkaClusterStatus(testServer.Ctx, testServer.ClusterID, updateReq)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())

	c := &dbapi.KafkaRequest{}
	if err := db.First(c, "id = ?", kafkaReqID).Error; err != nil {
		t.Errorf("failed to find kafka cluster with id %s due to error: %v", kafkaReqID, err)
	}
	g.Expect(c.Status).To(gomega.Equal(constants.KafkaRequestStatusReady.String()))
	g.Expect(c.FailedReason).To(gomega.BeEmpty())
}

func TestDataPlaneEndpoints_UpdateManagedKafka_PrewarmingStatusInfoMetrics(t *testing.T) {
	g := gomega.NewWithT(t)

	testServer := setup(t, func(account *v1.Account, cid string, h *coreTest.Helper) jwt.MapClaims {
		username, _ := account.GetUsername()
		return jwt.MapClaims{
			"username": username,
			"iss":      test.TestServices.KeycloakConfig.SSOProviderRealm().ValidIssuerURI,
			"realm_access": map[string][]string{
				"roles": {"kas_fleetshard_operator"},
			},
			"clientId": fmt.Sprintf("kas-fleetshard-agent-%s", cid),
		}
	}, nil)

	defer testServer.TearDown()

	updateReq := map[string]private.DataPlaneKafkaStatus{
		"reserved-kafka-standard-1": {
			Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{
				{
					Type:   "Ready",
					Reason: "Error",
					Status: "False",
				},
			},
		},
		"reserved-kafka-developer-1": {
			Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{
				{
					Type:   "Ready",
					Reason: "",
					Status: "False",
				},
			},
		},
		"reserved-kafka-developer-2": {
			Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{
				{
					Type:   "Ready",
					Reason: "Error",
					Status: "False",
				},
			},
		},
		"reserved-kafka-standard-2": {
			Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{
				{
					Type:   "Ready",
					Status: "True",
				},
			},
		},
		"reserved-kafka-developer-3": {
			Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{
				{
					Type:   "Ready",
					Reason: "",
					Status: "Unknown",
				},
			},
		},
	}
	resp, err := testServer.PrivateClient.AgentClustersApi.UpdateKafkaClusterStatus(testServer.Ctx, testServer.ClusterID, updateReq)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())

	prewarmedCountForStandard := map[string]int{
		"error":               1,
		"ready":               1,
		"unknown":             0,
		"deleted":             0,
		"rejected":            0,
		"installing":          0,
		"rejectedClusterFull": 0,
	}
	prewarmedCountForDeveloper := map[string]int{
		"error":               1,
		"ready":               0,
		"deleted":             0,
		"unknown":             1,
		"rejected":            0,
		"installing":          1,
		"rejectedClusterFull": 0,
	}

	// now verify that the metric value have been published
	for status, count := range prewarmedCountForStandard {
		metricValue := fmt.Sprintf("%d", count)
		checkMetricsError := common.WaitForMetricToBePresent(testServer.Helper, t, metrics.PrewarmingStatusInfoCount, metricValue, api.StandardTypeSupport.String(), testServer.ClusterID, status)
		g.Expect(checkMetricsError).NotTo(gomega.HaveOccurred())
	}
	for status, count := range prewarmedCountForDeveloper {
		metricValue := fmt.Sprintf("%d", count)
		checkMetricsError := common.WaitForMetricToBePresent(testServer.Helper, t, metrics.PrewarmingStatusInfoCount, metricValue, api.DeveloperTypeSupport.String(), testServer.ClusterID, status)
		g.Expect(checkMetricsError).NotTo(gomega.HaveOccurred())
	}
}

func findManagedKafkaByID(slice []private.ManagedKafka, kafkaId string) *private.ManagedKafka {
	match := func(item private.ManagedKafka) bool { return item.Metadata.Annotations.Bf2OrgId == kafkaId }
	for _, item := range slice {
		if match(item) {
			return &item
		}
	}
	return nil
}

func TestDataPlaneEndpoints_ReassignRejectedKafkaDueToInsufficientResources(t *testing.T) {
	g := gomega.NewWithT(t)
	region := "us-east-1"
	cloudProvider := "aws"

	standardClusterID := "standard"
	developerClusterID := "developer"

	standardClusterDNS := "standard-cluster-dns.org"
	developerClusterDNS := "developer-cluster-dns.org"

	configHook := func(clusterConfig *config.DataplaneClusterConfig, reconcilerConfig *workers.ReconcilerConfig) {
		reconcilerConfig.ReconcilerRepeatInterval = 1 * time.Second
		clusterConfig.DataPlaneClusterScalingType = config.ManualScaling
		clusterConfig.EnableReadyDataPlaneClustersReconcile = false
		clusterConfig.ClusterConfig = config.NewClusterConfig(config.ClusterList{
			config.ManualCluster{ClusterId: standardClusterID, ClusterDNS: standardClusterDNS, Status: api.ClusterReady, KafkaInstanceLimit: 1, Region: region, MultiAZ: true, CloudProvider: cloudProvider, Schedulable: true, SupportedInstanceType: "standard", ProviderType: api.ClusterProviderStandalone},
			config.ManualCluster{ClusterId: developerClusterID, ClusterDNS: developerClusterDNS, Status: api.ClusterReady, KafkaInstanceLimit: 1, Region: region, MultiAZ: false, CloudProvider: cloudProvider, Schedulable: true, SupportedInstanceType: "developer", ProviderType: api.ClusterProviderStandalone},
		})
	}
	testServer := setup(t, func(account *v1.Account, cid string, h *coreTest.Helper) jwt.MapClaims {
		username, _ := account.GetUsername()
		return jwt.MapClaims{
			"username": username,
			"iss":      test.TestServices.KeycloakConfig.SSOProviderRealm().ValidIssuerURI,
			"realm_access": map[string][]string{
				"roles": {"kas_fleetshard_operator"},
			},
			"clientId": fmt.Sprintf("kas-fleetshard-agent-%s", cid),
		}
	}, configHook)

	kasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(testServer.Helper, t)
	kasFleetshardSyncBuilder.SetUpdateKafkaStatusFunc(func(helper *coreTest.Helper, privateClient *private.APIClient) error {
		// We don't need to update the status of the kafka as it will be updated by the test itself
		return nil
	})

	kasFleetshardSyncBuilder.SetUpdateDataplaneClusterStatusFunc(func(helper *coreTest.Helper, privateClient *private.APIClient, ocmClient ocm.Client) error {
		clusters, err := test.TestServices.ClusterService.FindAllClusters(services.FindClusterCriteria{
			Status: api.ClusterReady,
		})

		if err != nil {
			return err
		}

		for _, cluster := range clusters {
			ctx, err := kasfleetshardsync.NewAuthenticatedContextForDataPlaneCluster(helper, cluster.ClusterID)
			if err != nil {
				return err
			}
			clusterStatusUpdateRequest := kasfleetshardsync.SampleDataPlaneclusterStatusRequestWithAvailableCapacity()
			resp, err := privateClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, cluster.ClusterID, *clusterStatusUpdateRequest)
			resp.Body.Close()
			if err != nil {
				return fmt.Errorf("failed to update cluster status via agent endpoint: %v", err)
			}
		}

		return nil
	})

	kasFleetshardSync := kasFleetshardSyncBuilder.Build()
	kasFleetshardSync.Start()

	t.Cleanup(func() {
		kasFleetshardSync.Stop()
		testServer.TearDown()
	})

	db := test.TestServices.DBFactory.New()
	var cluster api.Cluster
	if err := db.Where("cluster_id = ?", testServer.ClusterID).First(&cluster).Error; err != nil {
		g.Expect(err).NotTo(gomega.HaveOccurred())
		return
	}
	bootstrapServerHost := "prefix.some-bootstrap⁻host"
	adminApiServerUrl := "https://test-url.com"

	var testKafkas = []*dbapi.KafkaRequest{
		{
			ClusterID:              testServer.ClusterID,
			MultiAZ:                false,
			Name:                   mockKafkaName1,
			CloudProvider:          cloudProvider,
			Region:                 region,
			Status:                 constants.KafkaRequestStatusProvisioning.String(),
			BootstrapServerHost:    bootstrapServerHost,
			DesiredKafkaVersion:    "2.6.0",
			DesiredKafkaIBPVersion: "2.6",
			DesiredStrimziVersion:  "2.6",
			InstanceType:           types.DEVELOPER.String(),
			SizeId:                 "x1",
		},
		{
			ClusterID:              testServer.ClusterID,
			MultiAZ:                true,
			Name:                   mockKafkaName2,
			CloudProvider:          cloudProvider,
			Region:                 region,
			Status:                 constants.KafkaRequestStatusProvisioning.String(),
			BootstrapServerHost:    bootstrapServerHost,
			DesiredKafkaVersion:    "2.5.0",
			DesiredKafkaIBPVersion: "2.5",
			DesiredStrimziVersion:  "2.5",
			InstanceType:           types.STANDARD.String(),
			SizeId:                 "x1",
		},
	}

	// create dummy kafkas
	if err := db.Create(&testKafkas).Error; err != nil {
		g.Expect(err).NotTo(gomega.HaveOccurred())
		return
	}

	list, resp, err := testServer.PrivateClient.AgentClustersApi.GetKafkas(testServer.Ctx, testServer.ClusterID)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(list.Items).To(gomega.HaveLen(2))

	var provisioningKafkas []string
	updates := map[string]private.DataPlaneKafkaStatus{}
	for _, item := range list.Items {
		updates[item.Metadata.Annotations.Bf2OrgId] = private.DataPlaneKafkaStatus{
			AdminServerURI: adminApiServerUrl,
			Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{{
				Type:    "Ready",
				Reason:  "Rejected",
				Message: "Cluster has insufficient resources",
			}},
		}
		provisioningKafkas = append(provisioningKafkas, item.Metadata.Annotations.Bf2OrgId)
	}

	resp, err = testServer.PrivateClient.AgentClustersApi.UpdateKafkaClusterStatus(testServer.Ctx, testServer.ClusterID, updates)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())

	// Check that the kafkas have been unassigned from the current cluster.
	for _, cid := range provisioningKafkas {
		c := &dbapi.KafkaRequest{}
		err := db.First(c, "id = ?", cid).Error
		g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to find kafka cluster with id %s due to error: %v", cid, err)
		g.Expect(c.Status).To(gomega.Equal(constants.KafkaRequestStatusProvisioning.String()))
		g.Expect(c.ClusterID).To(gomega.BeEmpty())
		g.Expect(c.BootstrapServerHost).To(gomega.BeEmpty())
		g.Expect(c.DesiredKafkaVersion).To(gomega.BeEmpty())
		g.Expect(c.DesiredKafkaIBPVersion).To(gomega.BeEmpty())
		g.Expect(c.DesiredStrimziVersion).To(gomega.BeEmpty())
	}

	// Check that the developer kafka has been reassigned into a developer data plane cluster.
	developerKafka, err := common.WaitForKafkaClusterIDToBeAssigned(testServer.Helper.DBFactory(), mockKafkaName1)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(developerKafka.ClusterID).To(gomega.Equal(developerClusterID))
	g.Expect(developerKafka.BootstrapServerHost).To(gomega.HaveSuffix(developerClusterDNS))
	g.Expect(developerKafka.DesiredKafkaVersion).ToNot(gomega.BeEmpty())
	g.Expect(developerKafka.DesiredKafkaIBPVersion).ToNot(gomega.BeEmpty())
	g.Expect(developerKafka.DesiredStrimziVersion).ToNot(gomega.BeEmpty())

	// Check that the standard kafka has been reassigned into a standard data plane cluster.
	standardKafka, err := common.WaitForKafkaClusterIDToBeAssigned(testServer.Helper.DBFactory(), mockKafkaName2)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(standardKafka.ClusterID).To(gomega.Equal(standardClusterID))
	g.Expect(standardKafka.BootstrapServerHost).To(gomega.HaveSuffix(standardClusterDNS))
	g.Expect(standardKafka.DesiredKafkaVersion).ToNot(gomega.BeEmpty())
	g.Expect(standardKafka.DesiredKafkaIBPVersion).ToNot(gomega.BeEmpty())
	g.Expect(standardKafka.DesiredStrimziVersion).ToNot(gomega.BeEmpty())
}
