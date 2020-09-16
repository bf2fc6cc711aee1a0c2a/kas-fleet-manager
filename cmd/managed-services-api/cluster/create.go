package cluster

import (
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/environments"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
)

// NewCreateCommand creates a new command for creating clusters.
func NewCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new managed-services-api cluster",
		Long:  "Create a new managed-services-api cluster.",
		Run:   runCreate,
	}
	err := environments.Environment().AddFlags(cmd.PersistentFlags())
	if err != nil {
		glog.Fatalf("Unable to add environment flags to serve command: %s", err.Error())
	}

	cmd.Flags().String("region", "eu-west-1", "Cluster region ID")
	cmd.Flags().String("provider", "aws", "Cluster provider")

	return cmd
}

func runCreate(cmd *cobra.Command, _ []string) {
	region, err := cmd.Flags().GetString("region")
	if err != nil {
		glog.Fatalf("Unable to parse region flag: %s", err.Error())
	}
	provider, err := cmd.Flags().GetString("provider")
	if err != nil {
		glog.Fatalf("Unable to parse provider flag: %s", err.Error())
	}

	if err = environments.Environment().Initialize(); err != nil {
		glog.Fatalf("Unable to initialize environment: %s", err.Error())
	}

	env := environments.Environment()

	clusterService := services.NewClusterService(env.DBFactory, env.Clients.OCM.Connection, env.Config.AWS)

	cluster, err := clusterService.Create(&api.Cluster{
		CloudProvider: provider,
		Region:        region,
	})
	// see https://yourbasic.org/golang/gotcha-why-nil-error-not-equal-nil/ for the reason for casting.
	if err != (*errors.ServiceError)(nil) {
		glog.Fatalf("Unable to create cluster: %s", err.Error())
	}
	glog.Infof("Cluster %s created", cluster.ID())
}
