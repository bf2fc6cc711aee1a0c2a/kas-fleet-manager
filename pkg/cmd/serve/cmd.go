package serve

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/buildinformation"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

func NewServeCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve the kas-fleet-manager",
		Long:  "Serve the Kafka Service Fleet Manager.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			err := env.CreateServices()
			if err != nil {
				glog.Fatalf("Unable to initialize environment: %s", err.Error())
			}
			info, e := buildinformation.GetBuildInfo()
			if e != nil {
				glog.Fatalf("Unable to retrieve buildinfo: %s.", e.Error())
			}
			glog.Infof("GoVersion: %q. Commit time: %q. Architecture: %q. Operating System: %q. VCS Type: %q. CommitSha: %q.", info.GetGoVersion(), info.GetVCSTime(), info.GetArchitecture(), info.GetOperatingSystem(), info.GetVCSType(), info.GetCommitSHA())
		},
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
