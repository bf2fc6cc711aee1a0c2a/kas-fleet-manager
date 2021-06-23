package integration

import (
	"testing"
	"time"

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

	_, _, teardown := NewKafkaHelper(t, ocmServer)
	defer teardown()

	// wait and validate all workers are started.
	var clusterState bool
	err := wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		clusterState = testServices.ClusterManager.IsRunning()
		return clusterState, nil
	})
	Expect(err).NotTo(HaveOccurred(), "", clusterState, err)
	Expect(clusterState).To(Equal(true))

	kafkaState := false
	err = wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		if len(testServices.Workers) < 1 {
			return false, nil
		}
		for _, worker := range testServices.Workers {
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
	testServices.LeaderElectionManager.Stop()
	err = wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		clusterState = testServices.ClusterManager.IsRunning()
		return !clusterState, nil
	})
	Expect(err).NotTo(HaveOccurred(), "", clusterState, err)
	Expect(clusterState).To(Equal(false))

	testServices.LeaderElectionManager.Start()
	// Wait for Leader Election Manager to start it up again
	err = wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		clusterState = testServices.ClusterManager.IsRunning()
		return clusterState, nil
	})
	Expect(err).NotTo(HaveOccurred(), "", clusterState, err)
	Expect(clusterState).To(Equal(true))
}
