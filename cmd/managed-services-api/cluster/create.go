package cluster

import (
	"bytes"

	"github.com/golang/glog"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
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

	cluster, err := clusterService.Create(&api.Cluster{
		CloudProvider: provider,
		Region:        region,
		MultiAZ:       multiAZ,
	})
	if err != nil {
		glog.Fatalf("Unable to create cluster: %s", err.Error())
	}

	// as all fields on OCM structs are internal we cannot perform a standard json marshal as all fields will be empty,
	// instead we need to use the OCM type-specific marshal functions when converting a struct to json
	// declare a buffer to store the resulting json and invoke the OCM type-specific marshal function to populate the
	// buffer with a json string containing the internal cluster values.
	indentedCluster := new(bytes.Buffer)
	if err := clustersmgmtv1.MarshalCluster(cluster, indentedCluster); err != nil {
		glog.Fatalf("Unable to marshal cluster: %s", err.Error())
	}
	glog.V(10).Infof("%s", indentedCluster.String())
}
