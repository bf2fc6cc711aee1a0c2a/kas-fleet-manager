package phase

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/onsi/gomega"
)

func Test_PerformNamespaceOperation(t *testing.T) {

	tests := []struct {
		scenario       string
		ClusterPhase   dbapi.ConnectorClusterPhaseEnum
		operation      NamespaceOperation
		NamespacePhase dbapi.ConnectorNamespacePhaseEnum
		expectError    bool
		updated        bool
		result         dbapi.ConnectorNamespacePhaseEnum
	}{
		{
			scenario:       "connect disconnected Namespace in connected cluster",
			ClusterPhase:   dbapi.ConnectorClusterPhaseReady,
			NamespacePhase: dbapi.ConnectorNamespacePhaseDisconnected,
			operation:      ConnectNamespace,
			expectError:    false,
			updated:        true,
			result:         dbapi.ConnectorNamespacePhaseReady,
		},
		{
			scenario:       "connect disconnected Namespace in disconnected cluster",
			ClusterPhase:   dbapi.ConnectorClusterPhaseDisconnected,
			NamespacePhase: dbapi.ConnectorNamespacePhaseDisconnected,
			operation:      ConnectNamespace,
			expectError:    true,
			updated:        false,
			result:         dbapi.ConnectorNamespacePhaseDisconnected,
		},
		{
			scenario:       "disconnect disconnected Namespace",
			ClusterPhase:   dbapi.ConnectorClusterPhaseDisconnected,
			NamespacePhase: dbapi.ConnectorNamespacePhaseDisconnected,
			operation:      DisconnectNamespace,
			expectError:    false,
			updated:        false,
			result:         dbapi.ConnectorNamespacePhaseDisconnected,
		},
		// TODO add rest of the test scenarios
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.scenario, func(t *testing.T) {
			g := gomega.NewWithT(t)

			cluster := &dbapi.ConnectorCluster{
				Status: dbapi.ConnectorClusterStatus{
					Phase: tt.ClusterPhase,
				},
			}
			namespace := &dbapi.ConnectorNamespace{
				Status: dbapi.ConnectorNamespaceStatus{
					Phase: tt.NamespacePhase,
				},
			}
			updated, err := PerformNamespaceOperation(cluster, namespace, tt.operation)

			g.Expect(updated).Should(gomega.Equal(tt.updated), "PerformNamespaceOperation phase updated=%v, expect updated=%v", updated, tt.updated)
			g.Expect(err != nil).Should(gomega.Equal(tt.expectError), "PerformNamespaceOperation error=%v, expectError=%v", err, tt.expectError)

			phase := namespace.Status.Phase
			g.Expect(phase).Should(gomega.Equal(tt.result), "PerformNamespaceOperation phase=%v, expect phase=%v", phase, tt.result)
		})
	}
}
