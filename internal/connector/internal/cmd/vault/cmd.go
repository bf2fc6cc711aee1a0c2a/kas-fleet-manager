package vault

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/spf13/cobra"
)

func NewVaultCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vault",
		Short: "Manage vault secrets",
		Long:  "Manage vault secrets",
	}

	// add sub-commands
	cmd.AddCommand(NewListCommand(env))

	return cmd
}
