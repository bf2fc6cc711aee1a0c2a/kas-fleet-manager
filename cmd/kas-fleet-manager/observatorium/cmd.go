package observatorium

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

func NewRunObservatoriumCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "observatorium",
		Short: "Perform observatorium actions directly",
		Long:  "Perform observatorium actions directly.",
	}
	err := environments.Environment().AddFlags(cmd.PersistentFlags())
	if err != nil {
		glog.Fatalf("Unable to add environment flags to serve command: %s", err.Error())
	}

	// add sub-commands
	cmd.AddCommand(NewRunGetStateCommand())
	cmd.AddCommand(NewRunMetricsQueryRangeCommand())
	cmd.AddCommand(NewRunMetricsQueryCommand())
	return cmd
}
