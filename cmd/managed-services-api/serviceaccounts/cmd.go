package serviceaccounts

import (
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/environments"
)

func NewServiceAccountCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serviceaccount",
		Short: "Perform managed-services-api serviceaccount actions directly",
		Long:  "Perform managed-services-api serviceaccount actions directly.",
	}
	err := environments.Environment().AddFlags(cmd.PersistentFlags())
	if err != nil {
		glog.Fatalf("Unable to add environment flags to serve command: %s", err.Error())
	}

	// add sub-commands
	cmd.AddCommand(NewCreateCommand(), NewDeleteCommand(), NewListCommand(), NewResetCredsCommand())

	return cmd
}
