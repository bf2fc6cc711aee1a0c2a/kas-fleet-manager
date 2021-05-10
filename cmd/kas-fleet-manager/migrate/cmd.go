package migrate

import (
	"flag"

	"github.com/golang/glog"
	"github.com/spf13/cobra"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
)

var dbConfig = config.NewDatabaseConfig()

// migrate sub-command handles running migrations
func NewMigrateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run kas-fleet-manager data migrations",
		Long:  "Run Kafka Service Fleet Manager data migrations",
		Run:   runMigrate,
	}

	dbConfig.AddFlags(cmd.PersistentFlags())
	cmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	return cmd
}

func runMigrate(cmd *cobra.Command, args []string) {
	err := dbConfig.ReadFiles()
	if err != nil {
		glog.Fatal(err)
	}

	db.Migrate(db.NewConnectionFactory(dbConfig))
}
