package servecmd

import (
	"github.com/golang/glog"
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

	// start cluster worker
	cloudProviderService := environments.Environment().Services.CloudProviders
	clusterService := environments.Environment().Services.Cluster
	configService := environments.Environment().Services.Config
	serverConfig := *environments.Environment().Config.Server
	clusterManager := workers.NewClusterManager(clusterService, cloudProviderService, ocmClient, configService, serverConfig)
	clusterManager.Start()

	ocmClient = ocm.NewClient(environments.Environment().Clients.OCM.Connection)

	// start kafka worker
	clusterService = environments.Environment().Services.Cluster
	kafkaService := environments.Environment().Services.Kafka
	kafkaManager := workers.NewKafkaManager(kafkaService, clusterService, ocmClient)
	kafkaManager.Start()

	select {}
}
