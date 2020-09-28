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
	go func() {
		clusterService := environments.Environment().Services.Cluster
		ocmClient := ocm.NewClient(environments.Environment().Clients.OCM.Connection)
		cloudProviderService := environments.Environment().Services.CloudProviders
		clusterManager := workers.NewClusterManager(clusterService,cloudProviderService,ocmClient)
		clusterManager.Start()
	}()

	select {}
}
