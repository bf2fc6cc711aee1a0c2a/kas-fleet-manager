package integration

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	pollTimeout  = 5 * time.Minute
	pollInterval = 10 * time.Second
)

func TestLeaderElection_StartedAllWorkersAndDropThenUp(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	var clusterManager *workers.ClusterManager
	h.Env.MustResolveAll(&clusterManager)

	// wait and validate all workers are started.
	var clusterState bool
	err := wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		clusterState = clusterManager.IsRunning()
		return clusterState, nil
	})
	Expect(err).NotTo(HaveOccurred(), "", clusterState, err)
	Expect(clusterState).To(Equal(true))

	kafkaState := false
	err = wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		if len(h.Workers) < 1 {
			return false, nil
		}
		for _, worker := range h.Workers {
			if !worker.IsRunning() {
				return false, nil
			}
		}
		kafkaState = true
		return kafkaState, nil
	})

	Expect(err).NotTo(HaveOccurred(), "", kafkaState, err)
	Expect(kafkaState).To(Equal(true))

	// Take down a worker and valid it is really down
	h.LeaderElectionManager.Stop()
	err = wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		clusterState = clusterManager.IsRunning()
		return !clusterState, nil
	})
	Expect(err).NotTo(HaveOccurred(), "", clusterState, err)
	Expect(clusterState).To(Equal(false))

	h.LeaderElectionManager.Start()
	// Wait for Leader Election Manager to start it up again
	err = wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		clusterState = clusterManager.IsRunning()
		return clusterState, nil
	})
	Expect(err).NotTo(HaveOccurred(), "", clusterState, err)
	Expect(clusterState).To(Equal(true))
}
