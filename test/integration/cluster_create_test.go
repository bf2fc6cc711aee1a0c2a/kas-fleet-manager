package integration

import (
	. "github.com/onsi/gomega"
	api "gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	ocm "gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
	"gitlab.cee.redhat.com/service/managed-services-api/test"
	"gitlab.cee.redhat.com/service/managed-services-api/test/mocks"

	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

func TestSuccessfulClusterCreate(t *testing.T) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	clusterService := services.NewClusterService(h.Env().DBFactory, ocm.NewClient(h.Env().Clients.OCM.Connection), h.Env().Config.AWS)
	newCluster, err := clusterService.Create(&api.Cluster{
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Region:        mocks.MockCluster.Region().ID(),
	})
	Expect(err).NotTo(HaveOccurred(), "Error occurred when creating OSD cluster:  %v", err)
	Expect(newCluster.ID()).NotTo(BeEmpty(), "Expected ID assigned on cluster creation")

	var status string

	if err := wait.PollImmediate(30*time.Second, 120*time.Minute, func() (bool, error) {
		foundCluster, err := clusterService.FindClusterByID(newCluster.ID())
		if err != nil {
			return true, err
		}
		status = foundCluster.Status.String()
		return status == api.ClusterReady.String(), nil
	}); err != nil {
		t.Fatalf("Timed our waiting for cluster to become ready")
	}
	Expect(status).To(Equal(api.ClusterReady.String()))
}

func TestClusterCreateInvalidAwsCredentials(t *testing.T) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServerBuilder.SetClustersPostResponse(nil, errors.Validation("The provided AWS credentials are not valid"))
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	// setting AWS.AccountID to invalid value
	currentAWSAccountID := h.Env().Config.AWS.AccountID
	defer func(helper *test.Helper) {
		helper.Env().Config.AWS.AccountID = currentAWSAccountID
	}(h)
	h.Env().Config.AWS.AccountID = "123456789012"

	clusterService := services.NewClusterService(h.Env().DBFactory, ocm.NewClient(h.Env().Clients.OCM.Connection), h.Env().Config.AWS)

	cluster, err := clusterService.Create(&api.Cluster{
		CloudProvider: "aws",
		Region:        "eu-west-1",
	})
	Expect(err).To(HaveOccurred())
	Expect(cluster.ID()).To(Equal(""))
}

func TestClusterCreateInvalidToken(t *testing.T) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServerBuilder.SetClustersPostResponse(nil, errors.GeneralError("can't get access token: invalid_grant: Invalid refresh token"))
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	clusterService := services.NewClusterService(h.Env().DBFactory, ocm.NewClient(h.Env().Clients.OCM.Connection), h.Env().Config.AWS)
	// temporarily setting token to invalid value
	h.Env().Config.OCM.SelfTokenFile = "secrets/ocm-service.clientId"
	cluster, err := clusterService.Create(&api.Cluster{
		CloudProvider: "aws",
		Region:        "eu-west-1",
	})
	Expect(err).To(HaveOccurred())
	Expect(cluster.ID()).To(Equal(""))
}
