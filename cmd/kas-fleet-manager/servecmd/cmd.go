package servecmd

import (
	goerrors "errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/server"
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
	err = env.ServiceContainer.Resolve(&diWorkers)
	if !goerrors.Is(err, di.ErrTypeNotExists) {
		panic(err)
	}
	for _, worker := range diWorkers {
		workerList = append(workerList, worker)
	}

	// starts Leader Election manager to coordinate workers job in a single or a replicas setting
	leaderElectionManager := workers.NewLeaderElectionManager(workerList, env.DBFactory)
	leaderElectionManager.Start()

	select {}
}
