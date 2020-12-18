package servecmd

import (
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"

	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/environments"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/server"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/workers"
)

func NewServeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve the managed-services-api",
		Long:  "Serve the managed-services-api.",
		Run:   runServe,
	}
	err := environments.Environment().AddFlags(cmd.PersistentFlags())
	if err != nil {
		glog.Fatalf("Unable to add environment flags to serve command: %s", err.Error())
	}

	return cmd
}

func runServe(cmd *cobra.Command, args []string) {
	err := environments.Environment().Initialize()
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

	// Run the cluster manager
	ocmClient := ocm.NewClient(environments.Environment().Clients.OCM.Connection)

	// creates cluster worker
	cloudProviderService := environments.Environment().Services.CloudProviders
	clusterService := environments.Environment().Services.Cluster
	configService := environments.Environment().Services.Config
	serverConfig := *environments.Environment().Config.Server

	var workerList []workers.Worker

	//set Unique Id for each work to facilitate Leader Election process
	clusterManager := workers.NewClusterManager(clusterService, cloudProviderService, ocmClient, configService, serverConfig, uuid.New().String())
	workerList = append(workerList, clusterManager)

	ocmClient = ocm.NewClient(environments.Environment().Clients.OCM.Connection)

	// creates kafka worker
	clusterService = environments.Environment().Services.Cluster
	kafkaService := environments.Environment().Services.Kafka
	keycloakService := environments.Environment().Services.Keycloak
	observatoriumService := environments.Environment().Services.Observatorium

	//set Unique Id for each work to facilitate Leader Election process
	kafkaManager := workers.NewKafkaManager(kafkaService, clusterService, ocmClient, uuid.New().String(), keycloakService, observatoriumService)
	workerList = append(workerList, kafkaManager)

	// starts Leader Election manager to coordinate workers job in a single or a replicas setting
	leaderElectionManager := workers.NewLeaderLeaseManager(workerList, environments.Environment().DBFactory)
	leaderElectionManager.Start()

	select {}
}
