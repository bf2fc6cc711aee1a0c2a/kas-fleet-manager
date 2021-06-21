package serviceaccounts

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

func NewServiceAccountCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serviceaccount",
		Short: "Perform serviceaccount actions directly",
		Long:  "Perform serviceaccount actions directly.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			err := env.LoadConfigAndCreateServices()
			if err != nil {
				glog.Fatalf("Unable to initialize environment: %s", err.Error())
			}
		},
	}

	// add sub-commands
	cmd.AddCommand(
		NewCreateCommand(env),
		NewDeleteCommand(env),
		NewListCommand(env),
		NewResetCredsCommand(env),
	)

	return cmd
}
