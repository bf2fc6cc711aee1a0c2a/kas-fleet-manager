package dinosaur

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/test"
	"os"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/test/cucumber"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/test/mocks"
)

func TestMain(m *testing.M) {

	// Startup all the services and mocks that are needed to test the
	// connector features.
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, teardown := test.NewDinosaurHelper(&testing.T{}, ocmServer)
	defer teardown()

	status := cucumber.TestMain(h)
	if st := m.Run(); st > status {
		status = st
	}
	os.Exit(status)
}
