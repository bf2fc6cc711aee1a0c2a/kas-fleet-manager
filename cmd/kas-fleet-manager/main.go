package main

import (
	"flag"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/migrate"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/serve"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
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

	env, err := environments.NewEnv(environments.GetEnvironmentStrFromEnv(),
		kafka.ConfigProviders().AsOption(),
		connector.ConfigProviders().AsOption(),
	)
	if err != nil {
		glog.Fatalf("error initializing: %v", err)
	}

	rootCmd := &cobra.Command{
		Use:  "kas-fleet-manager",
		Long: "kas-fleet-manager serves as an example service template for new microservices",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			err = env.LoadConfigAndCreateServices()
			if err != nil {
				glog.Fatalf("Unable to initialize environment: %s", err.Error())
			}
		},
	}

	err = env.AddFlags(rootCmd.PersistentFlags())
	if err != nil {
		glog.Fatalf("Unable to add environment flags: %s", err.Error())
	}

	err = env.ConfigContainer.Invoke(func(subcommands []*cobra.Command) {

		// All subcommands under root
		rootCmd.AddCommand(
			migrate.NewMigrateCommand(),
			serve.NewServeCommand(env),
		)
		rootCmd.AddCommand(subcommands...)

		if err := rootCmd.Execute(); err != nil {
			glog.Fatalf("error running command: %v", err)
		}

	})
}
