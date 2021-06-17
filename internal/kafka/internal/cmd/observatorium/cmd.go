package observatorium

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/spf13/cobra"
)

func NewRunObservatoriumCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "observatorium",
		Short: "Perform observatorium actions directly",
		Long:  "Perform observatorium actions directly.",
	}

	// add sub-commands
	cmd.AddCommand(
		NewRunGetStateCommand(env),
		NewRunMetricsQueryRangeCommand(env),
		NewRunMetricsQueryCommand(env),
	)
	return cmd
}
