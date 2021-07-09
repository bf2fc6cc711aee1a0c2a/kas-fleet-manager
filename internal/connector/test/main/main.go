package main

import (
	"flag"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/providers/connector"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

func main() {
	// This is needed to make `glog` believe that the flags have already been parsed, otherwise
	// every log messages is prefixed by an error message stating the the flags haven't been
	// parsed.
	_ = flag.CommandLine.Parse([]string{})

	// Always log to stderr by default
	if err := flag.Set("logtostderr", "true"); err != nil {
		glog.Infof("Unable to set logtostderr to true")
	}

	env, err := environments.New(environments.GetEnvironmentStrFromEnv(), connector.ConfigProviders(false))
	if err != nil {
		glog.Fatalf("error initializing: %v", err)
	}
	defer env.Cleanup()

	rootCmd := &cobra.Command{
		Use:  "cos-fleet-manager",
		Long: "cos-fleet-manager implements the Connector Service Rest API",
	}

	err = env.AddFlags(rootCmd.PersistentFlags())
	if err != nil {
		glog.Fatalf("Unable to add global flags: %s", err.Error())
	}

	env.MustInvoke(func(subcommands []*cobra.Command) {

		// All subcommands under root
		rootCmd.AddCommand(subcommands...)

		if err := rootCmd.Execute(); err != nil {
			glog.Fatalf("error running command: %v", err)
		}

	})
	if err != nil {
		glog.Fatalf("Unable to initialize environment: %s", err.Error())
	}
}
