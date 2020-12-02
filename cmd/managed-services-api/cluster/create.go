package cluster

import (
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/environments"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/flags"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	customOcm "gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"
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

	cmd.Flags().String(FlagRegion, "us-east-1", "Cluster region ID")
	cmd.Flags().String(FlagProvider, "aws", "Cluster provider")
	cmd.Flags().Bool(FlagMultiAZ, true, "Whether Cluster request should be Multi AZ or not")

	return cmd
}

func runCreate(cmd *cobra.Command, _ []string) {
	region := flags.MustGetDefinedString(FlagRegion, cmd.Flags())
	provider := flags.MustGetDefinedString(FlagProvider, cmd.Flags())
	multiAZ := flags.MustGetBool(FlagMultiAZ, cmd.Flags())
	if err := environments.Environment().Initialize(); err != nil {
		glog.Fatalf("Unable to initialize environment: %s", err.Error())
	}

	env := environments.Environment()

	ocmClient := customOcm.NewClient(env.Clients.OCM.Connection)

	clusterService := services.NewClusterService(env.DBFactory, ocmClient, env.Config.AWS)

	clusterRequest := api.Cluster{
		CloudProvider: provider,
		Region:        region,
		MultiAZ:       multiAZ,
	}

	if err := clusterService.RegisterClusterJob(&clusterRequest); err != nil {
		glog.Fatalf("Unable to create cluster request: %s", err.Error())
	}
}
