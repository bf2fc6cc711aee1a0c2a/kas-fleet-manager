package phase

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/onsi/gomega"
	"testing"
)

func Test_FSMPerform_ConnectorNamespaceReady(t *testing.T) {
	type test struct {
		scenario                   string
		operation                  ProcessorOperation
		connectorNamespacePhase    dbapi.ConnectorNamespacePhaseEnum
		processorStartDesiredState dbapi.ProcessorDesiredState
		processorEndDesiredState   dbapi.ProcessorDesiredState
		expectError                bool
		updated                    bool
	}

	tests := []test{
		{
			scenario:                   "Create::Ready->Ready",
			operation:                  CreateProcessor,
			connectorNamespacePhase:    dbapi.ConnectorNamespacePhaseReady,
			processorStartDesiredState: dbapi.ProcessorReady,
			processorEndDesiredState:   dbapi.ProcessorReady,
			expectError:                false,
			updated:                    false,
		},
		{
			scenario:                   "Create::Stopped->Stopped",
			operation:                  CreateProcessor,
			connectorNamespacePhase:    dbapi.ConnectorNamespacePhaseReady,
			processorStartDesiredState: dbapi.ProcessorStopped,
			processorEndDesiredState:   dbapi.ProcessorStopped,
			expectError:                true,
			updated:                    false,
		},
		{
			scenario:                   "Create::Deleted->Deleted",
			operation:                  CreateProcessor,
			connectorNamespacePhase:    dbapi.ConnectorNamespacePhaseReady,
			processorStartDesiredState: dbapi.ProcessorDeleted,
			processorEndDesiredState:   dbapi.ProcessorDeleted,
			expectError:                true,
			updated:                    false,
		},
		{
			scenario:                   "Update::Ready->Ready",
			operation:                  UpdateProcessor,
			connectorNamespacePhase:    dbapi.ConnectorNamespacePhaseReady,
			processorStartDesiredState: dbapi.ProcessorReady,
			processorEndDesiredState:   dbapi.ProcessorReady,
			expectError:                false,
			updated:                    false,
		},
		{
			// Updating a stopped processor does not start it
			scenario:                   "Update::Stopped->Stopped",
			operation:                  UpdateProcessor,
			connectorNamespacePhase:    dbapi.ConnectorNamespacePhaseReady,
			processorStartDesiredState: dbapi.ProcessorStopped,
			processorEndDesiredState:   dbapi.ProcessorStopped,
			expectError:                false,
			updated:                    false,
		},
		{
			// You cannot update a deleted processor
			scenario:                   "Update::Deleted->Deleted",
			operation:                  UpdateProcessor,
			connectorNamespacePhase:    dbapi.ConnectorNamespacePhaseReady,
			processorStartDesiredState: dbapi.ProcessorDeleted,
			processorEndDesiredState:   dbapi.ProcessorDeleted,
			expectError:                true,
			updated:                    false,
		},
		{
			scenario:                   "Delete::Ready->Deleted",
			operation:                  DeleteProcessor,
			connectorNamespacePhase:    dbapi.ConnectorNamespacePhaseReady,
			processorStartDesiredState: dbapi.ProcessorReady,
			processorEndDesiredState:   dbapi.ProcessorDeleted,
			expectError:                false,
			updated:                    true,
		},
		{
			scenario:                   "Delete::Deleted->Deleted",
			operation:                  DeleteProcessor,
			connectorNamespacePhase:    dbapi.ConnectorNamespacePhaseReady,
			processorStartDesiredState: dbapi.ProcessorDeleted,
			processorEndDesiredState:   dbapi.ProcessorDeleted,
			expectError:                false,
			updated:                    false,
		},
		{
			scenario:                   "Delete::Stopped->Deleted",
			operation:                  DeleteProcessor,
			connectorNamespacePhase:    dbapi.ConnectorNamespacePhaseReady,
			processorStartDesiredState: dbapi.ProcessorStopped,
			processorEndDesiredState:   dbapi.ProcessorDeleted,
			expectError:                false,
			updated:                    true,
		},
		{
			scenario:                   "Stop::Ready->Stopped",
			operation:                  StopProcessor,
			connectorNamespacePhase:    dbapi.ConnectorNamespacePhaseReady,
			processorStartDesiredState: dbapi.ProcessorReady,
			processorEndDesiredState:   dbapi.ProcessorStopped,
			expectError:                false,
			updated:                    true,
		},
		{
			scenario:                   "Stop::Stopped->Stopped",
			operation:                  StopProcessor,
			connectorNamespacePhase:    dbapi.ConnectorNamespacePhaseReady,
			processorStartDesiredState: dbapi.ProcessorStopped,
			processorEndDesiredState:   dbapi.ProcessorStopped,
			expectError:                false,
			updated:                    false,
		},
		{
			// You cannot stop a deleted processor
			scenario:                   "Stop::Deleted->Stopped",
			operation:                  StopProcessor,
			connectorNamespacePhase:    dbapi.ConnectorNamespacePhaseReady,
			processorStartDesiredState: dbapi.ProcessorDeleted,
			processorEndDesiredState:   dbapi.ProcessorDeleted,
			expectError:                true,
			updated:                    false,
		},
		{
			scenario:                   "Restart::Ready->Ready",
			operation:                  RestartProcessor,
			connectorNamespacePhase:    dbapi.ConnectorNamespacePhaseReady,
			processorStartDesiredState: dbapi.ProcessorReady,
			processorEndDesiredState:   dbapi.ProcessorReady,
			expectError:                false,
			updated:                    false,
		},
		{
			scenario:                   "Restart::Stopped->Ready",
			operation:                  RestartProcessor,
			connectorNamespacePhase:    dbapi.ConnectorNamespacePhaseReady,
			processorStartDesiredState: dbapi.ProcessorStopped,
			processorEndDesiredState:   dbapi.ProcessorReady,
			expectError:                false,
			updated:                    true,
		},
		{
			// You cannot restart a deleted processor
			scenario:                   "Restart::Deleted->Deleted",
			operation:                  RestartProcessor,
			connectorNamespacePhase:    dbapi.ConnectorNamespacePhaseReady,
			processorStartDesiredState: dbapi.ProcessorDeleted,
			processorEndDesiredState:   dbapi.ProcessorDeleted,
			expectError:                true,
			updated:                    false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.scenario, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			namespace := dbapi.ConnectorNamespace{
				Status: dbapi.ConnectorNamespaceStatus{
					Phase: tt.connectorNamespacePhase,
				},
			}
			processor := dbapi.Processor{
				DesiredState: tt.processorStartDesiredState,
				Status: dbapi.ProcessorStatus{
					Phase: dbapi.ProcessorStatusPhasePreparing,
				},
			}

			phaseSaved := false
			updated, err := PerformProcessorOperation(&namespace, &processor, tt.operation, func(c *dbapi.Processor) *errors.ServiceError {
				phaseSaved = true
				return nil
			})

			g.Expect(updated).Should(gomega.Equal(tt.updated), "PerformProcessorOperation phase updated=%v, expect updated=%v", updated, tt.updated)
			g.Expect(err != nil).Should(gomega.Equal(tt.expectError), "PerformProcessorOperation error=%v, expect error=%v", err, tt.expectError)
			g.Expect(phaseSaved).Should(gomega.Equal(tt.updated), "PerformProcessorOperation phase updated=%v, expect phase updated=%v", phaseSaved, tt.updated)

			desiredState := processor.DesiredState
			g.Expect(desiredState).Should(gomega.Equal(tt.processorEndDesiredState), "PerformProcessorOperation phase=%v, expect desiredState=%v", processor.Status.Phase, tt.processorEndDesiredState)
		})
	}
}
