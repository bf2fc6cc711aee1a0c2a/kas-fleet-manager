// Package errors contains commands for inspecting the list of errors which can be returned by the service
package errors

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

func NewErrorsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "errors",
		Short: "Inspect the errors which can be returned by the service",
		Long:  "Inspect the errors which can be returned by the service",
	}
	err := environments.Environment().AddFlags(cmd.PersistentFlags())
	if err != nil {
		glog.Fatalf("Unable to add environment flags to errors command: %s", err.Error())
	}

	// add sub-commands
	cmd.AddCommand(NewListCommand())

	return cmd
}
