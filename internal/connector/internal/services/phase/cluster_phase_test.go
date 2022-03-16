package phase

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	. "github.com/onsi/gomega"
	"testing"
)

func Test_PerformClusterOperation(t *testing.T) {

	tests := []struct {
		scenario    string
		operation   ClusterOperation
		startPhase  dbapi.ConnectorClusterPhaseEnum
		expectError bool
		updated     bool
		result      dbapi.ConnectorClusterPhaseEnum
	}{
		{
			scenario:    "connect disconnected cluster",
			operation:   ConnectCluster,
			startPhase:  dbapi.ConnectorClusterPhaseDisconnected,
			expectError: false,
			updated:     true,
			result:      dbapi.ConnectorClusterPhaseReady,
		},
		{
			scenario:    "connect deleting cluster",
			operation:   ConnectCluster,
			startPhase:  dbapi.ConnectorClusterPhaseDeleting,
			expectError: false,
			updated:     false,
			result:      dbapi.ConnectorClusterPhaseDeleting,
		},
		{
			scenario:    "disconnect disconnected cluster",
			operation:   DisconnectCluster,
			startPhase:  dbapi.ConnectorClusterPhaseDisconnected,
			expectError: false,
			updated:     false,
			result:      dbapi.ConnectorClusterPhaseDisconnected,
		},
		// TODO add rest of the test scenarios
	}

	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {
			RegisterTestingT(t)

			cluster := &dbapi.ConnectorCluster{
				Status: dbapi.ConnectorClusterStatus{
					Phase: tt.startPhase,
				},
			}
			phaseSaved := false
			updated, err := PerformClusterOperation(cluster, tt.operation, func(c *dbapi.ConnectorCluster) *errors.ServiceError {
				phaseSaved = true
				return nil
			})

			Expect(updated).Should(Equal(tt.updated), "PerformClusterOperation phase updated=%v, expect updated=%v", updated, tt.updated)
			Expect(err != nil).Should(Equal(tt.expectError), "PerformClusterOperation error=%v, expect error=%v", err, tt.expectError)
			Expect(phaseSaved).Should(Equal(tt.updated), "PerformClusterOperation phase updated=%v, expect phase updated=%v", phaseSaved, tt.updated)

			phase := cluster.Status.Phase
			Expect(phase).Should(Equal(tt.result), "PerformClusterOperation phase=%v, expect phase=%v", phase, tt.result)
		})
	}
}
