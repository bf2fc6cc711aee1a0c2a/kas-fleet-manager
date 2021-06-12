package serviceaccounts

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

func NewServiceAccountCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serviceaccount",
		Short: "Perform serviceaccount actions directly",
		Long:  "Perform serviceaccount actions directly.",
	}
	err := environments.Environment().AddFlags(cmd.PersistentFlags())
	if err != nil {
		glog.Fatalf("Unable to add environment flags to serve command: %s", err.Error())
	}

	// add sub-commands
	cmd.AddCommand(NewCreateCommand(), NewDeleteCommand(), NewListCommand(), NewResetCredsCommand())

	return cmd
}
