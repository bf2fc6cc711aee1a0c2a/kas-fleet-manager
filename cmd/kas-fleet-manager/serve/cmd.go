package serve

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/server"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/signalbus"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

func NewServeCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve the kas-fleet-manager",
		Long:  "Serve the Kafka Service Fleet Manager.",
		Run: func(cmd *cobra.Command, args []string) {
			runServe(env, cmd, args)
		},
	}
	return cmd
}

func runServe(env *environments.Env, cmd *cobra.Command, args []string) {
	if err := env.ServiceContainer.Invoke(func(
		apiServer *server.ApiServer,
		metricsServer *server.MetricsServer,
		healthCheckServer *server.HealthCheckServer,
		signalBus *signalbus.PgSignalBus,
		leaderElectionManager *workers.LeaderElectionManager,
	) {
		// Run the servers
		go apiServer.Start()
		go metricsServer.Start()
		go healthCheckServer.Start()

		signalBus.Start()
		// starts Leader Election manager to coordinate workers job in a single or a replicas setting
		leaderElectionManager.Start()

	}); err != nil {
		glog.Fatalf("di failure: %s", err.Error())
	}
	select {}
}
