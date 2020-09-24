package integration

import (
	"github.com/golang/glog"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
	"gitlab.cee.redhat.com/service/managed-services-api/test"
	"gitlab.cee.redhat.com/service/managed-services-api/test/mocks"
	"testing"
)

func TestClusterCreate(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _ := test.RegisterIntegration(t, ocmServer)
	defer h.StopServer()

	clusterService := services.NewClusterService(h.Env().DBFactory, h.Env().Clients.OCM.Connection, h.Env().Config.AWS)

	cluster, err := clusterService.Create(&api.Cluster{
		CloudProvider: "aws",
		Region:        "eu-west-1",
	})
	if err != nil {
		t.Fatalf("Unable to create cluster: %s", err.Error())
	}
	glog.Infof("Cluster %s created", cluster.ID())
}
