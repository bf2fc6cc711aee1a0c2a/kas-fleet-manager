package main

import (
	"flag"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/cloudprovider"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/cluster"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/kafka"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/migrate"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/observatorium"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/servecmd"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/serviceaccounts"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

//nolint
func init() {
}

func main() {
	// This is needed to make `glog` believe that the flags have already been parsed, otherwise
	// every log messages is prefixed by an error message stating the the flags haven't been
	// parsed.
	_ = flag.CommandLine.Parse([]string{})

	//pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	// Always log to stderr by default
	if err := flag.Set("logtostderr", "true"); err != nil {
		glog.Infof("Unable to set logtostderr to true")
	}

	rootCmd := &cobra.Command{
		Use:  "kas-fleet-manager",
		Long: "kas-fleet-manager serves as an example service template for new microservices",
	}

	// All subcommands under root
	migrateCmd := migrate.NewMigrateCommand()
	serveCmd := servecmd.NewServeCommand()
	clusterCmd := cluster.NewClusterCommand()
	observatoriumCmd := observatorium.NewRunObservatoriumCommand()
	serviceaccountCmd := serviceaccounts.NewServiceAccountCommand()
	errorsCmd := errors.NewErrorsCommand()

	// Add subcommand(s)
	rootCmd.AddCommand(migrateCmd, serveCmd, clusterCmd, kafka.NewKafkaCommand(), cloudprovider.NewCloudProviderCommand(), observatoriumCmd, serviceaccountCmd, errorsCmd)

	if err := rootCmd.Execute(); err != nil {
		glog.Fatalf("error running command: %v", err)
	}
}
