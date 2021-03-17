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
	go func() {
		apiserver := server.NewAPIServer()
		apiserver.Start()
	}()

	go func() {
		metricsServer := server.NewMetricsServer()
		metricsServer.Start()
	}()

	go func() {
		healthcheckServer := server.NewHealthCheckServer()
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
	clusterService = environments.Environment().Services.Cluster
	kafkaService := environments.Environment().Services.Kafka
	observatoriumService := environments.Environment().Services.Observatorium
	quotaService := environments.Environment().Services.Quota
	//set Unique Id for each work to facilitate Leader Election process
	kafkaManager := workers.NewKafkaManager(kafkaService, clusterService, ocmClient, uuid.New().String(), keycloakService, observatoriumService, configService, quotaService)
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

	select {}
}
