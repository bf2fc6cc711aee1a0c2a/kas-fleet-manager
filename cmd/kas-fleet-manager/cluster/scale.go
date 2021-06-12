package cluster

import (
	"encoding/json"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/flags"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

// ClusterNodeScaleIncrement - default increment/ decrement node count when scaling multiAZ clusters
const DefaultClusterNodeScaleIncrement = 3

// NewScaleCommand creates a new command for scaling Compute nodes in a OSD cluster
func NewScaleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scale",
		Short: "Scale the managed services Compute nodes in a OSD cluster",
		Long:  "Scale Compute nodes (up or down) in a OSD cluster.",
	}
	err := environments.Environment().AddFlags(cmd.PersistentFlags())
	if err != nil {
		glog.Fatalf("Unable to add environment flags to serve command: %s", err.Error())
	}
	cmd.AddCommand(NewScaleUpCommand(), NewScaleDownCommand())
	return cmd
}

// NewScaleUpCommand creates a new command for scaling up Compute nodes in a OSD cluster
func NewScaleUpCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Scale up a node",
		Long:  "Scale up Compute nodes in a OSD cluster.",
		Run:   runScaleUp,
	}
	cmd.Flags().String(FlagClusterID, "", "Cluster ID")
	return cmd
}

// NewScaleDownCommand creates a new command for scaling down Compute nodes in a OSD cluster
func NewScaleDownCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "down",
		Short: "Scale down a node",
		Long:  "Scale down Compute nodes in a OSD cluster.",
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
	clusterService := env.Services.Cluster

	// scale up compute nodes
	cluster, err := clusterService.ScaleUpComputeNodes(clusterID, DefaultClusterNodeScaleIncrement)
	if err != nil {
		glog.Fatalf("Unable to scale up compute nodes: %s", err.Error())
	}

	// print the output
	if indentedCluster, err := json.Marshal(cluster); err != nil {
		glog.Fatalf("Unable to marshal cluster: %s", err.Error())
	} else {
		glog.V(10).Infof("%s", string(indentedCluster))
	}
}

func runScaleDown(cmd *cobra.Command, _ []string) {
	clusterID := flags.MustGetDefinedString(FlagClusterID, cmd.Flags())
	if err := environments.Environment().Initialize(); err != nil {
		glog.Fatalf("Unable to initialize environment: %s", err.Error())
	}
	env := environments.Environment()
	clusterService := env.Services.Cluster

	// scale down compute nodes
	cluster, err := clusterService.ScaleDownComputeNodes(clusterID, DefaultClusterNodeScaleIncrement)
	if err != nil {
		glog.Fatalf("Unable to scale down compute nodes: %s", err.Error())
	}

	// print the outputs
	if indentedCluster, err := json.Marshal(cluster); err != nil {
		glog.Fatalf("Unable to marshal cluster: %s", err.Error())
	} else {
		glog.V(10).Infof("%s", string(indentedCluster))
	}
}
