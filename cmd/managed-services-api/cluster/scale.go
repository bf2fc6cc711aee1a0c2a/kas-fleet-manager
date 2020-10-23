package cluster

import (
	"bytes"
	"github.com/golang/glog"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/environments"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/flags"
	customOcm "gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
)

// NewScaleCommand creates a new command for scaling machine pools
func NewScaleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scale",
		Short: "Scale the managed services machine pool",
		Long:  "Scale nodes up or down in the cluster machine pool.",
	}
	err := environments.Environment().AddFlags(cmd.PersistentFlags())
	if err != nil {
		glog.Fatalf("Unable to add environment flags to serve command: %s", err.Error())
	}
	cmd.AddCommand(NewScaleUpCommand(), NewScaleDownCommand())
	return cmd
}

// NewScaleUpCommand creates a new command for scaling up the machine pool
func NewScaleUpCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Scale up a node",
		Long:  "Scale up a node in the managed services machine pool.",
		Run:   runScaleUp,
	}
	cmd.Flags().String(FlagClusterID, "", "Cluster ID")
	return cmd
}

// NewScaleDownCommand creates a new command for scaling down the machine pool
func NewScaleDownCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "down",
		Short: "Scale down a node",
		Long:  "Scale down a node in the managed services machine pool.",
		Run:   runScaleDown,
	}
	cmd.Flags().String(FlagClusterID, "", "Cluster ID")
	return cmd
}

func runScaleUp(cmd *cobra.Command, _ []string) {
	clusterID := flags.MustGetDefinedString(FlagClusterID, cmd.Flags())
	if err := environments.Environment().Initialize(); err != nil {
		glog.Fatalf("Unable to initialize environment: %s", err.Error())
	}
	env := environments.Environment()
	ocmClient := customOcm.NewClient(env.Clients.OCM.Connection)
	clusterService := services.NewClusterService(env.DBFactory, ocmClient, env.Config.AWS)

	// scale up the machine pool
	machinePool, err := clusterService.ScaleUpMachinePool(clusterID)
	if err != nil {
		glog.Fatalf("Unable to scale up machine pool: %s", err.Error())
	}

	// print the output
	indentedPool := new(bytes.Buffer)
	if err := clustersmgmtv1.MarshalMachinePool(machinePool, indentedPool); err != nil {
		glog.Fatalf("Unable to marshal machine pool: %s", err.Error())
	}
	glog.V(10).Infof("%s", indentedPool.String())
}

func runScaleDown(cmd *cobra.Command, _ []string) {
	clusterID := flags.MustGetDefinedString(FlagClusterID, cmd.Flags())
	if err := environments.Environment().Initialize(); err != nil {
		glog.Fatalf("Unable to initialize environment: %s", err.Error())
	}
	env := environments.Environment()
	ocmClient := customOcm.NewClient(env.Clients.OCM.Connection)
	clusterService := services.NewClusterService(env.DBFactory, ocmClient, env.Config.AWS)

	// scale down the machine pool
	machinePool, err := clusterService.ScaleDownMachinePool(clusterID)
	if err != nil {
		glog.Fatalf("Unable to scale down machine pool: %s", err.Error())
	}

	// print the output
	indentedPool := new(bytes.Buffer)
	if err := clustersmgmtv1.MarshalMachinePool(machinePool, indentedPool); err != nil {
		glog.Fatalf("Unable to marshal machine pool: %s", err.Error())
	}
	glog.V(10).Infof("%s", indentedPool.String())
}
