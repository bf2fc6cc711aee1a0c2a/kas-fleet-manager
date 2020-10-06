package integration

import (
	"testing"

	"github.com/golang/glog"
	. "github.com/onsi/gomega"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	ocm "gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
	"gitlab.cee.redhat.com/service/managed-services-api/test"
	"gitlab.cee.redhat.com/service/managed-services-api/test/mocks"
)

func TestSuccessfulClusterCreate(t *testing.T) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _ := test.RegisterIntegration(t, ocmServer)
	defer h.StopServer()

	clusterService := services.NewClusterService(h.Env().DBFactory, ocm.NewClient(h.Env().Clients.OCM.Connection), h.Env().Config.AWS)
	cluster, err := clusterService.Create(&api.Cluster{
		CloudProvider: "aws",
		Region:        "eu-west-1",
	})
	Expect(err).NotTo(HaveOccurred(), "Error occured when creating OSD cluster:  %v", err)
	Expect(cluster.ID()).NotTo(BeEmpty(), "Expected ID assigned on cluster creation")
	if err != nil {
		t.Fatalf("Unable to create cluster: %s", err.Error())
	}

	glog.V(10).Infof("Cluster %s created", cluster.ID())
}

func TestClusterCreateInvalidAwsCredentials(t *testing.T) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServerBuilder.SetClustersPostResponse(nil, errors.Validation("The provided AWS credentials are not valid"))
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _ := test.RegisterIntegration(t, ocmServer)
	defer h.StopServer()

	clusterService := services.NewClusterService(h.Env().DBFactory, ocm.NewClient(h.Env().Clients.OCM.Connection), h.Env().Config.AWS)

	// setting AWS.AccountID to invalid value
	h.Env().Config.AWS.AccountID = "123456789012"

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

	h, _ := test.RegisterIntegration(t, ocmServer)
	defer h.StopServer()

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
