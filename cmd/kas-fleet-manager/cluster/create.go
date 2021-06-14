package cluster

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/flags"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

// NewCreateCommand creates a new command for creating clusters.
func NewCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new cluster",
		Long:  "Create a new cluster.",
		Run:   runCreate,
	}
	err := environments.Environment().AddFlags(cmd.PersistentFlags())
	if err != nil {
		glog.Fatalf("Unable to add environment flags to serve command: %s", err.Error())
	}

	cmd.Flags().String(FlagRegion, "us-east-1", "Cluster region ID")
	cmd.Flags().String(FlagProvider, "aws", "Cluster provider")
	cmd.Flags().Bool(FlagMultiAZ, true, "Whether Cluster request should be Multi AZ or not")
	cmd.Flags().String(FlagProviderType, "ocm", "The provider type")

	return cmd
}

func runCreate(cmd *cobra.Command, _ []string) {
	region := flags.MustGetDefinedString(FlagRegion, cmd.Flags())
	provider := flags.MustGetDefinedString(FlagProvider, cmd.Flags())
	multiAZ := flags.MustGetBool(FlagMultiAZ, cmd.Flags())
	providerType := flags.MustGetDefinedString(FlagProviderType, cmd.Flags())
	if err := environments.Environment().Initialize(); err != nil {
		glog.Fatalf("Unable to initialize environment: %s", err.Error())
	}

	env := environments.Environment()
	clusterService := env.Services.Cluster

	clusterRequest := api.Cluster{
		CloudProvider: provider,
		Region:        region,
		MultiAZ:       multiAZ,
		Status:        api.ClusterAccepted,
		ProviderType:  api.ClusterProviderType(providerType),
	}

	if err := clusterService.RegisterClusterJob(&clusterRequest); err != nil {
		glog.Fatalf("Unable to create cluster request: %s", err.Error())
	}

	glog.Infoln("Cluster request has been registered and will be reconciled shortly.")
}
