package main

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/server"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/onsi/gomega"
)

func TestInjections(t *testing.T) {
	g := gomega.NewWithT(t)

	env, err := environments.New(environments.DevelopmentEnv,
		kafka.ConfigProviders(),
	)
	g.Expect(err).To(gomega.BeNil())
	err = env.CreateServices()
	g.Expect(err).To(gomega.BeNil())

	var bootList []environments.BootService
	env.MustResolve(&bootList)
	g.Expect(len(bootList)).To(gomega.Equal(5))

	_, ok := bootList[0].(signalbus.SignalBus)
	g.Expect(ok).To(gomega.Equal(true))
	_, ok = bootList[1].(*server.ApiServer)
	g.Expect(ok).To(gomega.Equal(true))
	_, ok = bootList[2].(*server.MetricsServer)
	g.Expect(ok).To(gomega.Equal(true))
	_, ok = bootList[3].(*server.HealthCheckServer)
	g.Expect(ok).To(gomega.Equal(true))
	_, ok = bootList[4].(*workers.LeaderElectionManager)
	g.Expect(ok).To(gomega.Equal(true))

	var workerList []workers.Worker
	env.MustResolve(&workerList)
	g.Expect(workerList).To(gomega.HaveLen(8))

}
