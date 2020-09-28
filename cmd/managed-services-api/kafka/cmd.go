// Package cluster contains commands for interacting with cluster logic of the service directly instead of through the
// REST API exposed via the serve command.
package kafka

import (
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/environments"
)

func NewKafkaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kafka",
		Short: "Perform managed-services-api kafka actions directly",
		Long:  "Perform managed-services-api kafka actions directly.",
	}
	err := environments.Environment().AddFlags(cmd.PersistentFlags())
	if err != nil {
		glog.Fatalf("Unable to add environment flags to serve command: %s", err.Error())
	}

	// add sub-commands
	cmd.AddCommand(NewCreateCommand(), NewGetCommand(), NewDeleteCommand())

	return cmd
}
