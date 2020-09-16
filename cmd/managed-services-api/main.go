package main

import (
	"flag"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/cluster"

	"github.com/golang/glog"
	"github.com/spf13/cobra"

	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/migrate"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/servecmd"
)

//nolint
//go:generate go-bindata -o ../../data/generated/openapi/openapi.go -pkg openapi -prefix ../../openapi/ ../../openapi
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
		Use:  "managed-services-api",
		Long: "managed-services-api serves as an example service template for new microservices",
	}

	// All subcommands under root
	migrateCmd := migrate.NewMigrateCommand()
	serveCmd := servecmd.NewServeCommand()
	clusterCmd := cluster.NewClusterCommand()

	// Add subcommand(s)
	rootCmd.AddCommand(migrateCmd, serveCmd, clusterCmd)

	if err := rootCmd.Execute(); err != nil {
		glog.Fatalf("error running command: %v", err)
	}
}
