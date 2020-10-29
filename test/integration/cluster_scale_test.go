package integration

import (
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
	"testing"

	. "github.com/onsi/gomega"
	"gitlab.cee.redhat.com/service/managed-services-api/test"
	utils "gitlab.cee.redhat.com/service/managed-services-api/test/common"
	"gitlab.cee.redhat.com/service/managed-services-api/test/mocks"
)

func TestClusterScaleUp(t *testing.T) {
	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, _, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	clusterID, getClusterErr := utils.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details from persisted .json file: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	// create machine pool
	machinePool, err := h.Env().Services.Cluster.ScaleUpMachinePool(clusterID)
	Expect(err).To(BeNil())
	Expect(machinePool.ID()).To(Equal(services.DefaultMachinePoolID))
	Expect(machinePool.Replicas()).To(Equal(2))
}
