package migrate

import (
	"flag"

	"github.com/golang/glog"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/spf13/cobra"

	"gitlab.cee.redhat.com/service/sdb-ocm-example-service/pkg/config"
	"gitlab.cee.redhat.com/service/sdb-ocm-example-service/pkg/db"
)

var dbConfig = config.NewDatabaseConfig()

// migrate sub-command handles running migrations
func NewMigrateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run OCM Example service data migrations",
		Long:  "Run OCM Example service data migrations",
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
