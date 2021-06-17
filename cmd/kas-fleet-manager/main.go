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
		kafka.ConfigProviders(),
		connector.ConfigProviders(),
	)
	if err != nil {
		glog.Fatalf("error initializing: %v", err)
	}
	defer env.Cleanup()

	rootCmd := &cobra.Command{
		Use:  "kas-fleet-manager",
		Long: "kas-fleet-manager serves as an example service template for new microservices",
	}

	err = env.AddFlags(rootCmd.PersistentFlags())
	if err != nil {
		glog.Fatalf("Unable to add global flags: %s", err.Error())
	}

	err = env.CreateServices()
	if err != nil {
		glog.Fatalf("Unable to initialize environment: %s", err.Error())
	}

	env.MustInvoke(func(subcommands []*cobra.Command) {

		// All subcommands under root
		rootCmd.AddCommand(
			migrate.NewMigrateCommand(env),
			serve.NewServeCommand(env),
		)
		rootCmd.AddCommand(subcommands...)

		if err := rootCmd.Execute(); err != nil {
			glog.Fatalf("error running command: %v", err)
		}

	})
	if err != nil {
		glog.Fatalf("Unable to initialize environment: %s", err.Error())
	}
}
