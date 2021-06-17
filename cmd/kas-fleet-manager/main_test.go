package main

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/provider"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/server"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/signalbus"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	. "github.com/onsi/gomega"
	"testing"
)

func TestInjections(t *testing.T) {
	RegisterTestingT(t)

	env, err := environments.NewEnv(environments.DevelopmentEnv,
		kafka.ConfigProviders().AsOption(),
	)
	Expect(err).To(BeNil())
	err = env.CreateServices()
	Expect(err).To(BeNil())

	var bootList []provider.BootService
	env.MustResolve(&bootList)
	Expect(len(bootList)).To(Equal(5))

	_, ok := bootList[0].(*server.ApiServer)
	Expect(ok).To(Equal(true))
	_, ok = bootList[1].(*server.MetricsServer)
	Expect(ok).To(Equal(true))
	_, ok = bootList[2].(*server.HealthCheckServer)
	Expect(ok).To(Equal(true))
	_, ok = bootList[3].(signalbus.SignalBus)
	Expect(ok).To(Equal(true))
	_, ok = bootList[4].(*workers.LeaderElectionManager)
	Expect(ok).To(Equal(true))

	var workerList []workers.Worker
	env.MustResolve(&workerList)
	Expect(len(workerList)).To(Equal(7))

}
