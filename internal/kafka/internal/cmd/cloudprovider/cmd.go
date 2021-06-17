// Package cloudprovider contains commands for interacting with cloud provider service directly instead of through the
// REST API exposed via the serve command.

package cloudprovider

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/spf13/cobra"
)

func NewCloudProviderCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cloud_providers",
		Short: "Perform managed-services-api cloud providers actions directly",
		Long:  "Perform managed-services-api cloud providers actions directly.",
	}

	// add sub-commands
	cmd.AddCommand(
		NewProviderListCommand(env),
		NewRegionsListCommand(env),
	)

	return cmd
}
