package integration

import (
	"fmt"
	"testing"

	kafkaconstants "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	kafkatest "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/common"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kasfleetshardsync"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"

	"github.com/onsi/gomega"
)

func TestAdminKafka_KafkaSuspension(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	clusterList := config.ClusterList{
		{
			Name:                  "test-cluster",
			ClusterId:             "test-cluster-id",
			CloudProvider:         mocks.MockCloudProviderID,
			Region:                mocks.MockCloudRegionID,
			MultiAZ:               true,
			Schedulable:           true,
			KafkaInstanceLimit:    2,
			Status:                api.ClusterWaitingForKasFleetShardOperator,
			ProviderType:          api.ClusterProviderStandalone, // ensures there will be no errors with this test cluster not being available in ocm
			SupportedInstanceType: api.AllInstanceTypeSupport.String(),
		},
	}
	h, publicClient, teardown := kafkatest.NewKafkaHelperWithHooks(t, ocmServer, func(d *config.DataplaneClusterConfig) {
		d.DataPlaneClusterScalingType = config.ManualScaling
		d.ClusterConfig = config.NewClusterConfig(clusterList)
	})
	defer teardown()

	// run test only on mock mode - fleetshard sync will always be mocked so there's no point running against real env.
	ocmConfig := test.TestServices.OCMConfig
	if ocmConfig.MockMode != ocm.MockModeEmulateServer || h.Env.Name == environments.TestingEnv {
		t.SkipNow()
	}

	// run mock fleetshard sync
	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasFleetshardSync.Start()
	defer mockKasFleetshardSync.Stop()

	// wait for data plane cluster to become `ready`
	cluster, checkReadyErr := common.WaitForClusterStatus(test.TestServices.DBFactory, &test.TestServices.ClusterService, clusterList[0].ClusterId, api.ClusterReady)
	g.Expect(checkReadyErr).NotTo(gomega.HaveOccurred(), "error waiting for data plane cluster to be ready: %s %v", cluster.ClusterID, checkReadyErr)

	// setup private admin client
	adminClientCtx := NewAuthenticatedContextForAdminEndpoints(h, []string{testFullRole})
	adminClient := test.NewAdminPrivateAPIClient(h)

	// setup Kafka instance
	publicClientCtx := h.NewAuthenticatedContext(h.NewRandAccount(), nil)
	kafkaRequestPayload := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
		Plan:          fmt.Sprintf("%s.x1", types.STANDARD.String()),
	}
	publicKafkaReq, resp, err := publicClient.DefaultApi.CreateKafka(publicClientCtx, true, kafkaRequestPayload)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to create a Kafka instance")

	_, err = common.WaitForKafkaToReachStatus(publicClientCtx, kafkatest.TestServices.DBFactory, publicClient, publicKafkaReq.Id, kafkaconstants.KafkaRequestStatusPreparing)
	g.Expect(err).ToNot(gomega.HaveOccurred(), "error waiting for kafka to reach status %q", kafkaconstants.KafkaRequestStatusPreparing)

	// suspend Kafka instance
	// set 'suspending: true' on a non-ready Kafka instance: should not update the status
	suspendKafkaRequestPayload := private.KafkaUpdateRequest{
		Suspended: &[]bool{true}[0],
	}
	privateKafkaReq, resp, err := adminClient.DefaultApi.UpdateKafkaById(adminClientCtx, publicKafkaReq.Id, suspendKafkaRequestPayload)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(errors.Validation("").HttpCode))
	g.Expect(privateKafkaReq.Status).ToNot(gomega.Equal(kafkaconstants.KafkaRequestStatusSuspending.String()))

	// set 'suspending: true' on a ready Kafka instance: should set status to suspending
	_, err = common.WaitForKafkaToReachStatus(publicClientCtx, kafkatest.TestServices.DBFactory, publicClient, publicKafkaReq.Id, kafkaconstants.KafkaRequestStatusReady)
	g.Expect(err).ToNot(gomega.HaveOccurred(), "kafka failed to reach %q status", kafkaconstants.KafkaRequestStatusReady)

	privateKafkaReq, resp, err = adminClient.DefaultApi.UpdateKafkaById(adminClientCtx, publicKafkaReq.Id, suspendKafkaRequestPayload)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to suspend Kafka instance")
	g.Expect(privateKafkaReq.Status).To(gomega.Equal(kafkaconstants.KafkaRequestStatusSuspending.String()))

	// 'suspending' Kafkas should be updated to 'suspended' by the mock fleetshard sync
	_, err = common.WaitForKafkaToReachStatus(publicClientCtx, kafkatest.TestServices.DBFactory, publicClient, publicKafkaReq.Id, kafkaconstants.KafkaRequestStatusSuspended)
	g.Expect(err).ToNot(gomega.HaveOccurred(), "kafka failed to reach %q status", kafkaconstants.KafkaRequestStatusSuspended)

	privateKafkaReq, resp, err = adminClient.DefaultApi.GetKafkaById(adminClientCtx, privateKafkaReq.Id)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to get Kafka instance from admin get endpoint")
	g.Expect(privateKafkaReq.Status).To(gomega.Equal(kafkaconstants.KafkaRequestStatusSuspended.String()))

	// updating an already suspended Kafka instance with 'suspended: true' should not change its status
	privateKafkaReq, resp, err = adminClient.DefaultApi.UpdateKafkaById(adminClientCtx, publicKafkaReq.Id, suspendKafkaRequestPayload)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(privateKafkaReq.Status).To(gomega.Equal(kafkaconstants.KafkaRequestStatusSuspended.String()))

	// resume Kafka instance
	// set 'suspending: false' on a 'suspended' Kafka instance: should set status to resuming
	resumeKafkaRequestPayload := private.KafkaUpdateRequest{
		Suspended: &[]bool{false}[0],
	}
	privateKafkaReq, resp, err = adminClient.DefaultApi.UpdateKafkaById(adminClientCtx, publicKafkaReq.Id, resumeKafkaRequestPayload)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to resume Kafka instance")
	g.Expect(privateKafkaReq.Status).To(gomega.Equal(kafkaconstants.KafkaRequestStatusResuming.String()))

	// 'resuming' Kafkas should be updated to 'ready' by the mock fleetshard sync
	_, err = common.WaitForKafkaToReachStatus(publicClientCtx, kafkatest.TestServices.DBFactory, publicClient, publicKafkaReq.Id, kafkaconstants.KafkaRequestStatusReady)
	g.Expect(err).ToNot(gomega.HaveOccurred(), "kafka failed to reach %q status from %q", kafkaconstants.KafkaRequestStatusReady, kafkaconstants.KafkaRequestStatusResuming)

	privateKafkaReq, resp, err = adminClient.DefaultApi.GetKafkaById(adminClientCtx, privateKafkaReq.Id)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to get Kafka instance from admin get endpoint")
	g.Expect(privateKafkaReq.Status).To(gomega.Equal(kafkaconstants.KafkaRequestStatusReady.String()))
}
