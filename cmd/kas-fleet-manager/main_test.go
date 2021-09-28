package main

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/server"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	. "github.com/onsi/gomega"
)

func TestInjections(t *testing.T) {
	RegisterTestingT(t)

	env, err := environments.New(environments.DevelopmentEnv,
		kafka.ConfigProviders(),
	)
	Expect(err).To(BeNil())
	err = env.CreateServices()
	Expect(err).To(BeNil())

	var bootList []environments.BootService
	env.MustResolve(&bootList)
	Expect(len(bootList)).To(Equal(5))

	_, ok := bootList[0].(signalbus.SignalBus)
	Expect(ok).To(Equal(true))
	_, ok = bootList[1].(*server.ApiServer)
	Expect(ok).To(Equal(true))
	_, ok = bootList[2].(*server.MetricsServer)
	Expect(ok).To(Equal(true))
	_, ok = bootList[3].(*server.HealthCheckServer)
	Expect(ok).To(Equal(true))
	_, ok = bootList[4].(*workers.LeaderElectionManager)
	Expect(ok).To(Equal(true))

	var workerList []workers.Worker
	env.MustResolve(&workerList)
	Expect(workerList).To(HaveLen(9))

}
