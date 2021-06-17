// Package cluster contains commands for interacting with cluster logic of the service directly instead of through the
// REST API exposed via the serve command.
package cluster

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/spf13/cobra"
)

func NewClusterCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Perform managed-services-api cluster actions directly",
		Long:  "Perform managed-services-api cluster actions directly.",
	}

	// add sub-commands
	cmd.AddCommand(
		NewCreateCommand(env),
		NewScaleCommand(env),
	)

	return cmd
}
