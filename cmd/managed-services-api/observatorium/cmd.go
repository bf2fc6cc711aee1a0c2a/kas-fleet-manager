package observatorium

import (
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/environments"
)

func NewRunObservatoriumCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "Perform managed-services-api metrics actions directly",
		Long:  "Perform managed-services-api metrics actions directly.",
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
