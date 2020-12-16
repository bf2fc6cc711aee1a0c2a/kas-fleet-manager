package integration

import (
	. "github.com/onsi/gomega"
	"gitlab.cee.redhat.com/service/managed-services-api/test"
	"gitlab.cee.redhat.com/service/managed-services-api/test/mocks"
	"k8s.io/apimachinery/pkg/util/wait"
	"testing"
	"time"
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

	// wait and valida all workers are started.
	var clusterState bool
	err := wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		clusterState = h.ClusterWorker.IsRunning()
		return clusterState, nil
	})
	Expect(err).NotTo(HaveOccurred(), "", clusterState, err)
	Expect(clusterState).To(Equal(true))

	var kafkaState bool
	err = wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		kafkaState = h.KafkaWorker.IsRunning()
		return kafkaState, nil
	})
	Expect(err).NotTo(HaveOccurred(), "", kafkaState, err)
	Expect(kafkaState).To(Equal(true))

	// Take down a worker and valid it is really down
	h.ClusterWorker.SetIsRunning(false)
	err = wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		clusterState = h.ClusterWorker.IsRunning()
		return !clusterState, nil
	})
	Expect(err).NotTo(HaveOccurred(), "", clusterState, err)
	Expect(clusterState).To(Equal(false))

	// Wait for Leader Election Manager to start it up again
	err = wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		clusterState = h.ClusterWorker.IsRunning()
		return clusterState, nil
	})
	Expect(err).NotTo(HaveOccurred(), "", clusterState, err)
	Expect(clusterState).To(Equal(true))
}
