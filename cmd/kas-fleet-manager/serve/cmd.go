package serve

import (
	"context"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
)

func NewServeCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve the kas-fleet-manager",
		Long:  "Serve the Kafka Service Fleet Manager.",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
			defer cancel()
			env.Run(ctx)
		},
	}
	return cmd
}
