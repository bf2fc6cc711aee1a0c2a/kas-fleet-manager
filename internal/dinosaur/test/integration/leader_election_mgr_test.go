package integration

import (
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/test"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/test/mocks"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	pollTimeout  = 5 * time.Minute
	pollInterval = 1 * time.Second
)

func TestLeaderElection_StartedAllWorkersAndDropThenUp(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	_, _, teardown := test.NewDinosaurHelper(t, ocmServer)
	defer teardown()

	// wait and validate all workers are started.
	var clusterState bool
	err := wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		clusterState = test.TestServices.ClusterManager.IsRunning()
		return clusterState, nil
	})
	Expect(err).NotTo(HaveOccurred(), "", clusterState, err)
	Expect(clusterState).To(Equal(true))

	dinosaurState := false
	err = wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		if len(test.TestServices.Workers) < 1 {
			return false, nil
		}
		for _, worker := range test.TestServices.Workers {
			if !worker.IsRunning() {
				return false, nil
			}
		}
		dinosaurState = true
		return dinosaurState, nil
	})

	Expect(err).NotTo(HaveOccurred(), "", dinosaurState, err)
	Expect(dinosaurState).To(Equal(true))

	// Take down a worker and valid it is really down
	test.TestServices.LeaderElectionManager.Stop()
	err = wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		clusterState = test.TestServices.ClusterManager.IsRunning()
		return !clusterState, nil
	})
	Expect(err).NotTo(HaveOccurred(), "", clusterState, err)
	Expect(clusterState).To(Equal(false))

	test.TestServices.LeaderElectionManager.Start()
	// Wait for Leader Election Manager to start it up again
	err = wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		clusterState = test.TestServices.ClusterManager.IsRunning()
		return clusterState, nil
	})
	Expect(err).NotTo(HaveOccurred(), "", clusterState, err)
	Expect(clusterState).To(Equal(true))
}