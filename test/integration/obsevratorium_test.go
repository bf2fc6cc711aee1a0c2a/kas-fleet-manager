package integration

import (
	. "github.com/onsi/gomega"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
	"gitlab.cee.redhat.com/service/managed-services-api/test"
	"gitlab.cee.redhat.com/service/managed-services-api/test/mocks"
	"testing"
)

const (
	mockResourceNamespace = "my-kafka-namespace"
	mockKafkaClusterName  = "my-cluster"
)

func TestObservatorium_ResourceStateMetric(t *testing.T) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// start servers
	h, _, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()
	err := h.Env().LoadClients()
	Expect(err).NotTo(HaveOccurred(), "Error occur in loading client:  %v", err)

	defer h.Reset()
	service := services.NewObservatoriumService(h.Env().Clients.Observatorium)
	_, err = service.GetKafkaState(mockKafkaClusterName, mockResourceNamespace)
	Expect(err).NotTo(HaveOccurred(), "Error getting kafka state:  %v", err)

}
