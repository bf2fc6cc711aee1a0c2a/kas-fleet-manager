package cluster

import (
	"bytes"
	"github.com/golang/glog"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/environments"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/flags"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
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

	cmd.Flags().String(FlagRegion, "eu-west-1", "Cluster region ID")
	cmd.Flags().String(FlagProvider, "aws", "Cluster provider")

	return cmd
}

func runCreate(cmd *cobra.Command, _ []string) {
	region := flags.MustGetDefinedString(FlagRegion, cmd.Flags())
	provider := flags.MustGetDefinedString(FlagProvider, cmd.Flags())

	if err := environments.Environment().Initialize(); err != nil {
		glog.Fatalf("Unable to initialize environment: %s", err.Error())
	}

	env := environments.Environment()

	clusterService := services.NewClusterService(env.DBFactory, env.Clients.OCM.Connection, env.Config.AWS)

	cluster, err := clusterService.Create(&api.Cluster{
		CloudProvider: provider,
		Region:        region,
	})
	if err != nil {
		glog.Fatalf("Unable to create cluster: %s", err.Error())
	}

	indentedCluster := new(bytes.Buffer)
	if err := clustersmgmtv1.MarshalCluster(cluster, indentedCluster); err != nil {
		glog.Fatalf("Unable to marshal cluster: %s", err.Error())
	}
	glog.Infof("%s", indentedCluster.String())
}
