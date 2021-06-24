package migrate

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/golang/glog"

	"github.com/spf13/cobra"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
)

// migrate sub-command handles running migrations
func NewMigrateCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run kas-fleet-manager data migrations",
		Long:  "Run Kafka Service Fleet Manager data migrations",
		Run: func(cmd *cobra.Command, args []string) {
			// we dont do a env.CreateServices()
			// to avoid requiring all other env settings to be provided.
			env.MustInvoke(func(dbConfig *config.DatabaseConfig) {
				err := dbConfig.ReadFiles()
				if err != nil {
					glog.Fatal(err)
				}
				dbFactory, cleanup := db.NewConnectionFactory(dbConfig)
				defer cleanup()
				db.Migrate(dbFactory)
			})
		},
	}
	return cmd
}
