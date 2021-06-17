package serve

import (
	"context"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
)

func NewServeCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve the kas-fleet-manager",
		Long:  "Serve the Kafka Service Fleet Manager.",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Cancel the context when we get a signal...
			ch := make(chan os.Signal, 1)
			signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				select {
				case <-ch:
					cancel()
				case <-ctx.Done():
				}
			}()

			env.Run(ctx)
		},
	}
	return cmd
}
