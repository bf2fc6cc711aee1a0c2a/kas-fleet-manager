package servecmd

import (
	goerrors "errors"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/server"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/signalbus"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers/kafka_mgrs"
	"github.com/goava/di"
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

	if err := env.ServiceContainer.Invoke(func(apiserver *server.ApiServer, metricsServer *server.MetricsServer, healthcheckServer *server.HealthCheckServer) {
		// Run the servers
		go apiserver.Start()
		go metricsServer.Start()
		go healthcheckServer.Start()
	}); err != nil {
		glog.Fatalf("di failure: %s", err.Error())
	}

	// Replace the default signal bus with a clustered version.

	env.Services.SignalBus.(*signalbus.PgSignalBus).Start()

	// creates cluster worker
	cloudProviderService := env.Services.CloudProviders
	clusterService := env.Services.Cluster
	configService := env.Services.Config
	keycloakService := env.Services.Keycloak
	osdIdpKeycloakService := env.Services.OsdIdpKeycloak
	var workerList []workers.Worker
	kasFleetshardOperatorAddon := env.Services.KasFleetshardAddonService

	//set Unique Id for each work to facilitate Leader Election process
	clusterManager := workers.NewClusterManager(clusterService, cloudProviderService, configService, uuid.New().String(), kasFleetshardOperatorAddon, osdIdpKeycloakService, env.Services.SignalBus)
	workerList = append(workerList, clusterManager)

	// creates kafka worker
	kafkaService := environments.Environment().Services.Kafka
	observatoriumService := environments.Environment().Services.Observatorium
	clusterPlmtStrategy := env.Services.ClusterPlmtStrategy

	//create kafka manager per type and assign them a Unique Id for each work to facilitate Leader Election process
	kafkaManager := kafka_mgrs.NewKafkaManager(kafkaService, uuid.New().String(), configService, env.Services.SignalBus)
	acceptedKafkaManager := kafka_mgrs.NewAcceptedKafkaManager(kafkaService, uuid.New().String(), configService, env.QuotaServiceFactory, clusterPlmtStrategy, env.Services.SignalBus)
	preparingKafkaManager := kafka_mgrs.NewPreparingKafkaManager(kafkaService, uuid.New().String(), env.Services.SignalBus)
	deletingKafkaManager := kafka_mgrs.NewDeletingKafkaManager(kafkaService, uuid.New().String(), configService, env.QuotaServiceFactory, env.Services.SignalBus)
	provisioningKafkaManager := kafka_mgrs.NewProvisioningKafkaManager(kafkaService, uuid.New().String(), observatoriumService, configService, env.Services.SignalBus)
	readyKafkaManager := kafka_mgrs.NewReadyKafkaManager(kafkaService, uuid.New().String(), keycloakService, configService, env.Services.SignalBus)
	workerList = append(workerList, kafkaManager, acceptedKafkaManager, preparingKafkaManager, deletingKafkaManager, provisioningKafkaManager, readyKafkaManager)

	// Add the DI injected workers...
	var diWorkers []workers.Worker
	if err := env.ServiceContainer.Resolve(&diWorkers); err != nil && !goerrors.Is(err, di.ErrTypeNotExists) {
		panic(err)
	}
	workerList = append(workerList, diWorkers...)

	// starts Leader Election manager to coordinate workers job in a single or a replicas setting
	leaderElectionManager := workers.NewLeaderElectionManager(workerList, env.DBFactory)
	leaderElectionManager.Start()

	select {}
}
