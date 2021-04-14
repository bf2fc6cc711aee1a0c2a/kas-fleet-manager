package servecmd

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/server"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/signalbus"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
)

func NewServeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve the kas-fleet-manager",
		Long:  "Serve the Kafka Service Fleet Manager.",
		Run:   runServe,
	}
	err := environments.Environment().AddFlags(cmd.PersistentFlags())
	if err != nil {
		glog.Fatalf("Unable to add environment flags to serve command: %s", err.Error())
	}

	return cmd
}

func runServe(cmd *cobra.Command, args []string) {
	env := environments.Environment()
	err := env.Initialize()
	if err != nil {
		glog.Fatalf("Unable to initialize environment: %s", err.Error())
	}

	// Run the servers
	apiserver := server.NewAPIServer()
	go func() {
		apiserver.Start()
	}()

	metricsServer := server.NewMetricsServer()
	go func() {
		metricsServer.Start()
	}()

	healthcheckServer := server.NewHealthCheckServer()
	go func() {
		healthcheckServer.Start()
	}()

	// Replace the default signal bus with a clustered version.

	env.Services.SignalBus.(*signalbus.PgSignalBus).Start()

	// Run the cluster manager
	ocmClient := ocm.NewClient(env.Clients.OCM.Connection)

	// creates cluster worker
	cloudProviderService := env.Services.CloudProviders
	clusterService := env.Services.Cluster
	configService := env.Services.Config
	keycloakService := env.Services.Keycloak
	osdIdpKeycloakService := env.Services.OsdIdpKeycloak
	var workerList []workers.Worker
	kasFleetshardOperatorAddon := env.Services.KasFleetshardAddonService

	//set Unique Id for each work to facilitate Leader Election process
	clusterManager := workers.NewClusterManager(clusterService, cloudProviderService, ocmClient, configService, uuid.New().String(), kasFleetshardOperatorAddon, osdIdpKeycloakService)
	workerList = append(workerList, clusterManager)

	ocmClient = ocm.NewClient(env.Clients.OCM.Connection)

	// creates kafka worker
	kafkaService := environments.Environment().Services.Kafka
	observatoriumService := environments.Environment().Services.Observatorium
	quotaService := environments.Environment().Services.Quota
	clusterPlmtStrategy := env.Services.ClusterPlmtStrategy

	//set Unique Id for each work to facilitate Leader Election process
	kafkaManager := workers.NewKafkaManager(kafkaService, ocmClient, uuid.New().String(), keycloakService, observatoriumService, configService, quotaService, clusterPlmtStrategy)
	workerList = append(workerList, kafkaManager)

	// add the connector manager worker
	workerList = append(workerList, workers.NewConnectorManager(
		uuid.New().String(),
		env.Services.ConnectorTypes,
		env.Services.Connectors,
		env.Services.ConnectorCluster,
		env.Services.Observatorium,
		env.Services.Vault,
	))

	// starts Leader Election manager to coordinate workers job in a single or a replicas setting
	leaderElectionManager := workers.NewLeaderElectionManager(workerList, env.DBFactory)
	leaderElectionManager.Start()

	// Wait for shutdown signal...
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals
	glog.Infoln("shutting down")

	leaderElectionManager.Stop()
	_ = apiserver.Stop()
	_ = metricsServer.Stop()
	_ = healthcheckServer.Stop()

}
