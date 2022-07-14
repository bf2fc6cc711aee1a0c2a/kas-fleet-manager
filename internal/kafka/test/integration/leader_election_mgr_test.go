package integration

import (
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	pollTimeout  = 5 * time.Minute
	pollInterval = 1 * time.Second
)

func TestLeaderElection_StartedAllWorkersAndDropThenUp(t *testing.T) {
	g := gomega.NewWithT(t)
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	_, _, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	// wait and validate all workers are started.
	var clusterState bool
	err := wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		clusterState = test.TestServices.ClusterManager.IsRunning()
		return clusterState, nil
	})
	g.Expect(err).NotTo(gomega.HaveOccurred(), "", clusterState, err)
	g.Expect(clusterState).To(gomega.Equal(true))

	kafkaState := false
	err = wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		if len(test.TestServices.Workers) < 1 {
			return false, nil
		}
		for _, worker := range test.TestServices.Workers {
			if !worker.IsRunning() {
				return false, nil
			}
		}
		kafkaState = true
		return kafkaState, nil
	})

	g.Expect(err).NotTo(gomega.HaveOccurred(), "", kafkaState, err)
	g.Expect(kafkaState).To(gomega.Equal(true))

	// Take down a worker and valid it is really down
	test.TestServices.LeaderElectionManager.Stop()
	err = wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		clusterState = test.TestServices.ClusterManager.IsRunning()
		return !clusterState, nil
	})
	g.Expect(err).NotTo(gomega.HaveOccurred(), "", clusterState, err)
	g.Expect(clusterState).To(gomega.Equal(false))

	test.TestServices.LeaderElectionManager.Start()
	// Wait for Leader Election Manager to start it up again
	err = wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		clusterState = test.TestServices.ClusterManager.IsRunning()
		return clusterState, nil
	})
	g.Expect(err).NotTo(gomega.HaveOccurred(), "", clusterState, err)
	g.Expect(clusterState).To(gomega.Equal(true))
}
