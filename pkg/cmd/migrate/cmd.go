package migrate

import (
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
			env.MustInvoke(func(migrations []*db.Migration) {
				glog.Infoln("Migration starting")
				for _, migration := range migrations {
					migration.Migrate()
				}
				glog.Infoln("Migration done")
			})
		},
	}
	return cmd
}
