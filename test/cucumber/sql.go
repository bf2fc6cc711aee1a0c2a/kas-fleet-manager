// Runes a SQL statement against the DB
//    When I run SQL "UPDATE connectors SET connector_type_id='foo' WHERE id = '${connector_id}';" expect 1 row to be affected.

package cucumber

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/cucumber/godog"
)

func init() {
	StepModules = append(StepModules, func(ctx *godog.ScenarioContext, s *TestScenario) {
		ctx.Step(`^I run SQL "([^"]*)" expect (\d+) row to be affected\.$`, s.iRunSQLExpectRowToBeAffected)
	})
}

func (s *TestScenario) iRunSQLExpectRowToBeAffected(sql string, expected int64) error {

	var err error
	sql, err = s.Expand(sql)
	if err != nil {
		return err
	}

	var dbFactory *db.ConnectionFactory
	err = s.Suite.Helper.Env.ServiceContainer.Resolve(&dbFactory)
	if err != nil {
		return err
	}
	gorm := dbFactory.New()
	exec := gorm.Exec(sql)
	if exec.Error != nil {
		return exec.Error
	}
	if exec.RowsAffected != expected {
		return fmt.Errorf("expected %d rows to be affected but %d were affected", expected, exec.RowsAffected)
	}
	return nil
}
